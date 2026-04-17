package updraft

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestScanProjectDirSupportsLooseBackups(t *testing.T) {
	root := t.TempDir()

	writeTestFile(t, filepath.Join(root, "backup_2026-04-16_demo-db.gz"), "db")
	writeTestFile(t, filepath.Join(root, "backup_2026-04-16_demo-plugins.zip"), "plugins")
	writeTestFile(t, filepath.Join(root, "backup_2026-04-16_demo-uploads2.zip"), "uploads")
	writeTestFile(t, filepath.Join(root, "backup_2026-04-16_demo-themes.zip"), "themes")
	writeTestFile(t, filepath.Join(root, "backup_2026-04-16_demo-others.zip"), "others")

	selection, err := ScanProjectDir(root)
	if err != nil {
		t.Fatalf("ScanProjectDir returned error: %v", err)
	}

	if selection.Database == "" {
		t.Fatal("expected database backup to be detected")
	}
	if len(selection.Plugins) != 1 || len(selection.Uploads) != 1 || len(selection.Themes) != 1 || len(selection.Others) != 1 {
		t.Fatalf("unexpected selection: %#v", selection)
	}
	if err := selection.ValidateLocalProjectSet(); err != nil {
		t.Fatalf("ValidateLocalProjectSet returned error: %v", err)
	}
}

func TestScanProjectDirIgnoresCategorizedBackups(t *testing.T) {
	root := t.TempDir()

	writeTestFile(t, filepath.Join(root, "database", "backup-db.gz"), "db")
	writeTestFile(t, filepath.Join(root, "plugins", "plugins-a.zip"), "plugins")
	writeTestFile(t, filepath.Join(root, "uploads", "uploads-a.zip"), "uploads")
	writeTestFile(t, filepath.Join(root, "themes", "themes-a.zip"), "themes")
	writeTestFile(t, filepath.Join(root, "others", "others-a.zip"), "others")

	selection, err := ScanProjectDir(root)
	if err != nil {
		t.Fatalf("ScanProjectDir returned error: %v", err)
	}
	if selection.HasRecognizedFiles() {
		t.Fatalf("expected categorized backups to be ignored, got %#v", selection)
	}
}

func TestScanProjectDirRejectsMultipleDatabaseBackups(t *testing.T) {
	root := t.TempDir()

	writeTestFile(t, filepath.Join(root, "backup-db.gz"), "db1")
	writeTestFile(t, filepath.Join(root, "backup-db.sql"), "db2")

	_, err := ScanProjectDir(root)
	if err == nil {
		t.Fatal("expected multiple database backups to fail")
	}
	if !strings.Contains(err.Error(), "expected exactly 1 database backup") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestScanProjectDirRejectsUnsupportedLooseBackupFile(t *testing.T) {
	root := t.TempDir()

	writeTestFile(t, filepath.Join(root, "mystery.zip"), "broken")

	_, err := ScanProjectDir(root)
	if err == nil {
		t.Fatal("expected unsupported loose backup file to fail")
	}
	if !strings.Contains(err.Error(), "unsupported backup file") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateLocalProjectSetReportsMissingThemes(t *testing.T) {
	selection := Selection{
		Database: "/tmp/demo-db.gz",
		Plugins:  []string{"/tmp/plugins.zip"},
		Uploads:  []string{"/tmp/uploads.zip"},
	}

	err := selection.ValidateLocalProjectSet()
	if err == nil {
		t.Fatal("expected missing themes validation error")
	}
	if !strings.Contains(err.Error(), "themes") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func writeTestFile(t *testing.T, path string, content string) {
	t.Helper()

	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatalf("failed to create parent dir for %s: %v", path, err)
	}
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write %s: %v", path, err)
	}
}
