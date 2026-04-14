package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/gotcha190/ToBA/internal/templatesync"
)

type UpdateOptions struct {
	LinkPath string
}

func RunUpdate(opts UpdateOptions) error {
	repoRoot, err := os.Getwd()
	if err != nil {
		return err
	}

	repoRoot, err = filepath.Abs(repoRoot)
	if err != nil {
		return err
	}

	if opts.LinkPath == "" {
		if err := validateUpdateRepoRoot(repoRoot); err != nil {
			return err
		}

		if err := templatesync.SyncRepo(repoRoot); err != nil {
			return err
		}

		version, err := templatesync.EmbeddedDataVersion(repoRoot)
		if err != nil {
			return err
		}

		fmt.Printf("Synced templates to embedded files. Data version: %s\n", version)
		return nil
	}

	info, version, err := templatesync.InstallOverrideBackup(opts.LinkPath)
	if err != nil {
		return err
	}

	overrideRoot, err := templatesync.OverrideTemplatesDir()
	if err != nil {
		return err
	}

	fmt.Printf(
		"Installed %s override: %s -> %s\nOverride data version: %s\n",
		info.Category,
		info.SourcePath,
		filepath.Join(overrideRoot, info.TargetDir, info.FileName),
		version,
	)
	return nil
}

func validateUpdateRepoRoot(repoRoot string) error {
	requiredPaths := []string{
		filepath.Join(repoRoot, "go.mod"),
		filepath.Join(repoRoot, "templates"),
		filepath.Join(repoRoot, "internal", "templates"),
	}

	for _, path := range requiredPaths {
		if _, err := os.Stat(path); err != nil {
			if os.IsNotExist(err) {
				return fmt.Errorf("run 'ToBA update' from the ToBA repository root; missing %s", path)
			}
			return err
		}
	}

	return nil
}
