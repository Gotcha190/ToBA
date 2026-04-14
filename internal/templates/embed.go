package templates

import (
	"embed"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

//go:generate go run ../../cmd/sync-templates

//go:embed all:files
var files embed.FS

const (
	overridePrefix = "override:"
	embeddedPrefix = "embedded:"
	configDirName  = "toba"
)

var backupSlotPattern = regexp.MustCompile(`-(db|plugins\d*|uploads\d*|others\d*)\.(zip|gz)$`)

func Read(name string) ([]byte, error) {
	if strings.HasPrefix(name, overridePrefix) {
		return os.ReadFile(overridePath(strings.TrimPrefix(name, overridePrefix)))
	}
	if strings.HasPrefix(name, embeddedPrefix) {
		return files.ReadFile(templatePath(strings.TrimPrefix(name, embeddedPrefix)))
	}

	if content, err := os.ReadFile(overridePath(name)); err == nil {
		return content, nil
	} else if !os.IsNotExist(err) {
		return nil, err
	}

	return files.ReadFile(templatePath(name))
}

func List(dir string) ([]string, error) {
	entries, err := listEmbedded(dir)
	if err != nil {
		return nil, err
	}

	for _, override := range listOverrides(dir) {
		if !contains(entries, override) {
			entries = append(entries, override)
		}
	}

	sort.Strings(entries)
	return entries, nil
}

func templatePath(name string) string {
	cleaned := path.Clean(strings.TrimPrefix(name, "/"))
	if cleaned == "." {
		return "files"
	}

	return path.Join("files", cleaned)
}

func WordPressDataVersion() (string, error) {
	content, err := Read("wordpress/DATA_VERSION")
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(string(content)), nil
}

func WordPressBackupFiles(category string, suffix string) ([]string, error) {
	relativeDir := path.Join("wordpress", category)
	embeddedFiles, err := listEmbedded(relativeDir)
	if err != nil {
		return nil, err
	}

	overrideFiles := listOverrides(relativeDir)
	selected := map[string]string{}
	order := map[string]string{}

	for _, file := range embeddedFiles {
		if !strings.HasSuffix(strings.ToLower(file), strings.ToLower(suffix)) {
			continue
		}
		key := backupKey(file)
		selected[key] = embeddedPrefix + file
		order[key] = file
	}

	for _, file := range overrideFiles {
		if !strings.HasSuffix(strings.ToLower(file), strings.ToLower(suffix)) {
			continue
		}
		key := backupKey(file)
		selected[key] = overridePrefix + file
		order[key] = file
	}

	var keys []string
	for key := range selected {
		keys = append(keys, key)
	}
	sort.Slice(keys, func(i, j int) bool {
		return order[keys[i]] < order[keys[j]]
	})

	var resolved []string
	for _, key := range keys {
		resolved = append(resolved, selected[key])
	}

	return resolved, nil
}

func listEmbedded(dir string) ([]string, error) {
	root := templatePath(dir)
	var entries []string

	err := fs.WalkDir(files, root, func(entryPath string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}

		entries = append(entries, strings.TrimPrefix(entryPath, "files/"))
		return nil
	})
	if err != nil {
		return nil, err
	}

	return entries, nil
}

func listOverrides(dir string) []string {
	root := overridePath(dir)
	var entries []string

	err := filepath.WalkDir(root, func(entryPath string, d os.DirEntry, err error) error {
		if err != nil {
			if os.IsNotExist(err) {
				return filepath.SkipDir
			}
			return err
		}
		if d.IsDir() {
			return nil
		}

		relative, err := filepath.Rel(overridePath(""), entryPath)
		if err != nil {
			return err
		}
		entries = append(entries, filepath.ToSlash(relative))
		return nil
	})
	if err != nil {
		return nil
	}

	sort.Strings(entries)
	return entries
}

func overridePath(name string) string {
	root, err := overrideRoot()
	if err != nil {
		return filepath.Join(name)
	}
	if name == "" {
		return root
	}

	return filepath.Join(root, filepath.FromSlash(name))
}

func overrideRoot() (string, error) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}

	return filepath.Join(configDir, configDirName, "templates"), nil
}

func backupKey(name string) string {
	match := backupSlotPattern.FindStringSubmatch(strings.ToLower(path.Base(name)))
	if match == nil {
		return name
	}

	return match[1]
}

func contains(values []string, needle string) bool {
	for _, value := range values {
		if value == needle {
			return true
		}
	}

	return false
}
