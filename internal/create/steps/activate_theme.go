package steps

import "github.com/gotcha190/toba/internal/create"
import "github.com/gotcha190/toba/internal/wordpress"

type ActivateThemeStep struct{}

// NewActivateThemeStep creates the pipeline step responsible for activating
// the final WordPress theme after import and restore.
//
// Parameters:
// - none
//
// Returns:
// - a configured ActivateThemeStep instance
func NewActivateThemeStep() *ActivateThemeStep {
	return &ActivateThemeStep{}
}

// Name returns the human-readable pipeline label for this step.
//
// Parameters:
// - none
//
// Returns:
// - the display name used by pipeline logging
func (s *ActivateThemeStep) Name() string {
	return "Activate starter theme"
}

// Run activates the restored theme slug or the starter-repo theme name,
// depending on the selected starter data mode.
//
// Parameters:
// - ctx: shared create context containing starter data, project paths, and runner access
//
// Returns:
// - an error when theme detection or activation fails
//
// Side effects:
// - may query the imported database for the active theme slug
// - runs `lando wp theme activate ...` unless dry-run mode is enabled
func (s *ActivateThemeStep) Run(ctx *create.Context) error {
	if len(ctx.StarterData.ThemePaths) > 0 {
		if ctx.DryRun {
			ctx.Logger.Info("Would detect active theme slug from imported database and activate it")
			return nil
		}

		themeSlug, err := wordpress.DetectImportedThemeSlug(ctx.Runner, ctx.Paths.Root)
		if err != nil {
			return err
		}

		return wordpress.ActivateTheme(ctx.Runner, ctx.Paths.Root, themeSlug)
	}

	if ctx.DryRun {
		ctx.Logger.Info("Would run: lando wp theme activate " + ctx.Config.Name)
		return nil
	}

	return wordpress.ActivateTheme(ctx.Runner, ctx.Paths.Root, ctx.Config.Name)
}
