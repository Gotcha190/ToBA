package theme

import (
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
		return "", create.NewCodedError("THEME_DIR_EXISTS", "theme directory already exists: "+targetDir, nil)
	} else if !os.IsNotExist(err) {
		return "", err
	}

	if err := runner.Run(themesDir, "git", "clone", starterRepo, themeName); err != nil {
		return "", create.NewCodedError("THEME_CLONE_FAILED", "starter theme clone failed", err)
	}

	return targetDir, nil
}

func Build(runner create.CommandRunner, themeDir string) error {
	if err := runner.Run(themeDir, "lando", "composer", "install"); err != nil {
		return create.NewCodedError("THEME_BUILD_FAILED", "starter theme composer install failed", err)
	}

	if err := runner.Run(themeDir, "npm", "i"); err != nil {
		return create.NewCodedError("THEME_BUILD_FAILED", "starter theme npm install failed", err)
	}

	if err := runner.Run(themeDir, "npm", "run", "build"); err != nil {
		return create.NewCodedError("THEME_BUILD_FAILED", "starter theme build failed", err)
	}

	return nil
}

func GenerateAcornKey(runner create.CommandRunner, projectDir string) error {
	for range 2 {
		if err := runner.Run(projectDir, "lando", "wp", "acorn", "key:generate"); err != nil {
			return create.NewCodedError("ACORN_KEY_GENERATE_FAILED", "acorn key generation failed", err)
		}
	}

	return nil
}
