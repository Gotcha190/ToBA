package steps

import "github.com/gotcha190/toba/internal/create"

type ImportOthersStep struct{}

// NewImportOthersStep creates the pipeline step that restores optional
// "others" archives into wp-content.
//
// Returns:
// - a configured ImportOthersStep instance
func NewImportOthersStep() *ImportOthersStep {
	return &ImportOthersStep{}
}

// Name returns the human-readable pipeline label for this step.
//
// Returns:
// - the display name used by pipeline logging
func (s *ImportOthersStep) Name() string {
	return "Import others"
}

// Run restores optional others archives when they were prepared during the
// starter data phase.
//
// Parameters:
// - ctx: shared create context containing prepared others archives and project paths
//
// Returns:
// - an error when prepared archives exist but extraction fails
//
// Side effects:
// - may extract optional archives into wp-content
// - logs a skip message when no optional archives were prepared
func (s *ImportOthersStep) Run(ctx *create.Context) error {
	if len(ctx.StarterData.OthersPaths) == 0 {
		ctx.Logger.Info("Skipping others import: no local others backup prepared")
		return nil
	}

	return restoreLocalZips(ctx, ctx.StarterData.OthersPaths, ctx.Paths.WPContent, "others")
}
