package cli

import (
	"os"
	"path/filepath"
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
