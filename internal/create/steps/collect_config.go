package steps

import "github.com/gotcha190/toba/internal/create"

type CollectConfigStep struct{}

// NewCollectConfigStep creates the pipeline step that normalizes and logs the
// project configuration.
//
// Parameters:
// - none
//
// Returns:
// - a configured CollectConfigStep instance
func NewCollectConfigStep() *CollectConfigStep {
	return &CollectConfigStep{}
}

// Name returns the human-readable pipeline label for this step.
//
// Parameters:
// - none
//
// Returns:
// - the display name used by pipeline logging
func (s *CollectConfigStep) Name() string {
	return "Collect project configuration"
}

// Run normalizes the config, rebuilds derived project paths, and logs the
// resolved create settings.
//
// Parameters:
// - ctx: shared create context containing the mutable config and path state
//
// Returns:
// - an error when config normalization fails
//
// Side effects:
// - mutates ctx.Config, ctx.DryRun, and ctx.Paths
// - writes the resolved configuration to the logger
func (s *CollectConfigStep) Run(ctx *create.Context) error {
	if err := ctx.Config.Normalize(); err != nil {
		return err
	}

	ctx.DryRun = ctx.Config.DryRun
	ctx.Paths = create.NewProjectPaths(ctx.Paths.BaseDir, ctx.Config.Name)

	ctx.Logger.Info("Project: " + ctx.Config.Name)
	ctx.Logger.Info("PHP: " + ctx.Config.PHPVersion)
	ctx.Logger.Info("Domain: " + ctx.Config.Domain)
	ctx.Logger.Info("Database: " + ctx.Config.Database)
	if ctx.DryRun {
		ctx.Logger.Info("Dry run enabled")
	}

	return nil
}
