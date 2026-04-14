package steps

import "github.com/gotcha190/ToBA/internal/create"

type ImportPluginsStep struct{}

func NewImportPluginsStep() *ImportPluginsStep {
	return &ImportPluginsStep{}
}

func (s *ImportPluginsStep) Name() string {
	return "Import plugins"
}

func (s *ImportPluginsStep) Run(ctx *create.Context) error {
	return restoreTemplateZips(ctx, "wordpress/plugins", ctx.Paths.WPContent)
}
