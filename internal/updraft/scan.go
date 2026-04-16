package updraft

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

var categorizedDirs = map[string]struct{}{
	"database": {},
	"plugins":  {},
	"uploads":  {},
	"others":   {},
	"themes":   {},
}

type Selection struct {
	Root     string
	Database string
	Plugins  []string
	Uploads  []string
	Others   []string
	Themes   []string
}

func (s Selection) HasRecognizedFiles() bool {
	return s.Database != "" || len(s.Plugins) > 0 || len(s.Uploads) > 0 || len(s.Others) > 0 || len(s.Themes) > 0
}

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
		entryPath := filepath.Join(root, entry.Name())

		if entry.IsDir() {
			if _, ok := categorizedDirs[entry.Name()]; !ok {
				continue
			}

			matches, scanErr := scanCategorizedDir(root, entry.Name())
			if scanErr != nil {
				return Selection{}, scanErr
			}

			if matches.Database != "" {
				databaseFiles = append(databaseFiles, matches.Database)
			}
			pluginsFiles = append(pluginsFiles, matches.Plugins...)
			uploadsFiles = append(uploadsFiles, matches.Uploads...)
			othersFiles = append(othersFiles, matches.Others...)
			themesFiles = append(themesFiles, matches.Themes...)
			continue
		}

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
		Root:     root,
		Plugins:  pluginsFiles,
		Uploads:  uploadsFiles,
		Others:   othersFiles,
		Themes:   themesFiles,
	}
	if len(databaseFiles) == 1 {
		selection.Database = databaseFiles[0]
	}

	return selection, nil
}

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

func scanCategorizedDir(root string, category string) (Selection, error) {
	categoryRoot := filepath.Join(root, category)
	var matches []string

	err := filepath.WalkDir(categoryRoot, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}

		if matchesCategoryFile(category, d.Name()) {
			matches = append(matches, path)
			return nil
		}
		if looksLikeBackupArtifact(strings.ToLower(d.Name())) {
			relative, relErr := filepath.Rel(root, path)
			if relErr != nil {
				relative = path
			}
			return fmt.Errorf("unsupported backup file in %s: %s", category, relative)
		}
		return nil
	})
	if err != nil {
		return Selection{}, err
	}

	sort.Strings(matches)

	selection := Selection{Root: root}
	switch category {
	case "database":
		if len(matches) > 1 {
			return Selection{}, fmt.Errorf("expected exactly 1 database backup, found %d", len(matches))
		}
		if len(matches) == 1 {
			selection.Database = matches[0]
		}
	case "plugins":
		selection.Plugins = matches
	case "uploads":
		selection.Uploads = matches
	case "others":
		selection.Others = matches
	case "themes":
		selection.Themes = matches
	}

	return selection, nil
}

func matchesCategoryFile(category string, name string) bool {
	lower := strings.ToLower(name)
	switch category {
	case "database":
		return strings.HasSuffix(lower, ".sql") || strings.HasSuffix(lower, ".gz")
	default:
		return strings.HasSuffix(lower, ".zip")
	}
}

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

func looksLikeBackupArtifact(name string) bool {
	return strings.HasSuffix(name, ".zip") || strings.HasSuffix(name, ".sql") || strings.HasSuffix(name, ".gz")
}
