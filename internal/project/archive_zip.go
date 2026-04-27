package project

import (
	"archive/zip"
	"bytes"
	"os"
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
	file, err := os.Open(sourcePath)
	if err != nil {
		return err
	}
	defer func() {
		_ = file.Close()
	}()

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
