package templatesync

import (
	"io"
	"os"
	"path/filepath"
	"strings"
)

const (
	sourceTemplatesDir = "templates"
	targetTemplatesDir = "internal/templates/files"
	wordPressDir       = "wordpress"
)

func SyncRepo(repoRoot string) error {
	sourceRoot := filepath.Join(repoRoot, sourceTemplatesDir)
	targetRoot := filepath.Join(repoRoot, targetTemplatesDir)
	return Sync(sourceRoot, targetRoot)
}

func Sync(sourceRoot string, targetRoot string) error {
	if err := os.RemoveAll(targetRoot); err != nil {
		return err
	}

	return copyTreeExceptWordPress(sourceRoot, targetRoot)
}

func copyTreeExceptWordPress(sourceRoot string, targetRoot string) error {
	return filepath.WalkDir(sourceRoot, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}

		relative, err := filepath.Rel(sourceRoot, path)
		if err != nil {
			return err
		}
		if relative == wordPressDir || strings.HasPrefix(relative, wordPressDir+string(os.PathSeparator)) {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		targetPath := filepath.Join(targetRoot, relative)
		if d.IsDir() {
			return os.MkdirAll(targetPath, 0755)
		}

		info, err := d.Info()
		if err != nil {
			return err
		}

		if err := os.MkdirAll(filepath.Dir(targetPath), 0755); err != nil {
			return err
		}

		return copyFile(path, targetPath, info.Mode().Perm())
	})
}

func copyFile(sourcePath string, targetPath string, mode os.FileMode) error {
	sourceFile, err := os.Open(sourcePath)
	if err != nil {
		return err
	}
	defer sourceFile.Close()

	targetFile, err := os.OpenFile(targetPath, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, mode)
	if err != nil {
		return err
	}
	defer targetFile.Close()

	if _, err := io.Copy(targetFile, sourceFile); err != nil {
		return err
	}

	return nil
}
