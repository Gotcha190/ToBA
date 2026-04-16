package steps

import (
	"path/filepath"

	"github.com/gotcha190/ToBA/internal/create"
	"github.com/gotcha190/ToBA/internal/theme"
)

type BuildThemeStep struct{}

func NewBuildThemeStep() *BuildThemeStep {
	return &BuildThemeStep{}
}

func (s *BuildThemeStep) Name() string {
	return "Build starter theme"
}

func (s *BuildThemeStep) Run(ctx *create.Context) error {
	if len(ctx.StarterData.ThemePaths) > 0 {
		ctx.Logger.Info("Skipping starter build: using embedded theme backup")
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
