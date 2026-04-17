package steps

import (
	"os"
	"path/filepath"

	"github.com/gotcha190/ToBA/internal/project"
)

func makeStarterTempDir() (string, error) {
	return os.MkdirTemp("", "toba-starter-*")
}

func copyLocalFileToTemp(tempDir string, category string, sourcePath string) (string, error) {
	if sourcePath == "" {
		return "", nil
	}

	targetPath := filepath.Join(tempDir, category, filepath.Base(sourcePath))
	if err := project.CopyFile(sourcePath, targetPath); err != nil {
		return "", err
	}

	return targetPath, nil
}

func copyLocalFilesToTemp(tempDir string, category string, sourcePaths []string) ([]string, error) {
	targetPaths := make([]string, 0, len(sourcePaths))
	for _, sourcePath := range sourcePaths {
		targetPath, err := copyLocalFileToTemp(tempDir, category, sourcePath)
		if err != nil {
			return nil, err
		}
		targetPaths = append(targetPaths, targetPath)
	}

	return targetPaths, nil
}

func tempPathFromSource(tempDir string, category string, sourcePath string) string {
	if sourcePath == "" {
		return ""
	}
	return filepath.Join(tempDir, category, filepath.Base(sourcePath))
}

func tempPathsFromSources(tempDir string, category string, sourcePaths []string) []string {
	targetPaths := make([]string, 0, len(sourcePaths))
	for _, sourcePath := range sourcePaths {
		targetPaths = append(targetPaths, tempPathFromSource(tempDir, category, sourcePath))
	}

	return targetPaths
}

func pathBase(path string) string {
	return filepath.Base(path)
}
