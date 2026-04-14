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
	writeTestFile(t, filepath.Join(sourceRoot, "wordpress", "plugins", "backup-plugins.zip"), "plugins")
	writeTestFile(t, filepath.Join(sourceRoot, "wordpress", "database", "backup-db.gz"), "database")

	if err := Sync(sourceRoot, targetRoot); err != nil {
		t.Fatalf("Sync returned error: %v", err)
	}

	for _, expected := range []string{
		filepath.Join(targetRoot, "wp-cli.yml"),
		filepath.Join(targetRoot, "wordpress", "plugins", "backup-plugins.zip"),
		filepath.Join(targetRoot, "wordpress", "database", "backup-db.gz"),
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
	if len(manifest.Files) != 2 {
		t.Fatalf("expected 2 manifest entries, got %d", len(manifest.Files))
	}
}

func TestSyncRemovesStaleEmbeddedFiles(t *testing.T) {
	sourceRoot := filepath.Join(t.TempDir(), "templates")
	targetRoot := filepath.Join(t.TempDir(), "internal", "templates", "files")

	writeTestFile(t, filepath.Join(sourceRoot, "wp-cli.yml"), "path: app\n")
	writeTestFile(t, filepath.Join(sourceRoot, "wordpress", "plugins", "backup-plugins.zip"), "plugins")
	writeTestFile(t, filepath.Join(targetRoot, "stale.txt"), "stale")

	if err := Sync(sourceRoot, targetRoot); err != nil {
		t.Fatalf("Sync returned error: %v", err)
	}

	if _, err := os.Stat(filepath.Join(targetRoot, "stale.txt")); !os.IsNotExist(err) {
		t.Fatalf("expected stale.txt to be removed, got err=%v", err)
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
