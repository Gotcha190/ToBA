package main_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

var cliBinary string

func repoRoot() (string, error) {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		return "", os.ErrNotExist
	}

	return filepath.Dir(file), nil
}

func TestMain(m *testing.M) {
	root, err := repoRoot()
	if err != nil {
		os.Exit(1)
	}

	binDir, err := os.MkdirTemp("", "toba-e2e-*")
	if err != nil {
		os.Exit(1)
	}
	defer func() {
		_ = os.RemoveAll(binDir)
	}()

	binaryName := "toba-e2e"
	if runtime.GOOS == "windows" {
		binaryName += ".exe"
	}
	cliBinary = filepath.Join(binDir, binaryName)

	build := exec.Command("go", "build", "-o", cliBinary, ".")
	build.Dir = root
	build.Stdout = os.Stdout
	build.Stderr = os.Stderr
	if err := build.Run(); err != nil {
		os.Exit(1)
	}

	os.Exit(m.Run())
}

func runCLI(t *testing.T, env []string, args ...string) (string, error) {
	t.Helper()

	root, err := repoRoot()
	if err != nil {
		t.Fatal("failed to resolve repo root")
	}

	cmd := exec.Command(cliBinary, args...)
	cmd.Dir = root
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

	root, err := repoRoot()
	if err != nil {
		t.Fatal("failed to resolve repo root")
	}

	cmd := exec.Command(cliBinary, "create", "demo", "--dry-run")
	cmd.Dir = root
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
