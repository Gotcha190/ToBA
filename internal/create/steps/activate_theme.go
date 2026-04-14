package steps

import "github.com/gotcha190/ToBA/internal/create"
import "github.com/gotcha190/ToBA/internal/wordpress"

type ActivateThemeStep struct{}

func NewActivateThemeStep() *ActivateThemeStep {
	return &ActivateThemeStep{}
}

func (s *ActivateThemeStep) Name() string {
	return "Activate starter theme"
}

func (s *ActivateThemeStep) Run(ctx *create.Context) error {
	if ctx.DryRun {
		ctx.Logger.Info("Would run: lando wp theme activate " + ctx.Config.Name)
		return nil
	}

	return wordpress.ActivateTheme(ctx.Runner, ctx.Paths.Root, ctx.Config.Name)
}
