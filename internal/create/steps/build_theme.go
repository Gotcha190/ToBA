package steps

import (
	"path/filepath"

	"github.com/gotcha190/toba/internal/create"
	"github.com/gotcha190/toba/internal/theme"
)

type BuildThemeStep struct{}

// NewBuildThemeStep creates the pipeline step that builds the starter theme
// when the theme comes from a cloned repository.
//
// Parameters:
// - none
//
// Returns:
// - a configured BuildThemeStep instance
func NewBuildThemeStep() *BuildThemeStep {
	return &BuildThemeStep{}
}

// Name returns the human-readable pipeline label for this step.
//
// Parameters:
// - none
//
// Returns:
// - the display name used by pipeline logging
func (s *BuildThemeStep) Name() string {
	return "Build starter theme"
}

// Run builds the cloned starter theme unless the current run is restoring a
// local theme backup instead.
//
// Parameters:
// - ctx: shared create context containing starter mode, paths, and runner access
//
// Returns:
// - an error when dependency installation or the theme build fails
//
// Side effects:
// - may run `lando composer install`, `npm i`, and `npm run build`
// - writes dry-run messages instead of executing commands when dry-run mode is enabled
func (s *BuildThemeStep) Run(ctx *create.Context) error {
	if len(ctx.StarterData.ThemePaths) > 0 {
		ctx.Logger.Info("Skipping starter build: using local theme backup")
		return nil
	}

	themeDir := filepath.Join(ctx.Paths.Themes, ctx.Config.Name)

	if ctx.DryRun {
		ctx.Logger.Info("Would run: lando composer install")
		ctx.Logger.Info("Would run: npm i")
		ctx.Logger.Info("Would run: npm run build")
		return nil
	}

	return theme.Build(ctx.Runner, themeDir)
}
