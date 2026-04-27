package sourcedata

import (
	"os"
	"path/filepath"
)

// makeTempDir creates a temporary working directory for prepared starter assets.
//
// Returns:
// - the temporary directory path
// - an error when the OS cannot create the directory
func makeTempDir() (string, error) {
	return os.MkdirTemp("", "toba-starter-*")
}

// pathBase returns the last path element for local or remote file paths.
//
// Returns:
// - the trailing base name component
func pathBase(path string) string {
	return filepath.Base(path)
}
