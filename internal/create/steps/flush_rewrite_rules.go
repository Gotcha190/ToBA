package steps

import (
	"github.com/gotcha190/ToBA/internal/create"
	"github.com/gotcha190/ToBA/internal/wordpress"
)

type FlushRewriteRulesStep struct{}

func NewFlushRewriteRulesStep() *FlushRewriteRulesStep {
	return &FlushRewriteRulesStep{}
}

func (s *FlushRewriteRulesStep) Name() string {
	return "Refresh permalinks"
}

func (s *FlushRewriteRulesStep) Run(ctx *create.Context) error {
	if ctx.DryRun {
		ctx.Logger.Info("Would run: lando wp rewrite flush --hard")
		return nil
	}

	return wordpress.FlushRewriteRules(ctx.Runner, ctx.Paths.Root)
}
