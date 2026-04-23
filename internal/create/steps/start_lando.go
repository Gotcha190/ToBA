package steps

import (
	"github.com/gotcha190/toba/internal/create"
	"github.com/gotcha190/toba/internal/lando"
)

type StartLandoStep struct{}

// NewStartLandoStep creates the pipeline step that starts the local Lando app.
//
// Returns:
// - a configured StartLandoStep instance
func NewStartLandoStep() *StartLandoStep {
	return &StartLandoStep{}
}

// Name returns the human-readable pipeline label for this step.
//
// Returns:
// - the display name used by pipeline logging
func (s *StartLandoStep) Name() string {
	return "Start Lando"
}

// Run starts the local Lando app unless the create run is in dry-run mode.
//
// Parameters:
// - ctx: shared create context containing project paths and runner access
//
// Returns:
// - an error when `lando start` fails
//
// Side effects:
// - runs `lando start` unless dry-run mode is enabled
func (s *StartLandoStep) Run(ctx *create.Context) error {
	if ctx.DryRun {
		ctx.Logger.Info("Would run: lando start")
		return nil
	}

	return lando.Start(ctx.Runner, ctx.Paths.Root)
}
