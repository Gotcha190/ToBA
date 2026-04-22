package theme

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/gotcha190/toba/internal/create"
)

type MissingStarterRepoError struct{}

// Error explains how to provide the missing starter repository setting.
//
// Parameters:
// - none
//
// Returns:
// - the human-readable error string
func (e MissingStarterRepoError) Error() string {
	return "starter repo is not configured; add TOBA_STARTER_REPO to ~/.config/toba/.env via 'toba config' or pass --starter-repo and try again"
}

// Install clones the configured starter repository into themesDir/themeName.
//
// Parameters:
// - runner: command runner used to launch git
// - themesDir: local WordPress themes directory
// - starterRepo: git repository to clone
// - themeName: destination directory name for the cloned theme
//
// Returns:
// - the local theme directory path
// - an error when the repository is missing, the destination exists, or cloning fails
//
// Side effects:
// - may create the themes directory
// - runs `git clone` through the provided runner
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

// Build installs dependencies and builds the cloned starter theme.
//
// Parameters:
// - runner: command runner used for dependency installation and build commands
// - themeDir: local path to the cloned starter theme
//
// Returns:
// - an error when any dependency installation or build command fails
//
// Side effects:
// - runs `lando composer install`, `npm i`, and `npm run build`
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

// GenerateAcornKey runs the Acorn key generation command twice for the local
// project.
//
// Parameters:
// - runner: command runner used to launch the Acorn command
// - projectDir: local project root in which the command should run
//
// Returns:
// - an error when either Acorn key generation command fails
//
// Side effects:
// - runs `lando wp acorn key:generate` twice
func GenerateAcornKey(runner create.CommandRunner, projectDir string) error {
	for range 2 {
		if err := runner.Run(projectDir, "lando", "wp", "acorn", "key:generate"); err != nil {
			return create.NewCodedError("ACORN_KEY_GENERATE_FAILED", "acorn key generation failed", err)
		}
	}

	return nil
}
