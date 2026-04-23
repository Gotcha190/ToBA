package steps

import "github.com/gotcha190/toba/internal/create"

type ImportPluginsStep struct{}

// NewImportPluginsStep creates the pipeline step that restores plugin archives
// into wp-content.
//
// Returns:
// - a configured ImportPluginsStep instance
func NewImportPluginsStep() *ImportPluginsStep {
	return &ImportPluginsStep{}
}

// Name returns the human-readable pipeline label for this step.
//
// Returns:
// - the display name used by pipeline logging
func (s *ImportPluginsStep) Name() string {
	return "Import plugins"
}

// Run restores prepared plugin archives into the project wp-content directory.
//
// Parameters:
// - ctx: shared create context containing prepared plugin archives and project paths
//
// Returns:
// - an error when no plugin archives were prepared or archive extraction fails
//
// Side effects:
// - extracts plugin archives into the project wp-content directory
func (s *ImportPluginsStep) Run(ctx *create.Context) error {
	return restoreLocalZips(ctx, ctx.StarterData.PluginsPaths, ctx.Paths.WPContent, "plugins")
}
