package steps

import (
	"github.com/gotcha190/toba/internal/create"
	"github.com/gotcha190/toba/internal/theme"
)

type GenerateAcornKeyStep struct{}

// NewGenerateAcornKeyStep creates the pipeline step that generates the Acorn
// key for starter-repo based themes.
//
// Parameters:
// - none
//
// Returns:
// - a configured GenerateAcornKeyStep instance
func NewGenerateAcornKeyStep() *GenerateAcornKeyStep {
	return &GenerateAcornKeyStep{}
}

// Name returns the human-readable pipeline label for this step.
//
// Parameters:
// - none
//
// Returns:
// - the display name used by pipeline logging
func (s *GenerateAcornKeyStep) Name() string {
	return "Generate Acorn key"
}

// Run generates the Acorn key unless the run is restoring a local theme
// backup that already contains theme data.
//
// Parameters:
// - ctx: shared create context containing starter mode, project paths, and runner access
//
// Returns:
// - an error when Acorn key generation fails
//
// Side effects:
// - runs `lando wp acorn key:generate` twice unless dry-run mode is enabled
// - logs a skip message for local theme backup mode
func (s *GenerateAcornKeyStep) Run(ctx *create.Context) error {
	if len(ctx.StarterData.ThemePaths) > 0 {
		ctx.Logger.Info("Skipping Acorn key generation: using local theme backup")
		return nil
	}

	if ctx.DryRun {
		ctx.Logger.Info("Would run: lando wp acorn key:generate")
		ctx.Logger.Info("Would run: lando wp acorn key:generate")
		return nil
	}

	return theme.GenerateAcornKey(ctx.Runner, ctx.Paths.Root)
}
