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
	writeUpdateTestFile(t, filepath.Join(repoRoot, "templates", "wordpress", "plugins", "backup_2026-04-14-0921_demo_1234-plugins.zip"), "plugins")
	writeUpdateTestFile(t, filepath.Join(repoRoot, "templates", "wordpress", "database", "backup_2026-04-14-0921_demo_1234-db.gz"), "db")

	if err := RunUpdate(UpdateOptions{}); err != nil {
		t.Fatalf("RunUpdate returned error: %v", err)
	}

	for _, path := range []string{
		filepath.Join(repoRoot, "internal", "templates", "files", "wp-cli.yml"),
		filepath.Join(repoRoot, "internal", "templates", "files", "wordpress", "plugins", "backup_2026-04-14-0921_demo_1234-plugins.zip"),
		filepath.Join(repoRoot, "internal", "templates", "files", "wordpress", "DATA_VERSION"),
	} {
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("expected %s to exist: %v", path, err)
		}
	}
}

func TestRunUpdateLinkReplacesCategoryAndSyncsEmbedded(t *testing.T) {
	workingDir := t.TempDir()
	withWorkingDir(t, workingDir)

	configHome := filepath.Join(workingDir, ".config-home")
	if err := os.Setenv("XDG_CONFIG_HOME", configHome); err != nil {
		t.Fatalf("failed to set XDG_CONFIG_HOME: %v", err)
	}

	overrideRoot := filepath.Join(configHome, "toba", "templates")
	writeUpdateTestFile(t, filepath.Join(overrideRoot, "wordpress", "plugins", "backup_old-plugins.zip"), "old")
	writeUpdateTestFile(t, filepath.Join(overrideRoot, "wordpress", "plugins", "backup_old-plugins2.zip"), "keep")

	incomingDir := filepath.Join(workingDir, "incoming")
	writeUpdateTestFile(t, filepath.Join(incomingDir, "backup_2026-04-14-0921_demo_1234-plugins.zip"), "new")

	if err := RunUpdate(UpdateOptions{
		LinkPath: filepath.Join(incomingDir, "backup_2026-04-14-0921_demo_1234-plugins.zip"),
	}); err != nil {
		t.Fatalf("RunUpdate returned error: %v", err)
	}

	templateDir := filepath.Join(overrideRoot, "wordpress", "plugins")
	entries, err := os.ReadDir(templateDir)
	if err != nil {
		t.Fatalf("failed to read %s: %v", templateDir, err)
	}
	if len(entries) != 2 {
		t.Fatalf("unexpected templates/plugins contents: %#v", entries)
	}
	if entries[0].Name() != "backup_2026-04-14-0921_demo_1234-plugins.zip" || entries[1].Name() != "backup_old-plugins2.zip" {
		t.Fatalf("unexpected templates/plugins contents: %#v", entries)
	}

	content, err := os.ReadFile(filepath.Join(overrideRoot, "wordpress", "plugins", "backup_2026-04-14-0921_demo_1234-plugins.zip"))
	if err != nil {
		t.Fatalf("failed to read override plugin file: %v", err)
	}
	if string(content) != "new" {
		t.Fatalf("unexpected override content: %q", string(content))
	}
	kept, err := os.ReadFile(filepath.Join(overrideRoot, "wordpress", "plugins", "backup_old-plugins2.zip"))
	if err != nil {
		t.Fatalf("failed to read preserved override file: %v", err)
	}
	if string(kept) != "keep" {
		t.Fatalf("unexpected preserved override content: %q", string(kept))
	}
	for _, path := range []string{
		filepath.Join(overrideRoot, "wordpress", "DATA_VERSION"),
		filepath.Join(overrideRoot, "wordpress", "manifest.json"),
	} {
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("expected %s to exist: %v", path, err)
		}
	}
}

func TestRunUpdateLinkRejectsUnknownName(t *testing.T) {
	workingDir := t.TempDir()
	withWorkingDir(t, workingDir)

	writeUpdateTestFile(t, filepath.Join(workingDir, "incoming", "random.zip"), "bad")

	err := RunUpdate(UpdateOptions{
		LinkPath: filepath.Join(workingDir, "incoming", "random.zip"),
	})
	if err == nil {
		t.Fatal("expected RunUpdate to fail for unsupported file name")
	}
	if !strings.Contains(err.Error(), "unsupported Updraft backup name") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRunUpdateRequiresToBARepoRoot(t *testing.T) {
	workingDir := t.TempDir()
	withWorkingDir(t, workingDir)

	err := RunUpdate(UpdateOptions{})
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
