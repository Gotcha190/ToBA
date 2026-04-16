package templatesync

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSyncCopiesStaticTemplatesOnly(t *testing.T) {
	sourceRoot := filepath.Join(t.TempDir(), "templates")
	targetRoot := filepath.Join(t.TempDir(), "internal", "templates", "files")

	writeTestFile(t, filepath.Join(sourceRoot, "wp-cli.yml"), "path: app\n")
	writeTestFile(t, filepath.Join(sourceRoot, "config", "php.ini"), "memory_limit=512M\n")
	writeTestFile(t, filepath.Join(sourceRoot, "wordpress", "backup_2026-04-14-0921_demo_1234-plugins.zip"), "plugins")

	if err := Sync(sourceRoot, targetRoot); err != nil {
		t.Fatalf("Sync returned error: %v", err)
	}

	for _, expected := range []string{
		filepath.Join(targetRoot, "wp-cli.yml"),
		filepath.Join(targetRoot, "config", "php.ini"),
	} {
		if _, err := os.Stat(expected); err != nil {
			t.Fatalf("expected %s to exist: %v", expected, err)
		}
	}

	if _, err := os.Stat(filepath.Join(targetRoot, "wordpress")); !os.IsNotExist(err) {
		t.Fatalf("expected wordpress backups to be ignored, got err=%v", err)
	}
}

func TestSyncRemovesStaleEmbeddedFiles(t *testing.T) {
	sourceRoot := filepath.Join(t.TempDir(), "templates")
	targetRoot := filepath.Join(t.TempDir(), "internal", "templates", "files")

	writeTestFile(t, filepath.Join(sourceRoot, "wp-cli.yml"), "path: app\n")
	writeTestFile(t, filepath.Join(targetRoot, "stale.txt"), "stale")
	writeTestFile(t, filepath.Join(targetRoot, "wordpress", "stale.zip"), "stale")

	if err := Sync(sourceRoot, targetRoot); err != nil {
		t.Fatalf("Sync returned error: %v", err)
	}

	if _, err := os.Stat(filepath.Join(targetRoot, "stale.txt")); !os.IsNotExist(err) {
		t.Fatalf("expected stale.txt to be removed, got err=%v", err)
	}
	if _, err := os.Stat(filepath.Join(targetRoot, "wordpress")); !os.IsNotExist(err) {
		t.Fatalf("expected stale wordpress directory to be removed, got err=%v", err)
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
