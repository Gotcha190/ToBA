package steps

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/gotcha190/ToBA/internal/create"
	"github.com/gotcha190/ToBA/internal/project"
)

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

func restoreLocalZips(ctx *create.Context, archivePaths []string, destination string, label string) error {
	if len(archivePaths) == 0 {
		return fmt.Errorf("%s archives are not prepared", label)
	}

	if ctx.DryRun {
		ctx.Logger.Info("Would extract " + label + ": " + strings.Join(archivePaths, ", "))
		return nil
	}

	for _, archivePath := range archivePaths {
		if err := restoreLocalZip(ctx, archivePath, destination, label); err != nil {
			return err
		}
	}

	return nil
}

func relativePath(baseDir string, target string) string {
	relative, err := filepath.Rel(baseDir, target)
	if err != nil {
		return target
	}

	return relative
}
