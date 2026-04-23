package steps

import (
	"os"
	"path/filepath"

	"github.com/gotcha190/toba/internal/create"
)

type ClearImportedCachesStep struct{}

// NewClearImportedCachesStep creates the pipeline step that removes imported
// caches and flushes the WordPress object cache.
//
// Returns:
// - a configured ClearImportedCachesStep instance
func NewClearImportedCachesStep() *ClearImportedCachesStep {
	return &ClearImportedCachesStep{}
}

// Name returns the human-readable pipeline label for this step.
//
// Returns:
// - the display name used by pipeline logging
func (s *ClearImportedCachesStep) Name() string {
	return "Clear imported caches"
}

// Run removes wp-content/cache and flushes the WordPress cache inside the
// local environment.
//
// Parameters:
// - ctx: shared create context containing project paths and runner access
//
// Returns:
// - an error when cache deletion or cache flushing fails
//
// Side effects:
// - removes the imported cache directory from disk
// - runs `lando wp cache flush` unless dry-run mode is enabled
func (s *ClearImportedCachesStep) Run(ctx *create.Context) error {
	cacheDir := filepath.Join(ctx.Paths.WPContent, "cache")

	if ctx.DryRun {
		ctx.Logger.Info("Would remove: " + cacheDir)
		ctx.Logger.Info("Would run: lando wp cache flush")
		return nil
	}

	if err := os.RemoveAll(cacheDir); err != nil {
		return err
	}

	ctx.Logger.Info("Running: lando wp cache flush")
	if err := ctx.Runner.Run(ctx.Paths.Root, "lando", "wp", "cache", "flush"); err != nil {
		return err
	}

	return nil
}
