package templates

import (
	"embed"
	"io/fs"
	"path"
	"strings"
)

//go:generate go run ../../cmd/sync-templates

//go:embed all:files
var files embed.FS

const (
	embeddedPrefix = "embedded:"
)

func Read(name string) ([]byte, error) {
	if strings.HasPrefix(name, embeddedPrefix) {
		return files.ReadFile(templatePath(strings.TrimPrefix(name, embeddedPrefix)))
	}

	return files.ReadFile(templatePath(name))
}

func List(dir string) ([]string, error) {
	return listEmbedded(dir)
}

func templatePath(name string) string {
	cleaned := path.Clean(strings.TrimPrefix(name, "/"))
	if cleaned == "." {
		return "files"
	}

	return path.Join("files", cleaned)
}
func listEmbedded(dir string) ([]string, error) {
	root := templatePath(dir)
	var entries []string

	err := fs.WalkDir(files, root, func(entryPath string, d fs.DirEntry, err error) error {
		if err != nil {
			if entryPath == root {
				return nil
			}
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
