package steps

import (
	"github.com/gotcha190/ToBA/internal/create"
	"github.com/gotcha190/ToBA/internal/theme"
)

type InstallThemeStep struct{}

func NewInstallThemeStep() *InstallThemeStep {
	return &InstallThemeStep{}
}

func (s *InstallThemeStep) Name() string {
	return "Install starter theme"
}

func (s *InstallThemeStep) Run(ctx *create.Context) error {
	if len(ctx.StarterData.ThemePaths) > 0 {
		if ctx.DryRun {
			ctx.Logger.Info("Would extract themes: " + ctx.Paths.WPContent)
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
