package steps

import (
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gotcha190/toba/internal/create"
)

func prepareRemoteStarterData(ctx *create.Context) (runErr error) {
	target, err := parseSSHTarget(ctx.Config.SSHTarget)
	if err != nil {
		return err
	}

	ctx.Logger.Info("No local project backup folder found; using SSH starter data")

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
		ctx.Logger.Info("Would fetch starter data over SSH from " + ctx.Config.SSHTarget)
		return nil
	}

	tempDir, err := makeStarterTempDir()
	if err != nil {
		return err
	}
	ctx.StarterData.TempDir = tempDir

	prefix := fmt.Sprintf("toba-create-%d-%d", time.Now().Unix(), os.Getpid())
	remoteDatabase := filepath.Join(remoteWordPressRoot, prefix+".sql")
	remotePlugins := filepath.Join(remoteWordPressRoot, prefix+"-plugins.zip")
	remoteUploads := filepath.Join(remoteWordPressRoot, prefix+"-uploads.zip")
	createdRemoteArtifacts := false
	defer func() {
		if !createdRemoteArtifacts {
			return
		}

		cleanupErr := runSSHCommand(ctx, target, remoteWordPressRoot, "rm -f "+shellQuote(pathBase(remoteDatabase))+" "+shellQuote(pathBase(remotePlugins))+" "+shellQuote(pathBase(remoteUploads)))
		if cleanupErr == nil {
			return
		}

		ctx.Logger.Warning(
			"Failed to clean remote starter artifacts on " + target.UserHost + ":" + target.Port + ". " +
				"Remove manually if needed: " + pathBase(remoteDatabase) + ", " + pathBase(remotePlugins) + ", " + pathBase(remoteUploads) + ". Error: " + cleanupErr.Error(),
		)
	}()

	sourceURL, err := captureSSHCommand(ctx, target, remoteWordPressRoot, "wp84 option get home")
	if err != nil {
		return err
	}
	sourceURL, err = normalizeSourceURL(sourceURL)
	if err != nil {
		return err
	}

	ctx.Logger.Info("Preparing starter files on SSH host " + ctx.Config.SSHTarget)
	if err := runSSHCommand(ctx, target, remoteWordPressRoot, "wp84 db export "+shellQuote(pathBase(remoteDatabase))); err != nil {
		return err
	}
	createdRemoteArtifacts = true
	if err := runSSHCommand(ctx, target, filepath.Join(remoteWordPressRoot, "wp-content"), "zip -rq ../"+shellQuote(pathBase(remotePlugins))+" plugins"); err != nil {
		return err
	}
	if err := runSSHCommand(ctx, target, filepath.Join(remoteWordPressRoot, "wp-content"), "zip -rq ../"+shellQuote(pathBase(remoteUploads))+" uploads"); err != nil {
		return err
	}

	localDatabase := filepath.Join(tempDir, "database", pathBase(remoteDatabase))
	localPlugins := filepath.Join(tempDir, "plugins", pathBase(remotePlugins))
	localUploads := filepath.Join(tempDir, "uploads", pathBase(remoteUploads))

	ctx.Logger.Info("Downloading starter database over SSH")
	if err := copyRemoteFile(ctx, target, remoteDatabase, localDatabase); err != nil {
		return err
	}
	ctx.Logger.Info("Downloading starter plugins over SSH")
	if err := copyRemoteFile(ctx, target, remotePlugins, localPlugins); err != nil {
		return err
	}
	ctx.Logger.Info("Downloading starter uploads over SSH")
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
	return nil
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
