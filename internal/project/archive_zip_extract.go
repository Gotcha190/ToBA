package project

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// canUseSystemUnzip reports whether the system unzip binary is available.
//
// Returns:
// - true when unzip can be found on PATH
func canUseSystemUnzip() bool {
	_, err := exec.LookPath("unzip")
	return err == nil
}

const unzipBatchSize = 256

// extractZipWithSystemUnzip extracts selected files with the system unzip
// binary after creating required directories.
//
// Parameters:
// - sourcePath: zip archive path on disk
// - destDir: destination directory for extracted files
// - entries: validated archive entries
// - fileNames: file entry names to pass to unzip
//
// Returns:
// - an error when directory creation or unzip execution fails
//
// Side effects:
// - creates directories under destDir
// - executes the system unzip binary
func extractZipWithSystemUnzip(sourcePath string, destDir string, entries []zipEntryPlan, fileNames []string) error {
	if err := os.MkdirAll(destDir, 0755); err != nil {
		return err
	}

	for _, entry := range entries {
		if entry.isDir {
			if err := os.MkdirAll(entry.targetPath, 0755); err != nil {
				return err
			}
			continue
		}
		if err := os.MkdirAll(filepath.Dir(entry.targetPath), 0755); err != nil {
			return err
		}
	}

	if len(fileNames) == 0 {
		return nil
	}

	for i := 0; i < len(fileNames); i += unzipBatchSize {
		end := i + unzipBatchSize
		if end > len(fileNames) {
			end = len(fileNames)
		}

		args := make([]string, 0, 5+(end-i))
		args = append(args, "-qq", "-o", sourcePath)
		args = append(args, fileNames[i:end]...)
		args = append(args, "-d", destDir)

		cmd := exec.Command("unzip", args...)
		if output, err := cmd.CombinedOutput(); err != nil {
			message := strings.TrimSpace(string(output))
			if message == "" {
				return err
			}
			return fmt.Errorf("%w: %s", err, message)
		}
	}

	return nil
}

// extractZip writes each safe entry from reader into destDir while rejecting
// symlinks and path traversal attempts.
//
// Parameters:
// - reader: parsed zip archive reader
// - destDir: destination directory for extracted files
//
// Returns:
// - an error when any entry is unsafe or cannot be written
//
// Side effects:
// - creates directories and files under destDir
func extractZip(reader *zip.Reader, destDir string) error {
	entries, err := buildZipEntryPlans(reader, destDir)
	if err != nil {
		return err
	}

	return extractZipEntries(reader, entries)
}

// buildZipEntryPlans validates archive entries and resolves their destination
// paths.
//
// Parameters:
// - reader: parsed zip archive reader
// - destDir: destination directory for extracted files
//
// Returns:
// - validated entry plans
// - an error when an entry is unsafe
func buildZipEntryPlans(reader *zip.Reader, destDir string) ([]zipEntryPlan, error) {
	entries := make([]zipEntryPlan, 0, len(reader.File))
	for _, file := range reader.File {
		targetPath, err := secureJoin(destDir, file.Name)
		if err != nil {
			return nil, err
		}

		mode := file.Mode()
		if mode&os.ModeSymlink != 0 {
			return nil, fmt.Errorf("archive entry is a symlink: %s", file.Name)
		}

		fileMode := mode.Perm()
		if fileMode == 0 {
			fileMode = 0644
		}

		entries = append(entries, zipEntryPlan{
			entryName:  file.Name,
			targetPath: targetPath,
			mode:       fileMode,
			isDir:      file.FileInfo().IsDir(),
		})
	}

	return entries, nil
}

// extractZipEntries writes validated zip entries to disk.
//
// Parameters:
// - reader: parsed zip archive reader
// - entries: validated archive entries to extract
//
// Returns:
// - an error when any entry cannot be read or written
//
// Side effects:
// - creates directories and files for extracted entries
func extractZipEntries(reader *zip.Reader, entries []zipEntryPlan) error {
	zipFiles := make(map[string]*zip.File, len(reader.File))
	for _, file := range reader.File {
		zipFiles[file.Name] = file
	}

	for _, entry := range entries {
		if entry.isDir {
			if err := os.MkdirAll(entry.targetPath, 0755); err != nil {
				return err
			}
			continue
		}

		if err := os.MkdirAll(filepath.Dir(entry.targetPath), 0755); err != nil {
			return err
		}

		file := zipFiles[entry.entryName]
		if file == nil {
			return fmt.Errorf("archive entry disappeared during extraction: %s", entry.entryName)
		}

		input, err := file.Open()
		if err != nil {
			return err
		}

		output, err := os.OpenFile(entry.targetPath, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, entry.mode)
		if err != nil {
			_ = input.Close()
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
