package steps

import "github.com/gotcha190/ToBA/internal/create"

type ImportUploadsStep struct{}

func NewImportUploadsStep() *ImportUploadsStep {
	return &ImportUploadsStep{}
}

func (s *ImportUploadsStep) Name() string {
	return "Import uploads"
}

func (s *ImportUploadsStep) Run(ctx *create.Context) error {
	return restoreTemplateZips(ctx, "wordpress/uploads", ctx.Paths.WPContent)
}
