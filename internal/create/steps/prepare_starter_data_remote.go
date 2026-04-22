package steps

import (
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/gotcha190/toba/internal/create"
)

// prepareRemoteStarterData fetches starter data from the configured SSH host
// when no local backup folder is available.
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
func prepareRemoteStarterData(ctx *create.Context) (runErr error) {
	target, err := parseSSHTarget(ctx.Config.SSHTarget)
	if err != nil {
		return err
	}
	remoteWordPressRoot := strings.TrimSpace(ctx.Config.RemoteWordPressRoot)

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

	ctx.Logger.Info("Preparing starter files on SSH host " + ctx.Config.SSHTarget)
	sourceURL, err := captureSSHCommand(
		ctx,
		target,
		"",
		remoteStarterPreparationScript(remoteWordPressRoot, remoteDatabase, remotePlugins, remoteUploads),
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
	createdRemoteArtifacts = true

	sourceURL, err = normalizeSourceURL(sourceURL)
	if err != nil {
		return err
	}

	localDatabase := filepath.Join(tempDir, "database", pathBase(remoteDatabase))
	localPlugins := filepath.Join(tempDir, "plugins", pathBase(remotePlugins))
	localUploads := filepath.Join(tempDir, "uploads", pathBase(remoteUploads))

	ctx.Logger.Info("Downloading starter database over SSH")
	ctx.Logger.Info("Downloading starter plugins over SSH")
	ctx.Logger.Info("Downloading starter uploads over SSH")
	if err := downloadRemoteStarterFiles(ctx, target, []remoteStarterDownload{
		{
			name:       "database",
			remotePath: remoteDatabase,
			localPath:  localDatabase,
		},
		{
			name:       "plugins",
			remotePath: remotePlugins,
			localPath:  localPlugins,
		},
		{
			name:       "uploads",
			remotePath: remoteUploads,
			localPath:  localUploads,
		},
	}); err != nil {
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

type remoteStarterDownload struct {
	name       string
	remotePath string
	localPath  string
}

func downloadRemoteStarterFiles(ctx *create.Context, target sshTarget, downloads []remoteStarterDownload) error {
	var once sync.Once
	var runErr error
	var wg sync.WaitGroup

	for _, download := range downloads {
		download := download
		wg.Add(1)
		go func() {
			defer wg.Done()

			if err := copyRemoteFile(ctx, target, download.remotePath, download.localPath); err != nil {
				once.Do(func() {
					runErr = fmt.Errorf("failed to download starter %s over SSH: %w", download.name, err)
				})
			}
		}()
	}

	wg.Wait()
	return runErr
}

func remoteStarterPreparationScript(remoteWordPressRoot string, remoteDatabase string, remotePlugins string, remoteUploads string) string {
	return strings.Join([]string{
		"set -eu",
		"if [ ! -d " + shellQuote(remoteWordPressRoot) + " ]; then printf '%s\\n' " + shellQuote("__TOBA_REMOTE_ROOT_MISSING__") + "; exit 42; fi",
		"cleanup_on_error() { status=$?; if [ \"$status\" -ne 0 ]; then rm -f " + shellQuote(remoteDatabase) + " " + shellQuote(remotePlugins) + " " + shellQuote(remoteUploads) + "; fi; exit \"$status\"; }",
		"cleanup_on_signal() { rm -f " + shellQuote(remoteDatabase) + " " + shellQuote(remotePlugins) + " " + shellQuote(remoteUploads) + "; exit 130; }",
		"trap cleanup_on_error EXIT",
		"trap cleanup_on_signal HUP INT TERM",
		"cd " + shellQuote(remoteWordPressRoot),
		"source_url=$(wp84 option get home)",
		"wp84 db export " + shellQuote(pathBase(remoteDatabase)) + " >/dev/null",
		"cd wp-content",
		"zip -r -q ../" + shellQuote(pathBase(remotePlugins)) + " plugins",
		"zip -r -q -0 ../" + shellQuote(pathBase(remoteUploads)) + " . -i " + shellQuote("uploads/*"),
		"printf '%s\\n' \"$source_url\"",
	}, "; ")
}

func ensureRemoteWordPressRootExists(ctx *create.Context, target sshTarget, remoteWordPressRoot string) error {
	const missingMarker = "__TOBA_REMOTE_ROOT_MISSING__"

	script := "if [ -d " + shellQuote(remoteWordPressRoot) + " ]; then exit 0; fi; " +
		"printf '%s\\n' " + shellQuote(missingMarker) + "; exit 42"

	output, err := ctx.Runner.CaptureOutput("", "ssh", "-p", target.Port, target.UserHost, script)
	if err == nil {
		return nil
	}

	if strings.Contains(output, missingMarker) {
		return fmt.Errorf(
			"remote WordPress root %q does not exist on %s:%s; update TOBA_REMOTE_WORDPRESS_ROOT in the global config or pass --remote-wordpress-root",
			remoteWordPressRoot,
			target.UserHost,
			target.Port,
		)
	}

	detail := strings.TrimSpace(output)
	if detail == "" {
		return fmt.Errorf("failed to verify remote WordPress root %q on %s:%s: %w", remoteWordPressRoot, target.UserHost, target.Port, err)
	}

	return fmt.Errorf("failed to verify remote WordPress root %q on %s:%s: %w\n%s", remoteWordPressRoot, target.UserHost, target.Port, err, detail)
}

// normalizeSourceURL validates the captured remote site URL and returns a
// normalized string form.
//
// Parameters:
// - raw: raw URL string captured from the remote WordPress installation
//
// Returns:
// - a normalized URL string
// - an error when the URL is missing a scheme or host
func normalizeSourceURL(raw string) (string, error) {
	lines := strings.Split(raw, "\n")
	for i := len(lines) - 1; i >= 0; i-- {
		candidate := strings.TrimSpace(lines[i])
		if candidate == "" {
			continue
		}

		parsed, err := url.Parse(candidate)
		if err != nil {
			continue
		}
		if parsed.Scheme == "" || parsed.Host == "" {
			continue
		}

		return parsed.String(), nil
	}

	return "", fmt.Errorf("invalid remote WordPress home URL: %s", raw)
}
