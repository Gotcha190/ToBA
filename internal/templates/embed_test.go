package templates

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestWordPressBackupFilesPreferOverrideSlot(t *testing.T) {
	configHome := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", configHome)

	overrideRoot := filepath.Join(configHome, "toba", "templates", "wordpress", "plugins")
	if err := os.MkdirAll(overrideRoot, 0755); err != nil {
		t.Fatalf("failed to create override root: %v", err)
	}
	overrideFile := filepath.Join(overrideRoot, "backup_2099-01-01-0000_demo_override-plugins.zip")
	if err := os.WriteFile(overrideFile, []byte("override"), 0644); err != nil {
		t.Fatalf("failed to write override file: %v", err)
	}

	files, err := WordPressBackupFiles("plugins", ".zip")
	if err != nil {
		t.Fatalf("WordPressBackupFiles returned error: %v", err)
	}

	var sawOverride bool
	for _, file := range files {
		if strings.Contains(file, "override:wordpress/plugins/backup_2099-01-01-0000_demo_override-plugins.zip") {
			sawOverride = true
		}
		if strings.Contains(file, "embedded:wordpress/plugins/") && strings.Contains(file, "-plugins.zip") {
			t.Fatalf("expected embedded plugins slot to be overridden, got %q", file)
		}
	}
	if !sawOverride {
		t.Fatalf("expected override plugins archive in %v", files)
	}
}

func TestReadUsesOverridePrefix(t *testing.T) {
	configHome := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", configHome)

	overrideFile := filepath.Join(configHome, "toba", "templates", "wordpress", "database", "override-db.gz")
	if err := os.MkdirAll(filepath.Dir(overrideFile), 0755); err != nil {
		t.Fatalf("failed to create override dir: %v", err)
	}
	if err := os.WriteFile(overrideFile, []byte("override"), 0644); err != nil {
		t.Fatalf("failed to write override file: %v", err)
	}

	content, err := Read("override:wordpress/database/override-db.gz")
	if err != nil {
		t.Fatalf("Read returned error: %v", err)
	}
	if string(content) != "override" {
		t.Fatalf("unexpected override content: %q", string(content))
	}
}
