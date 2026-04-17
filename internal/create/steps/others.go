package steps

import "github.com/gotcha190/toba/internal/create"

type ImportOthersStep struct{}

func NewImportOthersStep() *ImportOthersStep {
	return &ImportOthersStep{}
}

func (s *ImportOthersStep) Name() string {
	return "Import others"
}

func (s *ImportOthersStep) Run(ctx *create.Context) error {
	if len(ctx.StarterData.OthersPaths) == 0 {
		ctx.Logger.Info("Skipping others import: no local others backup prepared")
		return nil
	}

	return restoreLocalZips(ctx, ctx.StarterData.OthersPaths, ctx.Paths.WPContent, "others")
}
