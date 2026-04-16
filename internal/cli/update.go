package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/gotcha190/ToBA/internal/templatesync"
)

func RunUpdate() error {
	repoRoot, err := os.Getwd()
	if err != nil {
		return err
	}

	repoRoot, err = filepath.Abs(repoRoot)
	if err != nil {
		return err
	}

	if err := validateUpdateRepoRoot(repoRoot); err != nil {
		return err
	}

	if err := templatesync.SyncRepo(repoRoot); err != nil {
		return err
	}

	fmt.Println("Synced static templates to embedded files.")
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
