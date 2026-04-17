package create

import "path/filepath"

type ProjectPaths struct {
	// Core projektu
	BaseDir   string
	Root      string
	AppDir    string
	ConfigDir string

	// WordPress (wewnątrz app/)
	WPContent string
	Plugins   string
	Uploads   string
	Themes    string

	// Inne
	DatabaseSQL string
}

// NewProjectPaths builds the canonical project directory layout for the given
// base directory and project name.
//
// Parameters:
// - baseDir: parent directory in which the project root will live
// - projectName: normalized project slug
//
// Returns:
//   - a ProjectPaths value containing the derived root, app, config, and
//     WordPress subpaths
func NewProjectPaths(baseDir, projectName string) ProjectPaths {
	root := filepath.Join(baseDir, projectName)
	app := filepath.Join(root, "app")
	config := filepath.Join(root, "config")

	wpContent := filepath.Join(app, "wp-content")

	return ProjectPaths{
		BaseDir:   baseDir,
		Root:      root,
		AppDir:    app,
		ConfigDir: config,

		WPContent: wpContent,
		Plugins:   filepath.Join(wpContent, "plugins"),
		Uploads:   filepath.Join(wpContent, "uploads"),
		Themes:    filepath.Join(wpContent, "themes"),

		DatabaseSQL: filepath.Join(app, "database.sql"),
	}
}
