package create

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadEnvConfig(t *testing.T) {
	configHome := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", configHome)

	globalEnvPath, err := GlobalEnvPath()
	if err != nil {
		t.Fatalf("GlobalEnvPath returned error: %v", err)
	}

	content := "" +
		"TOBA_PROJECT_NAME=demo\n" +
		"TOBA_PHP_VERSION=8.4\n" +
		"TOBA_DOMAIN=demo.lndo.site\n" +
		"TOBA_STARTER_REPO=git@example.com:company/starter.git\n" +
		"TOBA_SSH_TARGET=user@192.168.0.1 -p 22\n"

	if err := os.MkdirAll(filepath.Dir(globalEnvPath), 0755); err != nil {
		t.Fatalf("failed to create config dir: %v", err)
	}
	if err := os.WriteFile(globalEnvPath, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write global env: %v", err)
	}

	config, err := LoadEnvConfig()
	if err != nil {
		t.Fatalf("LoadEnvConfig returned error: %v", err)
	}

	if config.Name != "demo" || config.PHPVersion != "8.4" || config.Domain != "" || config.StarterRepo != "git@example.com:company/starter.git" || config.SSHTarget != "user@192.168.0.1 -p 22" {
		t.Fatalf("unexpected env config: %#v", config)
	}
}

func TestCopyLocalEnvToGlobal(t *testing.T) {
	repoDir := t.TempDir()
	configHome := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", configHome)

	content := []byte("TOBA_PROJECT_NAME=demo\nTOBA_STARTER_REPO=git@example.com:company/starter.git\n")
	if err := os.WriteFile(filepath.Join(repoDir, ".env"), content, 0644); err != nil {
		t.Fatalf("failed to write source .env: %v", err)
	}

	sourcePath, targetPath, err := CopyLocalEnvToGlobal(repoDir)
	if err != nil {
		t.Fatalf("CopyLocalEnvToGlobal returned error: %v", err)
	}

	if sourcePath != filepath.Join(repoDir, ".env") {
		t.Fatalf("unexpected source path: %s", sourcePath)
	}

	written, err := os.ReadFile(targetPath)
	if err != nil {
		t.Fatalf("failed to read global env: %v", err)
	}
	if string(written) != string(content) {
		t.Fatalf("unexpected copied content: %s", string(written))
	}
}
