package create

import (
	"path/filepath"
	"testing"
)

func TestNewProjectPaths(t *testing.T) {
	baseDir := "/workspace"
	projectName := "demo"

	paths := NewProjectPaths(baseDir, projectName)

	if paths.BaseDir != baseDir {
		t.Fatalf("expected BaseDir %q, got %q", baseDir, paths.BaseDir)
	}
	if paths.Root != filepath.Join(baseDir, projectName) {
		t.Fatalf("unexpected Root: %q", paths.Root)
	}
	if paths.AppDir != filepath.Join(baseDir, projectName, "app") {
		t.Fatalf("unexpected AppDir: %q", paths.AppDir)
	}
	if paths.ConfigDir != filepath.Join(baseDir, projectName, "config") {
		t.Fatalf("unexpected ConfigDir: %q", paths.ConfigDir)
	}
	if paths.WPContent != filepath.Join(baseDir, projectName, "app", "wp-content") {
		t.Fatalf("unexpected WPContent: %q", paths.WPContent)
	}
	if paths.Plugins != filepath.Join(baseDir, projectName, "app", "wp-content", "plugins") {
		t.Fatalf("unexpected Plugins: %q", paths.Plugins)
	}
	if paths.Uploads != filepath.Join(baseDir, projectName, "app", "wp-content", "uploads") {
		t.Fatalf("unexpected Uploads: %q", paths.Uploads)
	}
	if paths.Themes != filepath.Join(baseDir, projectName, "app", "wp-content", "themes") {
		t.Fatalf("unexpected Themes: %q", paths.Themes)
	}
	if paths.DatabaseSQL != filepath.Join(baseDir, projectName, "app", "database.sql") {
		t.Fatalf("unexpected DatabaseSQL: %q", paths.DatabaseSQL)
	}
}
