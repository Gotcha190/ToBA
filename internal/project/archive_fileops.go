package project

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// WriteGzipFile expands gzipped data held in memory into destPath.
//
// Parameters:
// - data: raw gzipped bytes
// - destPath: destination file path for the decompressed content
//
// Returns:
// - an error when the gzip stream cannot be decoded or written
//
// Side effects:
// - writes the decompressed file to destPath
func WriteGzipFile(data []byte, destPath string) error {
	reader, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		return err
	}
	defer func() {
		_ = reader.Close()
	}()

	return copyReaderToFile(reader, destPath, 0644)
}

// WriteGzipPath expands a gzipped source file into destPath.
//
// Parameters:
// - sourcePath: gzip file path on disk
// - destPath: destination file path for the decompressed content
//
// Returns:
// - an error when the gzip file cannot be opened, decoded, or written
//
// Side effects:
// - reads the compressed file from disk
// - writes the decompressed file to destPath
func WriteGzipPath(sourcePath string, destPath string) error {
	source, err := os.Open(sourcePath)
	if err != nil {
		return err
	}
	defer func() {
		_ = source.Close()
	}()

	reader, err := gzip.NewReader(source)
	if err != nil {
		return err
	}
	defer func() {
		_ = reader.Close()
	}()

	return copyReaderToFile(reader, destPath, 0644)
}

// CopyFile copies sourcePath to destPath while preserving the source file mode
// when possible.
//
// Parameters:
// - sourcePath: existing file to copy
// - destPath: destination file path
//
// Returns:
// - an error when the source file cannot be read or the destination cannot be written
//
// Side effects:
// - reads sourcePath from disk
// - writes a copy to destPath
func CopyFile(sourcePath string, destPath string) error {
	source, err := os.Open(sourcePath)
	if err != nil {
		return err
	}
	defer func() {
		_ = source.Close()
	}()

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

// copyReaderToFile writes all data from reader into destPath using mode.
//
// Parameters:
// - reader: source stream to copy from
// - destPath: destination file path
// - mode: file mode to use for the created destination file
//
// Returns:
// - an error when the destination cannot be created or writing fails
//
// Side effects:
// - creates parent directories for destPath when needed
// - writes the destination file on disk
func copyReaderToFile(reader io.Reader, destPath string, mode os.FileMode) error {
	if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
		return err
	}

	output, err := os.OpenFile(destPath, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, mode)
	if err != nil {
		return err
	}
	defer func() {
		_ = output.Close()
	}()

	if _, err := io.Copy(output, reader); err != nil {
		return err
	}

	return nil
}

// secureJoin joins baseDir with an archive entry name and rejects paths that
// would escape the destination directory.
//
// Parameters:
// - baseDir: extraction root
// - entryName: archive entry path to resolve
//
// Returns:
// - a safe destination path under baseDir
// - an error when the entry would escape the destination directory
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
