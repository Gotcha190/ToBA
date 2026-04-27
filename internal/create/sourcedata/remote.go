package sourcedata

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/gotcha190/toba/internal/create"
)

// prepareRemote fetches starter data from the configured SSH host when no
// local backup folder is available.
//
// Parameters:
// - ctx: shared create context containing SSH config, starter-data state, and runner access
//
// Returns:
// - an error when the SSH target is invalid or starter assets cannot be prepared or downloaded
//
// Side effects:
// - creates a temporary working directory
// - runs remote SSH commands and SCP downloads
// - populates ctx.StarterData with downloaded asset paths
func prepareRemote(ctx *create.Context) error {
	target, err := parseSSHTarget(ctx.Config.SSHTarget)
	if err != nil {
		return err
	}
	remoteWordPressRoot := strings.TrimSpace(ctx.Config.RemoteWordPressRoot)

	ctx.Logger.Info("No local project backup folder found; using SSH starter data")

	if ctx.DryRun {
		tempDir := filepath.Join(os.TempDir(), "toba-starter-dry-run")
		ctx.StarterData = create.StarterData{
			Mode:         ModeRemote,
			TempDir:      tempDir,
			DatabasePath: filepath.Join(tempDir, "remote", "starter.sql"),
			PluginsPaths: []string{filepath.Join(tempDir, "plugins", "starter-plugins.zip")},
			UploadsPaths: []string{filepath.Join(tempDir, "uploads", "starter-uploads.zip")},
			SourceURL:    "https://remote.example.test",
		}
		ctx.Logger.Info("Would fetch starter data over SSH from " + ctx.Config.SSHTarget)
		return nil
	}

	tempDir, err := makeTempDir()
	if err != nil {
		return err
	}
	ctx.StarterData.TempDir = tempDir

	artifacts := newRemoteArtifacts(tempDir, remoteWordPressRoot)
	defer func() {
		cleanupRemoteArtifacts(ctx, target, &artifacts)
	}()

	ctx.Logger.Info("Preparing starter files on SSH host " + ctx.Config.SSHTarget)
	sourceURL, err := captureSSHCommand(
		ctx,
		target,
		"",
		remotePreparationScript(
			artifacts.remoteWordPressRoot,
			artifacts.remoteDatabase,
			artifacts.remotePlugins,
			artifacts.remoteUploads,
			artifacts.remoteSourceURL,
		),
	)
	if err != nil {
		if strings.Contains(err.Error(), "__TOBA_REMOTE_ROOT_MISSING__") {
			return fmt.Errorf(
				"remote WordPress root %q does not exist on %s:%s; update TOBA_REMOTE_WORDPRESS_ROOT in the global config or pass --remote-wordpress-root",
				remoteWordPressRoot,
				target.UserHost,
				target.Port,
			)
		}
		return fmt.Errorf("failed to prepare remote starter files on %s:%s: %w", target.UserHost, target.Port, err)
	}

	artifacts.createdRemoteArtifacts = true

	sourceURL, err = normalizeSourceURL(sourceURL)
	if err != nil {
		return err
	}

	ctx.Logger.Info("Downloading starter database over SSH")
	ctx.Logger.Info("Downloading starter plugins over SSH")
	ctx.Logger.Info("Downloading starter uploads over SSH")
	if err := downloadRemoteFiles(ctx, target, artifacts.downloads()); err != nil {
		return err
	}

	ctx.StarterData = create.StarterData{
		Mode:         ModeRemote,
		TempDir:      tempDir,
		DatabasePath: artifacts.localDatabase,
		PluginsPaths: []string{artifacts.localPlugins},
		UploadsPaths: []string{artifacts.localUploads},
		SourceURL:    sourceURL,
	}
	return nil
}
