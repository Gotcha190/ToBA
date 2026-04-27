package steps

import (
	"github.com/gotcha190/toba/internal/create"
	"github.com/gotcha190/toba/internal/create/sourcedata"
)

const (
	starterDataModeLocal  = sourcedata.ModeLocal
	starterDataModeRemote = sourcedata.ModeRemote
)

type PrepareStarterDataStep struct{}

// NewPrepareStarterDataStep creates the pipeline step that selects and
// prepares the starter data source for the run.
//
// Returns:
// - a configured PrepareStarterDataStep instance
func NewPrepareStarterDataStep() *PrepareStarterDataStep {
	return &PrepareStarterDataStep{}
}

// Name returns the human-readable pipeline label for this step.
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
	return sourcedata.Prepare(ctx)
}
