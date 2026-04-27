package sourcedata

import (
	"fmt"
	"os"
	"strings"

	"github.com/gotcha190/toba/internal/create"
)

const (
	ModeLocal  = "local"
	ModeRemote = "remote"
)

// Prepare chooses between a local backup folder and the SSH fallback source
// and populates ctx.StarterData accordingly.
//
// Parameters:
// - ctx: shared create context containing project paths, config, and runtime state
//
// Returns:
// - an error when local starter data is invalid or SSH starter data cannot be configured or fetched
//
// Side effects:
// - mutates ctx.StarterData and ctx.UseExistingProjectDir
// - reads the filesystem to detect an existing project directory
func Prepare(ctx *create.Context) error {
	if ctx.StarterData.Mode == ModeRemote {
		return prepareRemote(ctx)
	}

	rootInfo, err := os.Stat(ctx.Paths.Root)
	switch {
	case err == nil && rootInfo.IsDir():
		return prepareLocal(ctx)
	case err == nil:
		return fmt.Errorf("project path exists and is not a directory: %s", ctx.Paths.Root)
	case !os.IsNotExist(err):
		return err
	default:
		if strings.TrimSpace(ctx.Config.SSHTarget) == "" {
			globalEnvPath, pathErr := create.GlobalEnvPath()
			if pathErr != nil {
				return fmt.Errorf("SSH starter source is not configured; set TOBA_SSH_TARGET in the global config or pass --ssh-target")
			}
			return fmt.Errorf("SSH starter source is not configured; fill in TOBA_SSH_TARGET in %s or pass --ssh-target", globalEnvPath)
		}
		if strings.TrimSpace(ctx.Config.RemoteWordPressRoot) == "" {
			globalEnvPath, pathErr := create.GlobalEnvPath()
			if pathErr != nil {
				return fmt.Errorf("SSH starter source is missing the remote WordPress root; set TOBA_REMOTE_WORDPRESS_ROOT in the global config or pass --remote-wordpress-root")
			}
			return fmt.Errorf("SSH starter source is missing the remote WordPress root; fill in TOBA_REMOTE_WORDPRESS_ROOT in %s or pass --remote-wordpress-root", globalEnvPath)
		}
		return prepareRemote(ctx)
	}
}
