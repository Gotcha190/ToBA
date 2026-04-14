package steps

import "github.com/gotcha190/ToBA/internal/create"

type ImportOthersStep struct{}

func NewImportOthersStep() *ImportOthersStep {
	return &ImportOthersStep{}
}

func (s *ImportOthersStep) Name() string {
	return "Import others"
}

func (s *ImportOthersStep) Run(ctx *create.Context) error {
	return restoreTemplateZips(ctx, "wordpress/others", ctx.Paths.WPContent)
}
