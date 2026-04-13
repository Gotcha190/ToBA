package steps

import (
	"github.com/gotcha190/ToBA/internal/create"
	"github.com/gotcha190/ToBA/internal/lando"
)

type StartLandoStep struct{}

func NewStartLandoStep() *StartLandoStep {
	return &StartLandoStep{}
}

func (s *StartLandoStep) Name() string {
	return "Start Lando"
}

func (s *StartLandoStep) Run(ctx *create.Context) error {
	if ctx.DryRun {
		ctx.Logger.Info("Would run: lando start")
		return nil
	}

	return lando.Start(ctx.Runner, ctx.Paths.Root)
}
