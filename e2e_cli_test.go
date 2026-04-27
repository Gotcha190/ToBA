package main_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func repoRoot(t *testing.T) string {
	t.Helper()

	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("failed to resolve test file path")
	}

	return filepath.Dir(file)
}

func runCLI(t *testing.T, env []string, args ...string) (string, error) {
	t.Helper()

	cmd := exec.Command("go", append([]string{"run", "."}, args...)...)
	cmd.Dir = repoRoot(t)
	cmd.Env = append(os.Environ(), env...)
	output, err := cmd.CombinedOutput()
	return string(output), err
}

func TestCLIEndToEndVersion(t *testing.T) {
	output, err := runCLI(t, nil, "version")
	if err != nil {
		t.Fatalf("expected version command to succeed, got %v with output %q", err, output)
	}

	if !strings.Contains(output, "toba version: ") {
		t.Fatalf("expected version output, got %q", output)
	}
}

func TestCLIEndToEndDoctor(t *testing.T) {
	binDir := t.TempDir()
	for _, name := range []string{"git", "node", "npm", "lando", "docker", "ssh", "scp", "zip"} {
		path := filepath.Join(binDir, name)
		if err := os.WriteFile(path, []byte("#!/usr/bin/env sh\nexit 0\n"), 0755); err != nil {
			t.Fatalf("failed to write fake binary %s: %v", name, err)
		}
	}

	output, err := runCLI(t, []string{"PATH=" + binDir}, "doctor")
	if err != nil {
		t.Fatalf("expected doctor command to succeed, got %v with output %q", err, output)
	}

	for _, name := range []string{"Git", "Node", "NPM", "Lando", "Docker", "SSH", "SCP", "Zip"} {
		if !strings.Contains(output, "[OK] "+name+" installed") {
			t.Fatalf("expected doctor output for %s, got %q", name, output)
		}
	}
}

func TestCLIEndToEndCreateDryRun(t *testing.T) {
	workDir := t.TempDir()
	configHome := t.TempDir()
	configDir := filepath.Join(configHome, "toba")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Fatalf("failed to create config dir: %v", err)
	}

	globalEnv := strings.Join([]string{
		"TOBA_PHP_VERSION=8.4",
		"TOBA_STARTER_REPO=git@example.com:company/starter.git",
		"TOBA_SSH_TARGET=user@example.com -p 22",
		"TOBA_REMOTE_WORDPRESS_ROOT=www/example.com",
		"",
	}, "\n")
	if err := os.WriteFile(filepath.Join(configDir, ".env"), []byte(globalEnv), 0644); err != nil {
		t.Fatalf("failed to write global env: %v", err)
	}

	cmd := exec.Command("go", "run", ".", "create", "demo", "--dry-run")
	cmd.Dir = repoRoot(t)
	cmd.Env = append(os.Environ(), "XDG_CONFIG_HOME="+configHome, "HOME="+workDir)
	outputBytes, err := cmd.CombinedOutput()
	output := string(outputBytes)
	if err != nil {
		t.Fatalf("expected create dry-run to succeed, got %v with output %q", err, output)
	}

	if !strings.Contains(output, "[OK] Project ready: https://demo.lndo.site") {
		t.Fatalf("expected final project URL in output, got %q", output)
	}

	if _, statErr := os.Stat(filepath.Join(workDir, "demo")); !os.IsNotExist(statErr) {
		t.Fatalf("expected dry-run not to create project dir, got stat err %v", statErr)
	}
}
