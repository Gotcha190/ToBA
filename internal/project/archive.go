package project

import (
	"archive/zip"
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
)

// ExtractZip expands a zip archive held in memory into destDir.
//
// Parameters:
// - data: raw zip archive bytes
// - destDir: destination directory for extracted files
//
// Returns:
// - an error when the archive cannot be read or extracted safely
//
// Side effects:
// - writes extracted files and directories to destDir
func ExtractZip(data []byte, destDir string) error {
	reader, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return err
	}

	return extractZip(reader, destDir)
}

// ExtractZipFile expands a zip archive from disk into destDir.
//
// Parameters:
// - sourcePath: zip archive path on disk
// - destDir: destination directory for extracted files
//
// Returns:
// - an error when the archive cannot be opened or extracted safely
//
// Side effects:
// - reads the archive from disk
// - writes extracted files and directories to destDir
func ExtractZipFile(sourcePath string, destDir string) error {
	reader, err := zip.OpenReader(sourcePath)
	if err != nil {
		return err
	}
	defer reader.Close()

	return extractZipPath(sourcePath, &reader.Reader, destDir)
}

// ExtractZipFiles expands multiple zip archives into destDir. Archives are
// extracted in parallel only when their target files do not overlap.
//
// Parameters:
// - sourcePaths: zip archive paths on disk
// - destDir: destination directory for extracted files
//
// Returns:
// - an error when any archive cannot be opened or extracted safely
//
// Side effects:
// - reads multiple archives from disk
// - writes extracted files and directories to destDir
func ExtractZipFiles(sourcePaths []string, destDir string) error {
	plans := make([]zipExtractionPlan, 0, len(sourcePaths))
	for _, sourcePath := range sourcePaths {
		if sourcePath == "" {
			continue
		}

		reader, err := zip.OpenReader(sourcePath)
		if err != nil {
			return err
		}

		plan, err := newZipExtractionPlan(sourcePath, &reader.Reader, destDir)
		closeErr := reader.Close()
		if err != nil {
			return err
		}
		if closeErr != nil {
			return closeErr
		}

		plans = append(plans, plan)
	}

	if len(plans) == 0 {
		return nil
	}

	if !zipPlansAreIndependent(plans) {
		for _, plan := range plans {
			if err := plan.extract(destDir); err != nil {
				return err
			}
		}
		return nil
	}

	return extractZipPlansInParallel(plans, destDir)
}

type zipEntryPlan struct {
	entryName  string
	targetPath string
	mode       os.FileMode
	isDir      bool
}

type zipExtractionPlan struct {
	sourcePath   string
	entries      []zipEntryPlan
	targets      map[string]struct{}
	canRunDirect bool
}

func newZipExtractionPlan(sourcePath string, reader *zip.Reader, destDir string) (zipExtractionPlan, error) {
	entries, err := buildZipEntryPlans(reader, destDir)
	if err != nil {
		return zipExtractionPlan{}, fmt.Errorf("inspect %s: %w", sourcePath, err)
	}

	targets := make(map[string]struct{}, len(entries))
	canRunDirect := true
	for _, entry := range entries {
		if entry.isDir {
			continue
		}

		normalizedTarget := filepath.Clean(entry.targetPath)
		if _, exists := targets[normalizedTarget]; exists {
			canRunDirect = false
		}
		targets[normalizedTarget] = struct{}{}
	}

	return zipExtractionPlan{
		sourcePath:   sourcePath,
		entries:      entries,
		targets:      targets,
		canRunDirect: canRunDirect,
	}, nil
}

func extractZipPath(sourcePath string, reader *zip.Reader, destDir string) error {
	plan, err := newZipExtractionPlan(sourcePath, reader, destDir)
	if err != nil {
		return err
	}

	return plan.extract(destDir)
}

func (p zipExtractionPlan) extract(destDir string) error {
	if p.canRunDirect && canUseSystemUnzip() {
		if err := extractZipWithSystemUnzip(p.sourcePath, destDir); err != nil {
			return fmt.Errorf("extract %s: %w", p.sourcePath, err)
		}
		return nil
	}

	reader, err := zip.OpenReader(p.sourcePath)
	if err != nil {
		return fmt.Errorf("open %s: %w", p.sourcePath, err)
	}
	defer reader.Close()

	if err := extractZipEntries(&reader.Reader, p.entries); err != nil {
		return fmt.Errorf("extract %s: %w", p.sourcePath, err)
	}

	return nil
}

func zipPlansAreIndependent(plans []zipExtractionPlan) bool {
	seen := make(map[string]struct{})
	for _, plan := range plans {
		for target := range plan.targets {
			if _, exists := seen[target]; exists {
				return false
			}
			seen[target] = struct{}{}
		}
	}

	return true
}

func extractZipPlansInParallel(plans []zipExtractionPlan, destDir string) error {
	workers := runtime.GOMAXPROCS(0)
	if workers < 1 {
		workers = 1
	}
	if workers > len(plans) {
		workers = len(plans)
	}

	work := make(chan zipExtractionPlan)
	errs := make(chan error, len(plans))

	var wg sync.WaitGroup
	for range workers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for plan := range work {
				if err := plan.extract(destDir); err != nil {
					errs <- err
				}
			}
		}()
	}

	for _, plan := range plans {
		work <- plan
	}
	close(work)

	wg.Wait()
	close(errs)

	for err := range errs {
		if err != nil {
			return err
		}
	}

	return nil
}

func canUseSystemUnzip() bool {
	_, err := exec.LookPath("unzip")
	return err == nil
}

func extractZipWithSystemUnzip(sourcePath string, destDir string) error {
	if err := os.MkdirAll(destDir, 0755); err != nil {
		return err
	}

	cmd := exec.Command("unzip", "-qq", "-o", sourcePath, "-d", destDir)
	if output, err := cmd.CombinedOutput(); err != nil {
		message := strings.TrimSpace(string(output))
		if message == "" {
			return err
		}
		return fmt.Errorf("%w: %s", err, message)
	}

	return nil
}

// extractZip validates zip entries and writes them into destDir while rejecting
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
	defer reader.Close()

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
	defer source.Close()

	reader, err := gzip.NewReader(source)
	if err != nil {
		return err
	}
	defer reader.Close()

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
	defer output.Close()

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
