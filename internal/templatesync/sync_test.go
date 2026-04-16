package templatesync

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSyncCopiesTemplatesAndWritesVersionedManifest(t *testing.T) {
	sourceRoot := filepath.Join(t.TempDir(), "templates")
	targetRoot := filepath.Join(t.TempDir(), "internal", "templates", "files")

	writeTestFile(t, filepath.Join(sourceRoot, "wp-cli.yml"), "path: app\n")
	writeTestFile(t, filepath.Join(sourceRoot, "wordpress", "backup_2026-04-14-0921_demo_1234-plugins.zip"), "plugins")
	writeTestFile(t, filepath.Join(sourceRoot, "wordpress", "backup_2026-04-14-0921_demo_1234-db.gz"), "database")
	writeTestFile(t, filepath.Join(sourceRoot, "wordpress", "backup_2026-04-14-0921_demo_1234-themes.zip"), "themes")

	if err := Sync(sourceRoot, targetRoot); err != nil {
		t.Fatalf("Sync returned error: %v", err)
	}

	for _, expected := range []string{
		filepath.Join(targetRoot, "wp-cli.yml"),
		filepath.Join(targetRoot, "wordpress", "plugins", "backup_2026-04-14-0921_demo_1234-plugins.zip"),
		filepath.Join(targetRoot, "wordpress", "database", "backup_2026-04-14-0921_demo_1234-db.gz"),
		filepath.Join(targetRoot, "wordpress", "themes", "backup_2026-04-14-0921_demo_1234-themes.zip"),
		filepath.Join(targetRoot, "wordpress", dataVersionFile),
		filepath.Join(targetRoot, "wordpress", manifestFile),
	} {
		if _, err := os.Stat(expected); err != nil {
			t.Fatalf("expected %s to exist: %v", expected, err)
		}
	}

	versionBytes, err := os.ReadFile(filepath.Join(targetRoot, "wordpress", dataVersionFile))
	if err != nil {
		t.Fatalf("failed to read DATA_VERSION: %v", err)
	}
	version := strings.TrimSpace(string(versionBytes))
	if len(version) != 12 {
		t.Fatalf("unexpected version: %q", version)
	}

	manifestBytes, err := os.ReadFile(filepath.Join(targetRoot, "wordpress", manifestFile))
	if err != nil {
		t.Fatalf("failed to read manifest: %v", err)
	}

	var manifest Manifest
	if err := json.Unmarshal(manifestBytes, &manifest); err != nil {
		t.Fatalf("failed to parse manifest: %v", err)
	}
	if manifest.Version != version {
		t.Fatalf("manifest version %q does not match DATA_VERSION %q", manifest.Version, version)
	}
	if len(manifest.Files) != 3 {
		t.Fatalf("expected 3 manifest entries, got %d", len(manifest.Files))
	}
}

func TestSyncRemovesStaleEmbeddedFiles(t *testing.T) {
	sourceRoot := filepath.Join(t.TempDir(), "templates")
	targetRoot := filepath.Join(t.TempDir(), "internal", "templates", "files")

	writeTestFile(t, filepath.Join(sourceRoot, "wp-cli.yml"), "path: app\n")
	writeTestFile(t, filepath.Join(sourceRoot, "wordpress", "backup_2026-04-14-0921_demo_1234-plugins.zip"), "plugins")
	writeTestFile(t, filepath.Join(targetRoot, "stale.txt"), "stale")

	if err := Sync(sourceRoot, targetRoot); err != nil {
		t.Fatalf("Sync returned error: %v", err)
	}

	if _, err := os.Stat(filepath.Join(targetRoot, "stale.txt")); !os.IsNotExist(err) {
		t.Fatalf("expected stale.txt to be removed, got err=%v", err)
	}
}

func TestSyncRejectsUnknownLooseWordPressBackup(t *testing.T) {
	sourceRoot := filepath.Join(t.TempDir(), "templates")
	targetRoot := filepath.Join(t.TempDir(), "internal", "templates", "files")

	writeTestFile(t, filepath.Join(sourceRoot, "wordpress", "mystery-backup.zip"), "broken")

	err := Sync(sourceRoot, targetRoot)
	if err == nil {
		t.Fatal("expected Sync to reject unknown loose wordpress backup")
	}
	if !strings.Contains(err.Error(), "unsupported wordpress backup file") {
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
