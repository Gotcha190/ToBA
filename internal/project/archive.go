package project

import (
	"archive/zip"
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

func ExtractZip(data []byte, destDir string) error {
	reader, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return err
	}

	return extractZip(reader, destDir)
}

func ExtractZipFile(sourcePath string, destDir string) error {
	file, err := os.Open(sourcePath)
	if err != nil {
		return err
	}
	defer file.Close()

	info, err := file.Stat()
	if err != nil {
		return err
	}

	reader, err := zip.NewReader(file, info.Size())
	if err != nil {
		return err
	}

	return extractZip(reader, destDir)
}

func extractZip(reader *zip.Reader, destDir string) error {
	for _, file := range reader.File {
		targetPath, err := secureJoin(destDir, file.Name)
		if err != nil {
			return err
		}

		mode := file.Mode()
		if mode&os.ModeSymlink != 0 {
			return fmt.Errorf("archive entry is a symlink: %s", file.Name)
		}

		if file.FileInfo().IsDir() {
			if err := os.MkdirAll(targetPath, 0755); err != nil {
				return err
			}
			continue
		}

		if err := os.MkdirAll(filepath.Dir(targetPath), 0755); err != nil {
			return err
		}

		input, err := file.Open()
		if err != nil {
			return err
		}

		fileMode := mode.Perm()
		if fileMode == 0 {
			fileMode = 0644
		}

		output, err := os.OpenFile(targetPath, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, fileMode)
		if err != nil {
			input.Close()
			return err
		}

		_, copyErr := io.Copy(output, input)
		closeErr := output.Close()
		inputCloseErr := input.Close()

		if copyErr != nil {
			return copyErr
		}
		if closeErr != nil {
			return closeErr
		}
		if inputCloseErr != nil {
			return inputCloseErr
		}
	}

	return nil
}

func WriteGzipFile(data []byte, destPath string) error {
	reader, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		return err
	}
	defer reader.Close()

	return copyReaderToFile(reader, destPath, 0644)
}

func WriteGzipPath(sourcePath string, destPath string) error {
	source, err := os.Open(sourcePath)
	if err != nil {
		return err
	}
	defer source.Close()

	reader, err := gzip.NewReader(source)
	if err != nil {
		return err
	}
	defer reader.Close()

	return copyReaderToFile(reader, destPath, 0644)
}

func CopyFile(sourcePath string, destPath string) error {
	source, err := os.Open(sourcePath)
	if err != nil {
		return err
	}
	defer source.Close()

	info, err := source.Stat()
	if err != nil {
		return err
	}

	mode := info.Mode().Perm()
	if mode == 0 {
		mode = 0644
	}

	return copyReaderToFile(source, destPath, mode)
}

func copyReaderToFile(reader io.Reader, destPath string, mode os.FileMode) error {
	if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
		return err
	}

	output, err := os.OpenFile(destPath, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, mode)
	if err != nil {
		return err
	}
	defer output.Close()

	if _, err := io.Copy(output, reader); err != nil {
		return err
	}

	return nil
}

func secureJoin(baseDir string, entryName string) (string, error) {
	cleanBase := filepath.Clean(baseDir)
	target := filepath.Join(cleanBase, filepath.FromSlash(entryName))
	relative, err := filepath.Rel(cleanBase, target)
	if err != nil {
		return "", err
	}

	if relative == ".." || strings.HasPrefix(relative, ".."+string(os.PathSeparator)) {
		return "", fmt.Errorf("archive entry escapes destination: %s", entryName)
	}

	return target, nil
}
