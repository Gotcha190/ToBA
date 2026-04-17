package steps

import (
	"os"
	"path/filepath"

	"github.com/gotcha190/toba/internal/project"
)

// makeStarterTempDir creates a temporary working directory for prepared
// starter assets.
//
// Parameters:
// - none
//
// Returns:
// - the temporary directory path
// - an error when the OS cannot create the directory
func makeStarterTempDir() (string, error) {
	return os.MkdirTemp("", "toba-starter-*")
}

// copyLocalFileToTemp copies one prepared starter file into its category
// folder inside tempDir.
//
// Parameters:
// - tempDir: temporary root used for prepared starter files
// - category: logical backup category such as database or plugins
// - sourcePath: source file to copy
//
// Returns:
// - the copied file path inside tempDir
// - an error when the source file cannot be copied
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

// copyLocalFilesToTemp copies multiple prepared starter files into tempDir and
// returns their new paths.
//
// Parameters:
// - tempDir: temporary root used for prepared starter files
// - category: logical backup category
// - sourcePaths: source files to copy
//
// Returns:
// - the copied file paths inside tempDir
// - an error when any file copy fails
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

// tempPathFromSource returns the temp path that would be used for sourcePath
// without copying the file.
//
// Parameters:
// - tempDir: temporary root used for prepared starter files
// - category: logical backup category
// - sourcePath: original source file path
//
// Returns:
// - the derived temp path, or an empty string when sourcePath is empty
func tempPathFromSource(tempDir string, category string, sourcePath string) string {
	if sourcePath == "" {
		return ""
	}
	return filepath.Join(tempDir, category, filepath.Base(sourcePath))
}

// tempPathsFromSources maps source paths to their expected locations in tempDir.
//
// Parameters:
// - tempDir: temporary root used for prepared starter files
// - category: logical backup category
// - sourcePaths: original source file paths
//
// Returns:
// - the derived temp paths for every source file
func tempPathsFromSources(tempDir string, category string, sourcePaths []string) []string {
	targetPaths := make([]string, 0, len(sourcePaths))
	for _, sourcePath := range sourcePaths {
		targetPaths = append(targetPaths, tempPathFromSource(tempDir, category, sourcePath))
	}

	return targetPaths
}

// pathBase returns the last path element for local or remote file paths.
//
// Parameters:
// - path: filesystem or remote path
//
// Returns:
// - the trailing base name component
func pathBase(path string) string {
	return filepath.Base(path)
}
