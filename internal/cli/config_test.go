package cli

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gotcha190/toba/internal/create"
)

func TestRunConfigCopiesLocalEnvFromRepo(t *testing.T) {
	repoDir := t.TempDir()
	withWorkingDir(t, repoDir)

	if err := os.MkdirAll(filepath.Join(repoDir, "cmd"), 0755); err != nil {
		t.Fatalf("failed to create cmd dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(repoDir, "go.mod"), []byte("module github.com/gotcha190/toba\n"), 0644); err != nil {
		t.Fatalf("failed to write go.mod: %v", err)
	}
	if err := os.WriteFile(filepath.Join(repoDir, "cmd", "root.go"), []byte("package cmd\n"), 0644); err != nil {
		t.Fatalf("failed to write cmd/root.go: %v", err)
	}

	content := []byte("TOBA_STARTER_REPO=git@example.com:company/starter.git\n")
	if err := os.WriteFile(filepath.Join(repoDir, ".env"), content, 0644); err != nil {
		t.Fatalf("failed to write source env: %v", err)
	}

	if err := RunConfig(); err != nil {
		t.Fatalf("RunConfig returned error: %v", err)
	}

	globalEnvPath, err := create.GlobalEnvPath()
	if err != nil {
		t.Fatalf("GlobalEnvPath returned error: %v", err)
	}

	written, err := os.ReadFile(globalEnvPath)
	if err != nil {
		t.Fatalf("failed to read global env: %v", err)
	}
	if string(written) != string(content) {
		t.Fatalf("unexpected copied content: %s", string(written))
	}
}

func TestRunConfigCreatesTemplateFromEmbeddedEnvExampleOutsideRepo(t *testing.T) {
	workDir := t.TempDir()
	withWorkingDir(t, workDir)

	if err := RunConfig(); err != nil {
		t.Fatalf("RunConfig returned error: %v", err)
	}

	globalEnvPath, err := create.GlobalEnvPath()
	if err != nil {
		t.Fatalf("GlobalEnvPath returned error: %v", err)
	}

	written, err := os.ReadFile(globalEnvPath)
	if err != nil {
		t.Fatalf("failed to read global env: %v", err)
	}
	if string(written) != "TOBA_PHP_VERSION=\nTOBA_STARTER_REPO=\nTOBA_SSH_TARGET=\nTOBA_REMOTE_WORDPRESS_ROOT=\n" {
		t.Fatalf("unexpected template content: %q", string(written))
	}
}

func TestRunConfigLogsTemplatePathWhenUsingExample(t *testing.T) {
	workDir := t.TempDir()
	withWorkingDir(t, workDir)

	stdout := os.Stdout
	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatalf("failed to create pipe: %v", err)
	}
	defer func() {
		_ = reader.Close()
	}()

	os.Stdout = writer
	runErr := RunConfig()
	_ = writer.Close()
	os.Stdout = stdout

	if runErr != nil {
		t.Fatalf("RunConfig returned error: %v", runErr)
	}

	output, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("failed to read stdout: %v", err)
	}

	globalEnvPath, err := create.GlobalEnvPath()
	if err != nil {
		t.Fatalf("GlobalEnvPath returned error: %v", err)
	}

	if !strings.Contains(string(output), "Created global config from embedded .env.example: "+globalEnvPath) {
		t.Fatalf("expected embedded template log, got %q", string(output))
	}
	if !strings.Contains(string(output), "Fill in the required values in "+globalEnvPath) {
		t.Fatalf("expected fill-in log, got %q", string(output))
	}
}

func TestRunConfigPromptsBeforeOverwritingExistingGlobalConfig(t *testing.T) {
	repoDir := t.TempDir()
	withWorkingDir(t, repoDir)

	if err := os.MkdirAll(filepath.Join(repoDir, "cmd"), 0755); err != nil {
		t.Fatalf("failed to create cmd dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(repoDir, "go.mod"), []byte("module github.com/gotcha190/toba\n"), 0644); err != nil {
		t.Fatalf("failed to write go.mod: %v", err)
	}
	if err := os.WriteFile(filepath.Join(repoDir, "cmd", "root.go"), []byte("package cmd\n"), 0644); err != nil {
		t.Fatalf("failed to write cmd/root.go: %v", err)
	}

	sourceContent := []byte("TOBA_STARTER_REPO=git@example.com:company/starter.git\n")
	if err := os.WriteFile(filepath.Join(repoDir, ".env"), sourceContent, 0644); err != nil {
		t.Fatalf("failed to write source env: %v", err)
	}

	globalEnvPath, err := create.GlobalEnvPath()
	if err != nil {
		t.Fatalf("GlobalEnvPath returned error: %v", err)
	}
	if err := os.MkdirAll(filepath.Dir(globalEnvPath), 0755); err != nil {
		t.Fatalf("failed to create global config dir: %v", err)
	}
	if err := os.WriteFile(globalEnvPath, []byte("TOBA_STARTER_REPO=old\n"), 0644); err != nil {
		t.Fatalf("failed to write existing global env: %v", err)
	}

	output := &strings.Builder{}
	if err := runConfigWithIO(strings.NewReader("n\n"), output); err != nil {
		t.Fatalf("runConfigWithIO returned error: %v", err)
	}

	written, err := os.ReadFile(globalEnvPath)
	if err != nil {
		t.Fatalf("failed to read global env: %v", err)
	}
	if string(written) != "TOBA_STARTER_REPO=old\n" {
		t.Fatalf("expected global env to stay unchanged, got %q", string(written))
	}
	if !strings.Contains(output.String(), "Overwrite existing global config at "+globalEnvPath+"? [y/N]: ") {
		t.Fatalf("expected overwrite prompt, got %q", output.String())
	}
	if !strings.Contains(output.String(), "Skipped updating global config: "+globalEnvPath) {
		t.Fatalf("expected skip log, got %q", output.String())
	}
}

func TestRunConfigInformsUserWhenOverwriteConfirmationIsMissing(t *testing.T) {
	repoDir := t.TempDir()
	withWorkingDir(t, repoDir)

	if err := os.MkdirAll(filepath.Join(repoDir, "cmd"), 0755); err != nil {
		t.Fatalf("failed to create cmd dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(repoDir, "go.mod"), []byte("module github.com/gotcha190/toba\n"), 0644); err != nil {
		t.Fatalf("failed to write go.mod: %v", err)
	}
	if err := os.WriteFile(filepath.Join(repoDir, "cmd", "root.go"), []byte("package cmd\n"), 0644); err != nil {
		t.Fatalf("failed to write cmd/root.go: %v", err)
	}

	if err := os.WriteFile(filepath.Join(repoDir, ".env"), []byte("TOBA_STARTER_REPO=git@example.com:company/starter.git\n"), 0644); err != nil {
		t.Fatalf("failed to write source env: %v", err)
	}

	globalEnvPath, err := create.GlobalEnvPath()
	if err != nil {
		t.Fatalf("GlobalEnvPath returned error: %v", err)
	}
	if err := os.MkdirAll(filepath.Dir(globalEnvPath), 0755); err != nil {
		t.Fatalf("failed to create global config dir: %v", err)
	}
	if err := os.WriteFile(globalEnvPath, []byte("TOBA_STARTER_REPO=old\n"), 0644); err != nil {
		t.Fatalf("failed to write existing global env: %v", err)
	}

	output := &strings.Builder{}
	if err := runConfigWithIO(strings.NewReader(""), output); err != nil {
		t.Fatalf("runConfigWithIO returned error: %v", err)
	}

	if !strings.Contains(output.String(), "No confirmation received; skipped updating global config: "+globalEnvPath) {
		t.Fatalf("expected missing-confirmation log, got %q", output.String())
	}
}

func TestRunConfigOverwritesExistingGlobalConfigWhenConfirmed(t *testing.T) {
	repoDir := t.TempDir()
	withWorkingDir(t, repoDir)

	if err := os.MkdirAll(filepath.Join(repoDir, "cmd"), 0755); err != nil {
		t.Fatalf("failed to create cmd dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(repoDir, "go.mod"), []byte("module github.com/gotcha190/toba\n"), 0644); err != nil {
		t.Fatalf("failed to write go.mod: %v", err)
	}
	if err := os.WriteFile(filepath.Join(repoDir, "cmd", "root.go"), []byte("package cmd\n"), 0644); err != nil {
		t.Fatalf("failed to write cmd/root.go: %v", err)
	}

	sourceContent := []byte("TOBA_STARTER_REPO=git@example.com:company/starter.git\n")
	if err := os.WriteFile(filepath.Join(repoDir, ".env"), sourceContent, 0644); err != nil {
		t.Fatalf("failed to write source env: %v", err)
	}

	globalEnvPath, err := create.GlobalEnvPath()
	if err != nil {
		t.Fatalf("GlobalEnvPath returned error: %v", err)
	}
	if err := os.MkdirAll(filepath.Dir(globalEnvPath), 0755); err != nil {
		t.Fatalf("failed to create global config dir: %v", err)
	}
	if err := os.WriteFile(globalEnvPath, []byte("TOBA_STARTER_REPO=old\n"), 0644); err != nil {
		t.Fatalf("failed to write existing global env: %v", err)
	}

	output := &strings.Builder{}
	if err := runConfigWithIO(strings.NewReader("y\n"), output); err != nil {
		t.Fatalf("runConfigWithIO returned error: %v", err)
	}

	written, err := os.ReadFile(globalEnvPath)
	if err != nil {
		t.Fatalf("failed to read global env: %v", err)
	}
	if string(written) != string(sourceContent) {
		t.Fatalf("expected global env to be overwritten, got %q", string(written))
	}
	if !strings.Contains(output.String(), "Copied config from "+filepath.Join(repoDir, ".env")+" to "+globalEnvPath) {
		t.Fatalf("expected copy log, got %q", output.String())
	}
}

func TestRunConfigReturnsFriendlyPermissionErrorWhenGlobalConfigCannotBeWritten(t *testing.T) {
	workDir := t.TempDir()
	withWorkingDir(t, workDir)

	globalEnvPath, err := create.GlobalEnvPath()
	if err != nil {
		t.Fatalf("GlobalEnvPath returned error: %v", err)
	}
	globalConfigDir := filepath.Dir(globalEnvPath)
	if err := os.MkdirAll(globalConfigDir, 0755); err != nil {
		t.Fatalf("failed to create global config dir: %v", err)
	}
	if err := os.Chmod(globalConfigDir, 0500); err != nil {
		t.Fatalf("failed to chmod global config dir: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chmod(globalConfigDir, 0755)
	})

	err = runConfigWithIO(strings.NewReader("y\n"), io.Discard)
	if err == nil {
		t.Fatal("expected permission error")
	}
	if !strings.Contains(err.Error(), "cannot write global config at "+globalEnvPath+": permission denied") {
		t.Fatalf("unexpected error: %v", err)
	}
}
