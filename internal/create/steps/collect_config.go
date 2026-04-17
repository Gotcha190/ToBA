package steps

import "github.com/gotcha190/toba/internal/create"

type CollectConfigStep struct{}

func NewCollectConfigStep() *CollectConfigStep {
	return &CollectConfigStep{}
}

func (s *CollectConfigStep) Name() string {
	return "Collect project configuration"
}

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
