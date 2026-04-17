package steps

import (
	"os"
	"path/filepath"

	"github.com/gotcha190/toba/internal/create"
	"github.com/gotcha190/toba/internal/lando"
	"github.com/gotcha190/toba/internal/templates"
)

type GenerateLandoConfigStep struct{}

func NewGenerateLandoConfigStep() *GenerateLandoConfigStep {
	return &GenerateLandoConfigStep{}
}

func (s *GenerateLandoConfigStep) Name() string {
	return "Generate .lando.yml"
}

func (s *GenerateLandoConfigStep) Run(ctx *create.Context) error {
	rendered, err := lando.RenderConfig(ctx.Config)
	if err != nil {
		return err
	}

	phpINI, err := templates.Read("config/php.ini")
	if err != nil {
		return err
	}
	wpCLIConfig, err := templates.Read("wp-cli.yml")
	if err != nil {
		return err
	}

	target := filepath.Join(ctx.Paths.Root, ".lando.yml")
	phpINITarget := filepath.Join(ctx.Paths.ConfigDir, "php.ini")
	wpCLITarget := filepath.Join(ctx.Paths.Root, "wp-cli.yml")
	if ctx.DryRun {
		ctx.Logger.Info("Would write: " + target)
		ctx.Logger.Info("Would write: " + phpINITarget)
		ctx.Logger.Info("Would write: " + wpCLITarget)
		return nil
	}

	if err := os.WriteFile(target, rendered, 0644); err != nil {
		return err
	}
	if err := os.WriteFile(phpINITarget, phpINI, 0644); err != nil {
		return err
	}
	if err := os.WriteFile(wpCLITarget, wpCLIConfig, 0644); err != nil {
		return err
	}

	ctx.Logger.Info("Generated: " + target)
	ctx.Logger.Info("Generated: " + phpINITarget)
	ctx.Logger.Info("Generated: " + wpCLITarget)
	return nil
}
