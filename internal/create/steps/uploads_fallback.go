package steps

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/gotcha190/toba/internal/create"
	"github.com/gotcha190/toba/internal/wordpress"
)

type ConfigureUploadsFallbackStep struct{}

// NewConfigureUploadsFallbackStep creates the pipeline step that writes the
// `.htaccess` fallback for skipped SSH uploads.
//
// Returns:
// - a configured ConfigureUploadsFallbackStep instance
func NewConfigureUploadsFallbackStep() *ConfigureUploadsFallbackStep {
	return &ConfigureUploadsFallbackStep{}
}

// Name returns the human-readable pipeline label for this step.
//
// Returns:
// - the display name used by pipeline logging
func (s *ConfigureUploadsFallbackStep) Name() string {
	return "Configure uploads fallback"
}

// Run writes the ToBA uploads fallback block into the project `.htaccess`.
//
// Parameters:
// - ctx: shared create context containing source URL and project paths
//
// Returns:
// - an error when source URL validation or file IO fails
//
// Side effects:
// - reads and writes app/.htaccess unless dry-run mode is enabled
func (s *ConfigureUploadsFallbackStep) Run(ctx *create.Context) error {
	if !ctx.Config.NoUploads {
		return nil
	}

	block, err := wordpress.UploadsFallbackBlock(ctx.StarterData.SourceURL)
	if err != nil {
		return err
	}

	htaccessPath := filepath.Join(ctx.Paths.AppDir, ".htaccess")
	if ctx.DryRun {
		ctx.Logger.Info("Would write uploads fallback to " + htaccessPath)
		return nil
	}

	existing, err := os.ReadFile(htaccessPath)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to read %s: %w", htaccessPath, err)
	}

	updated := wordpress.UpsertUploadsFallbackBlock(string(existing), block)
	if err := os.MkdirAll(filepath.Dir(htaccessPath), 0755); err != nil {
		return fmt.Errorf("failed to create parent directory for %s: %w", htaccessPath, err)
	}
	if err := os.WriteFile(htaccessPath, []byte(updated), 0644); err != nil {
		return fmt.Errorf("failed to write %s: %w", htaccessPath, err)
	}

	ctx.Logger.Info("Configured uploads fallback in " + htaccessPath)
	return nil
}
