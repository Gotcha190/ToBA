package steps

import (
	"fmt"
	"os"
	"strings"

	"github.com/gotcha190/toba/internal/create"
)

const (
	starterDataModeLocal  = "local"
	starterDataModeRemote = "remote"
)

type PrepareStarterDataStep struct{}

// NewPrepareStarterDataStep creates the pipeline step that selects and
// prepares the starter data source for the run.
//
// Parameters:
// - none
//
// Returns:
// - a configured PrepareStarterDataStep instance
func NewPrepareStarterDataStep() *PrepareStarterDataStep {
	return &PrepareStarterDataStep{}
}

// Name returns the human-readable pipeline label for this step.
//
// Parameters:
// - none
//
// Returns:
// - the display name used by pipeline logging
func (s *PrepareStarterDataStep) Name() string {
	return "Prepare starter data"
}

// Run chooses between a local backup folder and the SSH fallback source and
// populates ctx.StarterData accordingly.
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
func (s *PrepareStarterDataStep) Run(ctx *create.Context) error {
	rootInfo, err := os.Stat(ctx.Paths.Root)
	switch {
	case err == nil && rootInfo.IsDir():
		return prepareLocalStarterData(ctx)
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
		return prepareRemoteStarterData(ctx)
	}
}
