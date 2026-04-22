package steps

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/gotcha190/toba/internal/create"
	"github.com/gotcha190/toba/internal/project"
)

// restoreLocalZip extracts a single prepared archive into destination.
//
// Parameters:
// - ctx: shared create context containing logger state and dry-run mode
// - archivePath: prepared archive file to extract
// - destination: extraction target directory
// - label: logical backup category used in error messages
//
// Returns:
// - an error when the archive is missing or extraction fails
//
// Side effects:
// - may extract files into destination
// - logs the extraction result or planned action
func restoreLocalZip(ctx *create.Context, archivePath string, destination string, label string) error {
	if archivePath == "" {
		return fmt.Errorf("%s archive is not prepared", label)
	}

	if ctx.DryRun {
		ctx.Logger.Info("Would extract: " + archivePath + " -> " + destination)
		return nil
	}

	if err := project.ExtractZipFile(archivePath, destination); err != nil {
		return fmt.Errorf("extract %s: %w", archivePath, err)
	}

	ctx.Logger.Info("Extracted: " + archivePath)
	return nil
}

// restoreLocalZips extracts a prepared archive set into destination.
//
// Parameters:
// - ctx: shared create context containing logger state and dry-run mode
// - archivePaths: prepared archives to extract
// - destination: extraction target directory
// - label: logical backup category used in logs and errors
//
// Returns:
// - an error when no archives were prepared or any archive extraction fails
//
// Side effects:
// - may extract multiple archives into destination
// - logs planned or completed extraction actions
func restoreLocalZips(ctx *create.Context, archivePaths []string, destination string, label string) error {
	if len(archivePaths) == 0 {
		return fmt.Errorf("%s archives are not prepared", label)
	}

	if ctx.DryRun {
		ctx.Logger.Info("Would extract " + label + ": " + strings.Join(archivePaths, ", "))
		return nil
	}

	if len(archivePaths) == 1 {
		return restoreLocalZip(ctx, archivePaths[0], destination, label)
	}

	if err := project.ExtractZipFiles(archivePaths, destination); err != nil {
		return fmt.Errorf("extract %s archives: %w", label, err)
	}

	for _, archivePath := range archivePaths {
		ctx.Logger.Info("Extracted: " + archivePath)
	}

	return nil
}

// relativePath returns target relative to baseDir and falls back to target on
// path resolution failures.
//
// Parameters:
// - baseDir: base directory used for the relative-path calculation
// - target: path that should be made relative to baseDir
//
// Returns:
// - the relative path, or target unchanged when the calculation fails
func relativePath(baseDir string, target string) string {
	relative, err := filepath.Rel(baseDir, target)
	if err != nil {
		return target
	}

	return relative
}
