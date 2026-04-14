package steps

import (
	"github.com/gotcha190/ToBA/internal/create"
	"github.com/gotcha190/ToBA/internal/theme"
)

type GenerateAcornKeyStep struct{}

func NewGenerateAcornKeyStep() *GenerateAcornKeyStep {
	return &GenerateAcornKeyStep{}
}

func (s *GenerateAcornKeyStep) Name() string {
	return "Generate Acorn key"
}

func (s *GenerateAcornKeyStep) Run(ctx *create.Context) error {
	if ctx.DryRun {
		ctx.Logger.Info("Would run: lando wp acorn key:generate")
		ctx.Logger.Info("Would run: lando wp acorn key:generate")
		return nil
	}

	return theme.GenerateAcornKey(ctx.Runner, ctx.Paths.Root)
}
