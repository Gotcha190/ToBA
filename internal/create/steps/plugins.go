package steps

import "github.com/gotcha190/toba/internal/create"

type ImportPluginsStep struct{}

func NewImportPluginsStep() *ImportPluginsStep {
	return &ImportPluginsStep{}
}

func (s *ImportPluginsStep) Name() string {
	return "Import plugins"
}

func (s *ImportPluginsStep) Run(ctx *create.Context) error {
	return restoreLocalZips(ctx, ctx.StarterData.PluginsPaths, ctx.Paths.WPContent, "plugins")
}
