package sourcedata

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/gotcha190/toba/internal/create"
)

type remoteArtifacts struct {
	remoteWordPressRoot    string
	remoteDatabase         string
	remotePlugins          string
	remoteUploads          string
	remoteSourceURL        string
	localDatabase          string
	localPlugins           string
	localUploads           string
	createdRemoteArtifacts bool
}

// newRemoteArtifacts builds the local and remote paths for one SSH starter-data run.
//
// Returns:
// - the remote artifact set used during preparation and download
func newRemoteArtifacts(tempDir string, remoteWordPressRoot string) remoteArtifacts {
	runPrefix := fmt.Sprintf("toba-create-%d-%d", time.Now().Unix(), os.Getpid())
	remoteDatabase := filepath.Join(remoteWordPressRoot, runPrefix+".sql")
	remotePlugins := filepath.Join(remoteWordPressRoot, runPrefix+"-plugins.zip")
	remoteUploads := filepath.Join(remoteWordPressRoot, runPrefix+"-uploads.zip")
	remoteSourceURL := filepath.Join(remoteWordPressRoot, runPrefix+"-home.txt")

	return remoteArtifacts{
		remoteWordPressRoot: remoteWordPressRoot,
		remoteDatabase:      remoteDatabase,
		remotePlugins:       remotePlugins,
		remoteUploads:       remoteUploads,
		remoteSourceURL:     remoteSourceURL,
		localDatabase:       filepath.Join(tempDir, "database", pathBase(remoteDatabase)),
		localPlugins:        filepath.Join(tempDir, "plugins", pathBase(remotePlugins)),
		localUploads:        filepath.Join(tempDir, "uploads", pathBase(remoteUploads)),
	}
}

// downloads returns the remote files that must be copied back locally.
//
// Returns:
// - the remote download definitions for database, plugins, and uploads
func (a remoteArtifacts) downloads() []remoteDownload {
	return []remoteDownload{
		{name: "database", remotePath: a.remoteDatabase, localPath: a.localDatabase},
		{name: "plugins", remotePath: a.remotePlugins, localPath: a.localPlugins},
		{name: "uploads", remotePath: a.remoteUploads, localPath: a.localUploads},
	}
}

// cleanupRemoteArtifacts removes prepared files from the remote host after use.
//
// Returns:
// - null
//
// Side effects:
// - may run a remote cleanup command
// - may log a warning when cleanup fails
func cleanupRemoteArtifacts(ctx *create.Context, target sshTarget, artifacts *remoteArtifacts) {
	if artifacts == nil || !artifacts.createdRemoteArtifacts {
		return
	}

	cleanupErr := runSSHCommand(
		ctx,
		target,
		artifacts.remoteWordPressRoot,
		"rm -f "+shellQuote(pathBase(artifacts.remoteDatabase))+
			" "+shellQuote(pathBase(artifacts.remotePlugins))+
			" "+shellQuote(pathBase(artifacts.remoteUploads))+
			" "+shellQuote(pathBase(artifacts.remoteSourceURL)),
	)
	if cleanupErr == nil {
		return
	}

	ctx.Logger.Warning(
		"Failed to clean remote starter artifacts on " + target.UserHost + ":" + target.Port + ". " +
			"Remove manually if needed: " + pathBase(artifacts.remoteDatabase) + ", " + pathBase(artifacts.remotePlugins) + ", " + pathBase(artifacts.remoteUploads) + ", " + pathBase(artifacts.remoteSourceURL) + ". Error: " + cleanupErr.Error(),
	)
}
