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
		"TOBA_PHP_VERSION=8.4\n" +
		"TOBA_DOMAIN=demo.lndo.site\n" +
		"TOBA_STARTER_REPO=git@example.com:company/starter.git\n" +
		"TOBA_SSH_TARGET=user@192.168.0.1 -p 22\n" +
		"TOBA_REMOTE_WORDPRESS_ROOT=www/example.com\n"

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

	if config.Name != "" || config.PHPVersion != "8.4" || config.Domain != "" || config.StarterRepo != "git@example.com:company/starter.git" || config.SSHTarget != "user@192.168.0.1 -p 22" || config.RemoteWordPressRoot != "www/example.com" {
		t.Fatalf("unexpected env config: %#v", config)
	}
}

func TestResolveGlobalEnvInitializationPrefersLocalEnvInRepo(t *testing.T) {
	repoDir := t.TempDir()
	configHome := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", configHome)

	if err := os.MkdirAll(filepath.Join(repoDir, "cmd"), 0755); err != nil {
		t.Fatalf("failed to create cmd dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(repoDir, "go.mod"), []byte("module github.com/gotcha190/toba\n"), 0644); err != nil {
		t.Fatalf("failed to write go.mod: %v", err)
	}
	if err := os.WriteFile(filepath.Join(repoDir, "cmd", "root.go"), []byte("package cmd\n"), 0644); err != nil {
		t.Fatalf("failed to write cmd/root.go: %v", err)
	}

	expectedContent := []byte("TOBA_STARTER_REPO=git@example.com:company/starter.git\n")
	if err := os.WriteFile(filepath.Join(repoDir, ".env"), expectedContent, 0644); err != nil {
		t.Fatalf("failed to write source .env: %v", err)
	}
	if err := os.WriteFile(filepath.Join(repoDir, ".env.example"), []byte("TOBA_STARTER_REPO=\n"), 0644); err != nil {
		t.Fatalf("failed to write source .env.example: %v", err)
	}

	content, sourcePath, targetPath, fromTemplate, err := ResolveGlobalEnvInitialization(repoDir)
	if err != nil {
		t.Fatalf("ResolveGlobalEnvInitialization returned error: %v", err)
	}

	if sourcePath != filepath.Join(repoDir, ".env") {
		t.Fatalf("unexpected source path: %s", sourcePath)
	}
	if fromTemplate {
		t.Fatal("expected local .env to be treated as real config")
	}

	if string(content) != string(expectedContent) {
		t.Fatalf("unexpected copied content: %s", string(content))
	}
	if targetPath != filepath.Join(configHome, globalConfigDirName, envFileName) {
		t.Fatalf("unexpected target path: %s", targetPath)
	}
}

func TestResolveGlobalEnvInitializationFallsBackToLocalEnvExampleInRepo(t *testing.T) {
	repoDir := t.TempDir()
	configHome := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", configHome)

	if err := os.MkdirAll(filepath.Join(repoDir, "cmd"), 0755); err != nil {
		t.Fatalf("failed to create cmd dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(repoDir, "go.mod"), []byte("module github.com/gotcha190/toba\n"), 0644); err != nil {
		t.Fatalf("failed to write go.mod: %v", err)
	}
	if err := os.WriteFile(filepath.Join(repoDir, "cmd", "root.go"), []byte("package cmd\n"), 0644); err != nil {
		t.Fatalf("failed to write cmd/root.go: %v", err)
	}

	expectedContent := []byte("TOBA_STARTER_REPO=\n")
	if err := os.WriteFile(filepath.Join(repoDir, ".env.example"), expectedContent, 0644); err != nil {
		t.Fatalf("failed to write source .env.example: %v", err)
	}

	content, sourcePath, targetPath, fromTemplate, err := ResolveGlobalEnvInitialization(repoDir)
	if err != nil {
		t.Fatalf("ResolveGlobalEnvInitialization returned error: %v", err)
	}

	if sourcePath != filepath.Join(repoDir, ".env.example") {
		t.Fatalf("unexpected source path: %s", sourcePath)
	}
	if !fromTemplate {
		t.Fatal("expected .env.example to be treated as template")
	}

	if string(content) != string(expectedContent) {
		t.Fatalf("unexpected copied content: %s", string(content))
	}
	if targetPath != filepath.Join(configHome, globalConfigDirName, envFileName) {
		t.Fatalf("unexpected target path: %s", targetPath)
	}
}

func TestResolveGlobalEnvInitializationUsesEmbeddedTemplateOutsideRepo(t *testing.T) {
	sourceDir := t.TempDir()
	configHome := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", configHome)

	content, sourcePath, targetPath, fromTemplate, err := ResolveGlobalEnvInitialization(sourceDir)
	if err != nil {
		t.Fatalf("ResolveGlobalEnvInitialization returned error: %v", err)
	}

	if sourcePath != "embedded:.env.example" {
		t.Fatalf("unexpected source path: %s", sourcePath)
	}
	if !fromTemplate {
		t.Fatal("expected embedded .env.example to be treated as template")
	}

	if string(content) != "TOBA_PHP_VERSION=\nTOBA_STARTER_REPO=\nTOBA_SSH_TARGET=\nTOBA_REMOTE_WORDPRESS_ROOT=\n" {
		t.Fatalf("unexpected embedded template content: %q", string(content))
	}
	if targetPath != filepath.Join(configHome, globalConfigDirName, envFileName) {
		t.Fatalf("unexpected target path: %s", targetPath)
	}
}

func TestWriteGlobalEnvWritesContent(t *testing.T) {
	configHome := t.TempDir()
	targetPath := filepath.Join(configHome, globalConfigDirName, envFileName)

	if err := WriteGlobalEnv(targetPath, []byte("TOBA_STARTER_REPO=git@example.com:company/starter.git\n")); err != nil {
		t.Fatalf("WriteGlobalEnv returned error: %v", err)
	}

	written, err := os.ReadFile(targetPath)
	if err != nil {
		t.Fatalf("failed to read written global env: %v", err)
	}
	if string(written) != "TOBA_STARTER_REPO=git@example.com:company/starter.git\n" {
		t.Fatalf("unexpected content: %q", string(written))
	}
}
