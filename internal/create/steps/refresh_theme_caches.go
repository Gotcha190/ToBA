package steps

import (
	"github.com/gotcha190/toba/internal/create"
	"github.com/gotcha190/toba/internal/wordpress"
)

type RefreshThemeCachesStep struct{}

// NewRefreshThemeCachesStep creates the pipeline step that rebuilds
// theme-related cache layers after restore.
//
// Returns:
// - a configured RefreshThemeCachesStep instance
func NewRefreshThemeCachesStep() *RefreshThemeCachesStep {
	return &RefreshThemeCachesStep{}
}

// Name returns the human-readable pipeline label for this step.
//
// Returns:
// - the display name used by pipeline logging
func (s *RefreshThemeCachesStep) Name() string {
	return "Refresh theme caches"
}

// Run refreshes theme caches and downgrades failures to warnings so the
// project can still finish bootstrapping.
//
// Parameters:
// - ctx: shared create context containing project paths, runner access, and logger access
//
// Returns:
// - always nil; refresh failures are reported as warnings instead
//
// Side effects:
// - may run Acorn cache maintenance commands
// - may write warning messages when cache refresh fails
func (s *RefreshThemeCachesStep) Run(ctx *create.Context) error {
	if ctx.DryRun {
		ctx.Logger.Info("Would detect available Acorn cache commands and run the appropriate refresh sequence")
		ctx.Logger.Info("Standard: lando wp acorn optimize, lando wp acorn cache:clear, lando wp acorn acf:cache")
		ctx.Logger.Info("Fallback: lando wp acorn optimize:clear, lando wp acorn acf:cache")
		return nil
	}

	if err := wordpress.RefreshThemeCaches(ctx.Runner, ctx.Paths.Root); err != nil {
		ctx.Logger.Warning(
			"Failed to refresh theme caches automatically. Run manually: " +
				"lando wp acorn list and use either " +
				"`optimize + cache:clear + acf:cache` or `optimize:clear + acf:cache`. Error: " +
				err.Error(),
		)
	}

	return nil
}
