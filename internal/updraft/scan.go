package updraft

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type Selection struct {
	Root     string
	Database string
	Plugins  []string
	Uploads  []string
	Others   []string
	Themes   []string
}

// HasRecognizedFiles reports whether the selection contains at least one known
// local backup artifact.
//
// Parameters:
// - none
//
// Returns:
// - true when any recognized backup category is populated
func (s Selection) HasRecognizedFiles() bool {
	return s.Database != "" || len(s.Plugins) > 0 || len(s.Uploads) > 0 || len(s.Others) > 0 || len(s.Themes) > 0
}

// ValidateLocalProjectSet checks that all required backup categories are
// present in the selection.
//
// Parameters:
// - none
//
// Returns:
// - an error listing the missing categories when the local backup set is incomplete
func (s Selection) ValidateLocalProjectSet() error {
	var missing []string

	if s.Database == "" {
		missing = append(missing, "database")
	}
	if len(s.Plugins) == 0 {
		missing = append(missing, "plugins")
	}
	if len(s.Uploads) == 0 {
		missing = append(missing, "uploads")
	}
	if len(s.Themes) == 0 {
		missing = append(missing, "themes")
	}

	if len(missing) > 0 {
		return fmt.Errorf("missing required categories: %s", strings.Join(missing, ", "))
	}

	return nil
}

// ScanProjectDir inspects the root directory for loose Updraft backup files
// and groups them by category.
//
// Parameters:
// - root: project directory that should be scanned for loose backup files
//
// Returns:
// - the grouped local backup selection
// - an error when the directory cannot be read or contains unsupported backup artifacts
func ScanProjectDir(root string) (Selection, error) {
	entries, err := os.ReadDir(root)
	if err != nil {
		return Selection{}, err
	}

	var databaseFiles []string
	var pluginsFiles []string
	var uploadsFiles []string
	var othersFiles []string
	var themesFiles []string

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		entryPath := filepath.Join(root, entry.Name())

		category, ok, classifyErr := ClassifyLooseBackupFile(entry.Name())
		if classifyErr != nil {
			return Selection{}, fmt.Errorf("unsupported backup file in %s: %s", root, entry.Name())
		}
		if !ok {
			continue
		}

		switch category {
		case "database":
			databaseFiles = append(databaseFiles, entryPath)
		case "plugins":
			pluginsFiles = append(pluginsFiles, entryPath)
		case "uploads":
			uploadsFiles = append(uploadsFiles, entryPath)
		case "others":
			othersFiles = append(othersFiles, entryPath)
		case "themes":
			themesFiles = append(themesFiles, entryPath)
		}
	}

	sort.Strings(databaseFiles)
	sort.Strings(pluginsFiles)
	sort.Strings(uploadsFiles)
	sort.Strings(othersFiles)
	sort.Strings(themesFiles)

	if len(databaseFiles) > 1 {
		return Selection{}, fmt.Errorf("expected exactly 1 database backup, found %d", len(databaseFiles))
	}

	selection := Selection{
		Root:    root,
		Plugins: pluginsFiles,
		Uploads: uploadsFiles,
		Others:  othersFiles,
		Themes:  themesFiles,
	}
	if len(databaseFiles) == 1 {
		selection.Database = databaseFiles[0]
	}

	return selection, nil
}

// ClassifyLooseBackupFile identifies the backup category represented by name.
//
// Parameters:
// - name: backup filename to classify
//
// Returns:
// - the detected category name
// - whether the file matches a supported backup pattern
// - an error when the file looks like a backup artifact but uses an unsupported naming scheme
func ClassifyLooseBackupFile(name string) (string, bool, error) {
	lower := strings.ToLower(strings.TrimSpace(name))
	switch {
	case lower == "" || lower == ".gitkeep":
		return "", false, nil
	case strings.HasSuffix(lower, "-db.gz"), strings.HasSuffix(lower, "-db.sql"), lower == "db.sql", lower == "db.gz":
		return "database", true, nil
	case isNumberedArchive(lower, "plugins"):
		return "plugins", true, nil
	case isNumberedArchive(lower, "uploads"):
		return "uploads", true, nil
	case isNumberedArchive(lower, "others"):
		return "others", true, nil
	case isNumberedArchive(lower, "themes"):
		return "themes", true, nil
	case looksLikeBackupArtifact(lower):
		return "", false, fmt.Errorf("unsupported backup file: %s", name)
	default:
		return "", false, nil
	}
}

// isNumberedArchive reports whether name matches a category zip file with an
// optional numeric suffix.
//
// Parameters:
// - name: archive filename to inspect
// - suffix: expected archive category suffix such as plugins or uploads
//
// Returns:
// - true when the filename matches the expected numbered archive pattern
func isNumberedArchive(name string, suffix string) bool {
	if !strings.HasSuffix(name, ".zip") {
		return false
	}

	stem := strings.TrimSuffix(name, ".zip")
	idx := strings.LastIndex(stem, "-"+suffix)
	if idx == -1 {
		return false
	}

	numberSuffix := stem[idx+len(suffix)+1:]
	if numberSuffix == "" {
		return true
	}

	for _, r := range numberSuffix {
		if r < '0' || r > '9' {
			return false
		}
	}

	return true
}

// looksLikeBackupArtifact reports whether name resembles a supported backup
// artifact extension even if its exact category is unknown.
//
// Parameters:
// - name: filename to inspect
//
// Returns:
// - true when the filename uses a known backup artifact extension
func looksLikeBackupArtifact(name string) bool {
	return strings.HasSuffix(name, ".zip") || strings.HasSuffix(name, ".sql") || strings.HasSuffix(name, ".gz")
}
