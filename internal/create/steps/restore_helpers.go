package steps

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/gotcha190/ToBA/internal/create"
	"github.com/gotcha190/ToBA/internal/project"
	"github.com/gotcha190/ToBA/internal/templates"
)

func restoreTemplateZips(ctx *create.Context, templateDir string, destination string) error {
	archives, err := listTemplateFiles(templateDir, ".zip")
	if err != nil {
		return err
	}
	if len(archives) == 0 {
		return fmt.Errorf("no ZIP archives found in %s", templateDir)
	}

	for _, archive := range archives {
		if ctx.DryRun {
			ctx.Logger.Info("Would extract: " + archive + " -> " + destination)
			continue
		}

		content, err := templates.Read(archive)
		if err != nil {
			return err
		}
		if err := project.ExtractZip(content, destination); err != nil {
			return fmt.Errorf("extract %s: %w", archive, err)
		}

		ctx.Logger.Info("Extracted: " + archive)
	}

	return nil
}

func listTemplateFiles(templateDir string, suffix string) ([]string, error) {
	if strings.HasPrefix(templateDir, "wordpress/") {
		category := strings.TrimPrefix(templateDir, "wordpress/")
		return templates.WordPressBackupFiles(category, suffix)
	}

	entries, err := templates.List(templateDir)
	if err != nil {
		return nil, err
	}

	var matches []string
	for _, entry := range entries {
		if strings.HasSuffix(strings.ToLower(entry), strings.ToLower(suffix)) {
			matches = append(matches, entry)
		}
	}

	return matches, nil
}

func relativePath(baseDir string, target string) string {
	relative, err := filepath.Rel(baseDir, target)
	if err != nil {
		return target
	}

	return relative
}
