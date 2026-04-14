package theme

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/gotcha190/ToBA/internal/create"
)

type MissingStarterRepoError struct{}

func (e MissingStarterRepoError) Error() string {
	return "starter repo is not configured; add TOBA_STARTER_REPO to ~/.config/toba/.env via 'ToBA config init' or pass --starter-repo and try again"
}

func Install(runner create.CommandRunner, themesDir string, starterRepo string, themeName string) (string, error) {
	if strings.TrimSpace(starterRepo) == "" {
		return "", MissingStarterRepoError{}
	}

	if err := os.MkdirAll(themesDir, 0755); err != nil {
		return "", err
	}

	targetDir := filepath.Join(themesDir, themeName)
	if _, err := os.Stat(targetDir); err == nil {
		return "", fmt.Errorf("theme directory already exists: %s", targetDir)
	} else if !os.IsNotExist(err) {
		return "", err
	}

	if err := runner.Run(themesDir, "git", "clone", starterRepo, themeName); err != nil {
		return "", fmt.Errorf("starter theme clone failed: %w", err)
	}

	return targetDir, nil
}

func Build(runner create.CommandRunner, themeDir string) error {
	if err := runner.Run(themeDir, "lando", "composer", "install"); err != nil {
		return fmt.Errorf("starter theme composer install failed: %w", err)
	}

	if err := runner.Run(themeDir, "npm", "i"); err != nil {
		return fmt.Errorf("starter theme npm install failed: %w", err)
	}

	if err := runner.Run(themeDir, "npm", "run", "build"); err != nil {
		return fmt.Errorf("starter theme build failed: %w", err)
	}

	return nil
}

func GenerateAcornKey(runner create.CommandRunner, projectDir string) error {
	for range 2 {
		if err := runner.Run(projectDir, "lando", "wp", "acorn", "key:generate"); err != nil {
			return fmt.Errorf("acorn key generation failed: %w", err)
		}
	}

	return nil
}
