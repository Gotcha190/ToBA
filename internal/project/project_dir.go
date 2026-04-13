package project

import "os"

type ErrDirExists struct {
	Path string
}

func (e ErrDirExists) Error() string {
	return "directory already exists: " + e.Path
}

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
