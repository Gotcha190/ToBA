package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunUpdateSyncsTemplates(t *testing.T) {
	repoRoot := newUpdateRepoRoot(t)
	withWorkingDir(t, repoRoot)

	writeUpdateTestFile(t, filepath.Join(repoRoot, "templates", "wp-cli.yml"), "path: app\n")
	writeUpdateTestFile(t, filepath.Join(repoRoot, "templates", "wordpress", "backup_2026-04-14-0921_demo_1234-plugins.zip"), "plugins")
	writeUpdateTestFile(t, filepath.Join(repoRoot, "templates", "wordpress", "backup_2026-04-14-0921_demo_1234-db.gz"), "db")
	writeUpdateTestFile(t, filepath.Join(repoRoot, "templates", "wordpress", "backup_2026-04-14-0921_demo_1234-themes.zip"), "themes")

	if err := RunUpdate(); err != nil {
		t.Fatalf("RunUpdate returned error: %v", err)
	}

	for _, path := range []string{
		filepath.Join(repoRoot, "internal", "templates", "files", "wp-cli.yml"),
		filepath.Join(repoRoot, "internal", "templates", "files", "wordpress", "plugins", "backup_2026-04-14-0921_demo_1234-plugins.zip"),
		filepath.Join(repoRoot, "internal", "templates", "files", "wordpress", "database", "backup_2026-04-14-0921_demo_1234-db.gz"),
		filepath.Join(repoRoot, "internal", "templates", "files", "wordpress", "themes", "backup_2026-04-14-0921_demo_1234-themes.zip"),
		filepath.Join(repoRoot, "internal", "templates", "files", "wordpress", "DATA_VERSION"),
		filepath.Join(repoRoot, "internal", "templates", "files", "wordpress", "manifest.json"),
	} {
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("expected %s to exist: %v", path, err)
		}
	}
}

func TestRunUpdateRequiresToBARepoRoot(t *testing.T) {
	workingDir := t.TempDir()
	withWorkingDir(t, workingDir)

	err := RunUpdate()
	if err == nil {
		t.Fatal("expected RunUpdate to fail outside ToBA repo root")
	}
	if !strings.Contains(err.Error(), "run 'ToBA update' from the ToBA repository root") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func newUpdateRepoRoot(t *testing.T) string {
	t.Helper()

	repoRoot := t.TempDir()
	writeUpdateTestFile(t, filepath.Join(repoRoot, "go.mod"), "module github.com/gotcha190/ToBA\n")
	if err := os.MkdirAll(filepath.Join(repoRoot, "internal", "templates"), 0755); err != nil {
		t.Fatalf("failed to create internal/templates: %v", err)
	}

	return repoRoot
}

func writeUpdateTestFile(t *testing.T, path string, content string) {
	t.Helper()

	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatalf("failed to create parent dir for %s: %v", path, err)
	}
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write %s: %v", path, err)
	}
}
