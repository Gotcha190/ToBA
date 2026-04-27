package project

import (
	"archive/zip"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sync"
)

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
	fileNames    []string
	canRunDirect bool
}

// newZipExtractionPlan validates archive entries and prepares extraction
// metadata for one zip file.
//
// Parameters:
// - sourcePath: zip archive path used in error messages and direct extraction
// - reader: parsed zip archive reader
// - destDir: destination directory for extracted files
//
// Returns:
// - the prepared extraction plan
// - an error when any archive entry is unsafe
func newZipExtractionPlan(sourcePath string, reader *zip.Reader, destDir string) (zipExtractionPlan, error) {
	entries, err := buildZipEntryPlans(reader, destDir)
	if err != nil {
		return zipExtractionPlan{}, fmt.Errorf("inspect %s: %w", sourcePath, err)
	}

	targets := make(map[string]struct{}, len(entries))
	fileNames := make([]string, 0, len(entries))
	canRunDirect := true
	for _, entry := range entries {
		if entry.isDir {
			continue
		}

		fileNames = append(fileNames, entry.entryName)
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
		fileNames:    fileNames,
		canRunDirect: canRunDirect,
	}, nil
}

// extract runs this zip extraction plan.
//
// Parameters:
// - destDir: destination directory for extracted files
//
// Returns:
// - an error when extraction fails
//
// Side effects:
// - writes extracted directories and files under destDir
func (p zipExtractionPlan) extract(destDir string) error {
	if p.canRunDirect && canUseSystemUnzip() {
		if err := extractZipWithSystemUnzip(p.sourcePath, destDir, p.entries, p.fileNames); err != nil {
			return fmt.Errorf("extract %s: %w", p.sourcePath, err)
		}
		return nil
	}

	reader, err := zip.OpenReader(p.sourcePath)
	if err != nil {
		return fmt.Errorf("open %s: %w", p.sourcePath, err)
	}
	defer func() {
		_ = reader.Close()
	}()

	if err := extractZipEntries(&reader.Reader, p.entries); err != nil {
		return fmt.Errorf("extract %s: %w", p.sourcePath, err)
	}

	return nil
}

// zipPlansAreIndependent reports whether extraction plans write disjoint
// target files.
//
// Parameters:
// - plans: prepared zip extraction plans to compare
//
// Returns:
// - true when no two plans write the same file target
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

// extractZipPlansInParallel extracts independent zip plans using worker
// goroutines.
//
// Parameters:
// - plans: prepared zip extraction plans
// - destDir: destination directory for extracted files
//
// Returns:
// - an error when any plan extraction fails
//
// Side effects:
// - writes extracted directories and files under destDir
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
