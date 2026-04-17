package steps

import "github.com/gotcha190/toba/internal/create"

type ImportUploadsStep struct{}

// NewImportUploadsStep creates the pipeline step that restores uploads
// archives into wp-content.
//
// Parameters:
// - none
//
// Returns:
// - a configured ImportUploadsStep instance
func NewImportUploadsStep() *ImportUploadsStep {
	return &ImportUploadsStep{}
}

// Name returns the human-readable pipeline label for this step.
//
// Parameters:
// - none
//
// Returns:
// - the display name used by pipeline logging
func (s *ImportUploadsStep) Name() string {
	return "Import uploads"
}

// Run restores prepared uploads archives into the project wp-content directory.
//
// Parameters:
// - ctx: shared create context containing prepared uploads archives and project paths
//
// Returns:
// - an error when no uploads archives were prepared or archive extraction fails
//
// Side effects:
// - extracts uploads archives into the project wp-content directory
func (s *ImportUploadsStep) Run(ctx *create.Context) error {
	return restoreLocalZips(ctx, ctx.StarterData.UploadsPaths, ctx.Paths.WPContent, "uploads")
}
