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
	"github.com/gotcha190/ToBA/internal/templates"
)

const (
	starterDataModeEmbedded = "embedded"
	starterDataModeRemote   = "remote"
)

const (
	remoteWordPressRoot = "www/toba.tamago-dev.pl" // TODO: derive "toba" from the starter address instead of hardcoding it.
)

var (
	sshTargetPattern      = regexp.MustCompile(`^([^\s@]+@[^\s]+)\s+-p\s+([0-9]+)$`)
	starterTemplateFiles  = templates.WordPressBackupFiles
	starterTemplateReader = templates.Read
)

type PrepareStarterDataStep struct{}

type sshTarget struct {
	UserHost string
	Port     string
}

type starterSelection struct {
	Database string
	Plugins  []string
	Uploads  []string
	Others   []string
	Themes   []string
}

func NewPrepareStarterDataStep() *PrepareStarterDataStep {
	return &PrepareStarterDataStep{}
}

func (s *PrepareStarterDataStep) Name() string {
	return "Prepare starter data"
}

func (s *PrepareStarterDataStep) Run(ctx *create.Context) (runErr error) {
	selection, err := detectStarterSelection()
	if err != nil {
		return err
	}

	if selection.Database != "" {
		ctx.Logger.Info("Using embedded starter override prepared by 'toba update'")
		return prepareEmbeddedStarterData(ctx, selection)
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
			DatabasePath: filepath.Join(tempDir, "starter.sql"),
			PluginsPaths: []string{filepath.Join(tempDir, "starter-plugins.zip")},
			UploadsPaths: []string{filepath.Join(tempDir, "starter-uploads.zip")},
			SourceURL:    "https://remote.example.test",
		}
		if len(selection.Others) > 0 {
			ctx.StarterData.OthersPaths = tempPathsFromSelection(tempDir, selection.Others)
		}
		if len(selection.Themes) > 0 {
			ctx.StarterData.ThemePaths = tempPathsFromSelection(tempDir, selection.Themes)
		}
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

	localDatabase := filepath.Join(tempDir, pathBase(remoteDatabase))
	localPlugins := filepath.Join(tempDir, pathBase(remotePlugins))
	localUploads := filepath.Join(tempDir, pathBase(remoteUploads))

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
	if len(selection.Others) > 0 {
		othersPaths, err := copyEmbeddedTemplatesToTemp(tempDir, selection.Others)
		if err != nil {
			return err
		}
		ctx.StarterData.OthersPaths = othersPaths
	}
	if len(selection.Themes) > 0 {
		themePaths, err := copyEmbeddedTemplatesToTemp(tempDir, selection.Themes)
		if err != nil {
			return err
		}
		ctx.StarterData.ThemePaths = themePaths
	}
	return nil
}

func detectStarterSelection() (starterSelection, error) {
	databaseFiles, err := collectStarterFiles("database", ".sql", ".gz")
	if err != nil {
		return starterSelection{}, err
	}
	pluginsFiles, err := collectStarterFiles("plugins", ".zip")
	if err != nil {
		return starterSelection{}, err
	}
	uploadsFiles, err := collectStarterFiles("uploads", ".zip")
	if err != nil {
		return starterSelection{}, err
	}
	othersFiles, err := collectStarterFiles("others", ".zip")
	if err != nil {
		return starterSelection{}, err
	}
	themeFiles, err := collectStarterFiles("themes", ".zip")
	if err != nil {
		return starterSelection{}, err
	}

	selection := starterSelection{}
	corePresent := 0
	if len(databaseFiles) > 0 {
		corePresent++
	}
	if len(pluginsFiles) > 0 {
		corePresent++
	}
	if len(uploadsFiles) > 0 {
		corePresent++
	}

	if len(databaseFiles) > 1 {
		return starterSelection{}, fmt.Errorf("expected exactly 1 database override in wordpress/database, found %d", len(databaseFiles))
	}
	if corePresent > 0 && corePresent < 3 {
		var missing []string
		if len(databaseFiles) == 0 {
			missing = append(missing, "database")
		}
		if len(pluginsFiles) == 0 {
			missing = append(missing, "plugins")
		}
		if len(uploadsFiles) == 0 {
			missing = append(missing, "uploads")
		}

		return starterSelection{}, fmt.Errorf(
			"incomplete embedded starter override; missing %s. TODO: fetch only missing categories from SSH in a future iteration",
			strings.Join(missing, ", "),
		)
	}

	if len(databaseFiles) == 1 {
		selection.Database = databaseFiles[0]
	}
	if len(pluginsFiles) > 0 {
		selection.Plugins = append([]string(nil), pluginsFiles...)
	}
	if len(uploadsFiles) > 0 {
		selection.Uploads = append([]string(nil), uploadsFiles...)
	}
	if len(othersFiles) > 0 {
		selection.Others = append([]string(nil), othersFiles...)
	}
	if len(themeFiles) > 0 {
		selection.Themes = append([]string(nil), themeFiles...)
	}

	return selection, nil
}

func prepareEmbeddedStarterData(ctx *create.Context, selection starterSelection) error {
	if ctx.DryRun {
		tempDir := filepath.Join(os.TempDir(), "toba-starter-dry-run")
		ctx.StarterData = create.StarterData{
			Mode:         starterDataModeEmbedded,
			TempDir:      tempDir,
			DatabasePath: filepath.Join(tempDir, filepath.Base(selection.Database)),
			PluginsPaths: tempPathsFromSelection(tempDir, selection.Plugins),
			UploadsPaths: tempPathsFromSelection(tempDir, selection.Uploads),
		}
		if len(selection.Others) > 0 {
			ctx.StarterData.OthersPaths = tempPathsFromSelection(tempDir, selection.Others)
		}
		if len(selection.Themes) > 0 {
			ctx.StarterData.ThemePaths = tempPathsFromSelection(tempDir, selection.Themes)
		}
		return nil
	}

	tempDir, err := os.MkdirTemp("", "toba-starter-*")
	if err != nil {
		return err
	}
	ctx.StarterData.TempDir = tempDir

	databasePath, err := copyEmbeddedTemplateToTemp(tempDir, selection.Database)
	if err != nil {
		return err
	}
	pluginsPaths, err := copyEmbeddedTemplatesToTemp(tempDir, selection.Plugins)
	if err != nil {
		return err
	}
	uploadsPaths, err := copyEmbeddedTemplatesToTemp(tempDir, selection.Uploads)
	if err != nil {
		return err
	}

	ctx.StarterData = create.StarterData{
		Mode:         starterDataModeEmbedded,
		TempDir:      tempDir,
		DatabasePath: databasePath,
		PluginsPaths: pluginsPaths,
		UploadsPaths: uploadsPaths,
	}
	if len(selection.Others) > 0 {
		othersPaths, err := copyEmbeddedTemplatesToTemp(tempDir, selection.Others)
		if err != nil {
			return err
		}
		ctx.StarterData.OthersPaths = othersPaths
	}
	if len(selection.Themes) > 0 {
		themePaths, err := copyEmbeddedTemplatesToTemp(tempDir, selection.Themes)
		if err != nil {
			return err
		}
		ctx.StarterData.ThemePaths = themePaths
	}

	return nil
}

func collectStarterFiles(category string, suffixes ...string) ([]string, error) {
	var matches []string

	for _, suffix := range suffixes {
		files, err := starterTemplateFiles(category, suffix)
		if err != nil {
			return nil, err
		}
		matches = append(matches, files...)
	}

	return matches, nil
}

func copyEmbeddedTemplateToTemp(tempDir string, templatePath string) (string, error) {
	content, err := starterTemplateReader(templatePath)
	if err != nil {
		return "", err
	}

	targetPath := filepath.Join(tempDir, filepath.Base(templatePath))
	if err := os.WriteFile(targetPath, content, 0644); err != nil {
		return "", err
	}

	return targetPath, nil
}

func copyEmbeddedTemplatesToTemp(tempDir string, templatePaths []string) ([]string, error) {
	targetPaths := make([]string, 0, len(templatePaths))
	for _, templatePath := range templatePaths {
		targetPath, err := copyEmbeddedTemplateToTemp(tempDir, templatePath)
		if err != nil {
			return nil, err
		}
		targetPaths = append(targetPaths, targetPath)
	}

	return targetPaths, nil
}

func tempPathsFromSelection(tempDir string, templatePaths []string) []string {
	targetPaths := make([]string, 0, len(templatePaths))
	for _, templatePath := range templatePaths {
		targetPaths = append(targetPaths, filepath.Join(tempDir, filepath.Base(templatePath)))
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
