package steps

import (
	"github.com/gotcha190/toba/internal/create"
	"github.com/gotcha190/toba/internal/wordpress"
)

type RefreshThemeCachesStep struct{}

func NewRefreshThemeCachesStep() *RefreshThemeCachesStep {
	return &RefreshThemeCachesStep{}
}

func (s *RefreshThemeCachesStep) Name() string {
	return "Refresh theme caches"
}

func (s *RefreshThemeCachesStep) Run(ctx *create.Context) error {
	if ctx.DryRun {
		ctx.Logger.Info("Would run: lando wp acorn optimize")
		ctx.Logger.Info("Would run: lando wp acorn cache:clear")
		ctx.Logger.Info("Would run: lando wp acorn acf:cache")
		return nil
	}

	if err := wordpress.RefreshThemeCaches(ctx.Runner, ctx.Paths.Root); err != nil {
		ctx.Logger.Warning(
			"Failed to refresh theme caches automatically. Run manually: " +
				"lando wp acorn optimize, lando wp acorn cache:clear, lando wp acorn acf:cache. Error: " +
				err.Error(),
		)
	}

	return nil
}
