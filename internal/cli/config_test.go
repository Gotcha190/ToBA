package cli

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gotcha190/ToBA/internal/create"
)

func TestRunConfigInitCopiesLocalEnvToGlobal(t *testing.T) {
	repoDir := t.TempDir()
	withWorkingDir(t, repoDir)

	content := []byte("TOBA_PROJECT_NAME=demo\nTOBA_STARTER_REPO=git@example.com:company/starter.git\n")
	if err := os.WriteFile(filepath.Join(repoDir, ".env"), content, 0644); err != nil {
		t.Fatalf("failed to write source env: %v", err)
	}

	if err := RunConfigInit(); err != nil {
		t.Fatalf("RunConfigInit returned error: %v", err)
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

func TestRunConfigInitLogsCopiedPaths(t *testing.T) {
	repoDir := t.TempDir()
	withWorkingDir(t, repoDir)

	content := []byte("TOBA_PROJECT_NAME=demo\n")
	if err := os.WriteFile(filepath.Join(repoDir, ".env"), content, 0644); err != nil {
		t.Fatalf("failed to write source env: %v", err)
	}

	stdout := os.Stdout
	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatalf("failed to create pipe: %v", err)
	}
	defer reader.Close()

	os.Stdout = writer
	runErr := RunConfigInit()
	writer.Close()
	os.Stdout = stdout

	if runErr != nil {
		t.Fatalf("RunConfigInit returned error: %v", runErr)
	}

	output, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("failed to read stdout: %v", err)
	}
	if !strings.Contains(string(output), "Copied config from") {
		t.Fatalf("expected copied config log, got %q", string(output))
	}
}
