package project

import "os"

type ErrDirExists struct {
	Path string
}

// Error formats the path of the directory that already exists.
//
// Parameters:
// - none
//
// Returns:
// - the human-readable error string
func (e ErrDirExists) Error() string {
	return "directory already exists: " + e.Path
}

// DirExists reports whether path exists and is a directory.
//
// Parameters:
// - path: filesystem path to inspect
//
// Returns:
// - true when path exists and is a directory
// - an error when the filesystem lookup fails unexpectedly
func DirExists(path string) (bool, error) {
	info, err := os.Stat(path)
	if err == nil {
		return info.IsDir(), nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}

// CreateDir creates path or returns ErrDirExists when the directory already
// exists.
//
// Parameters:
// - path: directory path to create
//
// Returns:
// - an error when the directory already exists or cannot be created
//
// Side effects:
// - creates the directory and any missing parents on disk
func CreateDir(path string) error {
	info, err := os.Stat(path)
	if err == nil && info.IsDir() {
		return ErrDirExists{Path: path}
	}

	if err != nil && !os.IsNotExist(err) {
		return err
	}

	return os.MkdirAll(path, 0755)
}
