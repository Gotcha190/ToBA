package steps

import (
	"github.com/gotcha190/toba/internal/create"
	"github.com/gotcha190/toba/internal/wordpress"
)

type FlushRewriteRulesStep struct{}

// NewFlushRewriteRulesStep creates the pipeline step that refreshes WordPress
// rewrite rules.
//
// Parameters:
// - none
//
// Returns:
// - a configured FlushRewriteRulesStep instance
func NewFlushRewriteRulesStep() *FlushRewriteRulesStep {
	return &FlushRewriteRulesStep{}
}

// Name returns the human-readable pipeline label for this step.
//
// Parameters:
// - none
//
// Returns:
// - the display name used by pipeline logging
func (s *FlushRewriteRulesStep) Name() string {
	return "Refresh permalinks"
}

// Run flushes WordPress rewrite rules in the local Lando environment.
//
// Parameters:
// - ctx: shared create context containing project paths and runner access
//
// Returns:
// - an error when the rewrite flush command fails
//
// Side effects:
// - runs `lando wp rewrite flush --hard` unless dry-run mode is enabled
func (s *FlushRewriteRulesStep) Run(ctx *create.Context) error {
	if ctx.DryRun {
		ctx.Logger.Info("Would run: lando wp rewrite flush --hard")
		return nil
	}

	return wordpress.FlushRewriteRules(ctx.Runner, ctx.Paths.Root)
}
