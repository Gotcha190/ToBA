package steps

import "github.com/gotcha190/toba/internal/create"

type ImportUploadsStep struct{}

func NewImportUploadsStep() *ImportUploadsStep {
	return &ImportUploadsStep{}
}

func (s *ImportUploadsStep) Name() string {
	return "Import uploads"
}

func (s *ImportUploadsStep) Run(ctx *create.Context) error {
	return restoreLocalZips(ctx, ctx.StarterData.UploadsPaths, ctx.Paths.WPContent, "uploads")
}
