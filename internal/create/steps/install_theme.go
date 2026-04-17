package steps

import (
	"github.com/gotcha190/toba/internal/create"
	"github.com/gotcha190/toba/internal/theme"
)

type InstallThemeStep struct{}

// NewInstallThemeStep creates the pipeline step that restores local theme
// archives or clones the starter theme repository.
//
// Parameters:
// - none
//
// Returns:
// - a configured InstallThemeStep instance
func NewInstallThemeStep() *InstallThemeStep {
	return &InstallThemeStep{}
}

// Name returns the human-readable pipeline label for this step.
//
// Parameters:
// - none
//
// Returns:
// - the display name used by pipeline logging
func (s *InstallThemeStep) Name() string {
	return "Install starter theme"
}

// Run installs the theme from local archives or the configured starter
// repository depending on the resolved starter data.
//
// Parameters:
// - ctx: shared create context containing starter data, paths, config, and runner access
//
// Returns:
// - an error when local theme extraction fails or the starter theme cannot be cloned
//
// Side effects:
// - may extract theme archives into wp-content
// - may run `git clone` through the configured runner
// - logs planned actions instead of mutating the project in dry-run mode
func (s *InstallThemeStep) Run(ctx *create.Context) error {
	if len(ctx.StarterData.ThemePaths) > 0 {
		if ctx.DryRun {
			ctx.Logger.Info("Would extract local themes: " + ctx.Paths.WPContent)
			return nil
		}

		return restoreLocalZips(ctx, ctx.StarterData.ThemePaths, ctx.Paths.WPContent, "themes")
	}

	if ctx.DryRun {
		if ctx.Config.StarterRepo == "" {
			return theme.MissingStarterRepoError{}
		}
		ctx.Logger.Info("Would run: git clone " + ctx.Config.StarterRepo + " " + ctx.Config.Name)
		return nil
	}

	_, err := theme.Install(ctx.Runner, ctx.Paths.Themes, ctx.Config.StarterRepo, ctx.Config.Name)
	return err
}
