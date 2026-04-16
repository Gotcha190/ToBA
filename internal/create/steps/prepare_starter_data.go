package steps

import (
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/gotcha190/ToBA/internal/create"
	"github.com/gotcha190/ToBA/internal/project"
	"github.com/gotcha190/ToBA/internal/updraft"
)

const (
	starterDataModeLocal  = "local"
	starterDataModeRemote = "remote"
)

const (
	remoteWordPressRoot = "www/toba.tamago-dev.pl" // TODO: derive "toba" from the starter address instead of hardcoding it.
)

var (
	sshTargetPattern = regexp.MustCompile(`^([^\s@]+@[^\s]+)\s+-p\s+([0-9]+)$`)
)

type PrepareStarterDataStep struct{}

type sshTarget struct {
	UserHost string
	Port     string
}

func NewPrepareStarterDataStep() *PrepareStarterDataStep {
	return &PrepareStarterDataStep{}
}

func (s *PrepareStarterDataStep) Name() string {
	return "Prepare starter data"
}

func (s *PrepareStarterDataStep) Run(ctx *create.Context) (runErr error) {
	rootInfo, err := os.Stat(ctx.Paths.Root)
	switch {
	case err == nil && rootInfo.IsDir():
		return prepareLocalProjectStarterData(ctx)
	case err == nil:
		return fmt.Errorf("project path exists and is not a directory: %s", ctx.Paths.Root)
	case !os.IsNotExist(err):
		return err
	}

	target, err := parseSSHTarget(ctx.Config.SSHTarget)
	if err != nil {
		return err
	}

	if ctx.DryRun {
		tempDir := filepath.Join(os.TempDir(), "toba-starter-dry-run")
		ctx.StarterData = create.StarterData{
			Mode:         starterDataModeRemote,
			TempDir:      tempDir,
			DatabasePath: filepath.Join(tempDir, "remote", "starter.sql"),
			PluginsPaths: []string{filepath.Join(tempDir, "plugins", "starter-plugins.zip")},
			UploadsPaths: []string{filepath.Join(tempDir, "uploads", "starter-uploads.zip")},
			SourceURL:    "https://remote.example.test",
		}
		ctx.Logger.Info("No local project backup folder found; using SSH starter data")
		ctx.Logger.Info("Would fetch starter data over SSH from " + ctx.Config.SSHTarget)
		return nil
	}

	tempDir, err := os.MkdirTemp("", "toba-starter-*")
	if err != nil {
		return err
	}
	ctx.StarterData.TempDir = tempDir

	prefix := fmt.Sprintf("toba-create-%d-%d", time.Now().Unix(), os.Getpid())
	remoteDatabase := filepath.Join(remoteWordPressRoot, prefix+".sql")
	remotePlugins := filepath.Join(remoteWordPressRoot, prefix+"-plugins.zip")
	remoteUploads := filepath.Join(remoteWordPressRoot, prefix+"-uploads.zip")
	defer func() {
		cleanupErr := runSSHCommand(ctx, target, remoteWordPressRoot, "rm -f "+shellQuote(pathBase(remoteDatabase))+" "+shellQuote(pathBase(remotePlugins))+" "+shellQuote(pathBase(remoteUploads)))
		if cleanupErr == nil {
			return
		}

		ctx.Logger.Warning("Failed to clean remote starter artifacts: " + cleanupErr.Error())
	}()

	sourceURL, err := captureSSHCommand(ctx, target, remoteWordPressRoot, "wp84 option get home")
	if err != nil {
		return err
	}
	sourceURL, err = normalizeSourceURL(sourceURL)
	if err != nil {
		return err
	}

	if err := runSSHCommand(ctx, target, remoteWordPressRoot, "wp84 db export "+shellQuote(pathBase(remoteDatabase))); err != nil {
		return err
	}
	if err := runSSHCommand(ctx, target, filepath.Join(remoteWordPressRoot, "wp-content"), "zip -rq ../"+shellQuote(pathBase(remotePlugins))+" plugins"); err != nil {
		return err
	}
	if err := runSSHCommand(ctx, target, filepath.Join(remoteWordPressRoot, "wp-content"), "zip -rq ../"+shellQuote(pathBase(remoteUploads))+" uploads"); err != nil {
		return err
	}

	localDatabase := filepath.Join(tempDir, "database", pathBase(remoteDatabase))
	localPlugins := filepath.Join(tempDir, "plugins", pathBase(remotePlugins))
	localUploads := filepath.Join(tempDir, "uploads", pathBase(remoteUploads))

	if err := copyRemoteFile(ctx, target, remoteDatabase, localDatabase); err != nil {
		return err
	}
	if err := copyRemoteFile(ctx, target, remotePlugins, localPlugins); err != nil {
		return err
	}
	if err := copyRemoteFile(ctx, target, remoteUploads, localUploads); err != nil {
		return err
	}

	ctx.StarterData = create.StarterData{
		Mode:         starterDataModeRemote,
		TempDir:      tempDir,
		DatabasePath: localDatabase,
		PluginsPaths: []string{localPlugins},
		UploadsPaths: []string{localUploads},
		SourceURL:    sourceURL,
	}
	ctx.Logger.Info("No local project backup folder found; using SSH starter data")
	return nil
}

func prepareLocalProjectStarterData(ctx *create.Context) error {
	selection, err := updraft.ScanProjectDir(ctx.Paths.Root)
	if err != nil {
		return fmt.Errorf("local project backup in %s is invalid: %w", ctx.Paths.Root, err)
	}
	if !selection.HasRecognizedFiles() {
		return fmt.Errorf("project directory %s exists but contains no recognizable Updraft backup files", ctx.Paths.Root)
	}
	if err := selection.ValidateLocalProjectSet(); err != nil {
		return fmt.Errorf("local project backup in %s is incomplete: %w", ctx.Paths.Root, err)
	}

	ctx.UseExistingProjectDir = true
	ctx.Logger.Info("Using local project backup folder: " + ctx.Paths.Root)

	if ctx.DryRun {
		tempDir := filepath.Join(os.TempDir(), "toba-starter-dry-run")
		ctx.StarterData = create.StarterData{
			Mode:         starterDataModeLocal,
			TempDir:      tempDir,
			DatabasePath: filepath.Join(tempDir, "database", filepath.Base(selection.Database)),
			PluginsPaths: tempPathsFromSources(tempDir, "plugins", selection.Plugins),
			UploadsPaths: tempPathsFromSources(tempDir, "uploads", selection.Uploads),
			OthersPaths:  tempPathsFromSources(tempDir, "others", selection.Others),
			ThemePaths:   tempPathsFromSources(tempDir, "themes", selection.Themes),
		}
		return nil
	}

	tempDir, err := os.MkdirTemp("", "toba-starter-*")
	if err != nil {
		return err
	}

	databasePath, err := copyLocalFileToTemp(tempDir, "database", selection.Database)
	if err != nil {
		return err
	}
	pluginsPaths, err := copyLocalFilesToTemp(tempDir, "plugins", selection.Plugins)
	if err != nil {
		return err
	}
	uploadsPaths, err := copyLocalFilesToTemp(tempDir, "uploads", selection.Uploads)
	if err != nil {
		return err
	}
	othersPaths, err := copyLocalFilesToTemp(tempDir, "others", selection.Others)
	if err != nil {
		return err
	}
	themePaths, err := copyLocalFilesToTemp(tempDir, "themes", selection.Themes)
	if err != nil {
		return err
	}

	ctx.StarterData = create.StarterData{
		Mode:         starterDataModeLocal,
		TempDir:      tempDir,
		DatabasePath: databasePath,
		PluginsPaths: pluginsPaths,
		UploadsPaths: uploadsPaths,
		OthersPaths:  othersPaths,
		ThemePaths:   themePaths,
	}

	return nil
}

func copyLocalFileToTemp(tempDir string, category string, sourcePath string) (string, error) {
	if sourcePath == "" {
		return "", nil
	}

	targetPath := filepath.Join(tempDir, category, filepath.Base(sourcePath))
	if err := project.CopyFile(sourcePath, targetPath); err != nil {
		return "", err
	}

	return targetPath, nil
}

func copyLocalFilesToTemp(tempDir string, category string, sourcePaths []string) ([]string, error) {
	targetPaths := make([]string, 0, len(sourcePaths))
	for _, sourcePath := range sourcePaths {
		targetPath, err := copyLocalFileToTemp(tempDir, category, sourcePath)
		if err != nil {
			return nil, err
		}
		targetPaths = append(targetPaths, targetPath)
	}

	return targetPaths, nil
}

func tempPathsFromSources(tempDir string, category string, sourcePaths []string) []string {
	targetPaths := make([]string, 0, len(sourcePaths))
	for _, sourcePath := range sourcePaths {
		targetPaths = append(targetPaths, filepath.Join(tempDir, category, filepath.Base(sourcePath)))
	}

	return targetPaths
}

func parseSSHTarget(raw string) (sshTarget, error) {
	trimmed := strings.TrimSpace(raw)
	match := sshTargetPattern.FindStringSubmatch(trimmed)
	if match == nil {
		return sshTarget{}, fmt.Errorf("invalid TOBA_SSH_TARGET %q; expected format: user@host -p port (example: toba@185.238.75.243 -p 22666)", raw)
	}

	if _, err := strconv.Atoi(match[2]); err != nil {
		return sshTarget{}, fmt.Errorf("invalid TOBA_SSH_TARGET %q; expected numeric port in format: user@host -p port", raw)
	}

	return sshTarget{
		UserHost: match[1],
		Port:     match[2],
	}, nil
}

func runSSHCommand(ctx *create.Context, target sshTarget, remoteDir string, script string) error {
	return ctx.Runner.Run("", "ssh", "-p", target.Port, target.UserHost, remoteScript(remoteDir, script))
}

func captureSSHCommand(ctx *create.Context, target sshTarget, remoteDir string, script string) (string, error) {
	output, err := ctx.Runner.CaptureOutput("", "ssh", "-p", target.Port, target.UserHost, remoteScript(remoteDir, script))
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(output), nil
}

func copyRemoteFile(ctx *create.Context, target sshTarget, remotePath string, localPath string) error {
	if err := os.MkdirAll(filepath.Dir(localPath), 0755); err != nil {
		return err
	}

	return ctx.Runner.Run("", "scp", "-P", target.Port, target.UserHost+":"+remotePath, localPath)
}

func remoteScript(remoteDir string, script string) string {
	return "cd " + shellQuote(remoteDir) + " && " + script
}

func shellQuote(value string) string {
	return "'" + strings.ReplaceAll(value, "'", `'\''`) + "'"
}

func normalizeSourceURL(raw string) (string, error) {
	trimmed := strings.TrimSpace(raw)
	parsed, err := url.Parse(trimmed)
	if err != nil {
		return "", err
	}
	if parsed.Scheme == "" || parsed.Host == "" {
		return "", fmt.Errorf("invalid remote WordPress home URL: %s", raw)
	}

	return parsed.String(), nil
}

func pathBase(path string) string {
	return filepath.Base(path)
}
