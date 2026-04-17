package cmd

import (
	"io"
	"os"
	"strings"
	"testing"

	"github.com/gotcha190/toba/internal/create"
)

func TestPrintUsageDoesNotMentionUpdate(t *testing.T) {
	stdout := os.Stdout
	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatalf("failed to create pipe: %v", err)
	}
	defer reader.Close()

	os.Stdout = writer
	printUsage()
	writer.Close()
	os.Stdout = stdout

	output, readErr := io.ReadAll(reader)
	if readErr != nil {
		t.Fatalf("failed to read usage output: %v", readErr)
	}

	if strings.Contains(string(output), "update") {
		t.Fatalf("usage should not mention update, got %q", string(output))
	}
}

func TestRunConfigWithoutInitCreatesGlobalTemplate(t *testing.T) {
	workDir := t.TempDir()
	original, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get working directory: %v", err)
	}
	if err := os.Chdir(workDir); err != nil {
		t.Fatalf("failed to chdir: %v", err)
	}
	t.Cleanup(func() {
		if err := os.Chdir(original); err != nil {
			t.Fatalf("failed to restore working directory: %v", err)
		}
	})

	configHome := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", configHome)

	if err := runConfig(nil); err != nil {
		t.Fatalf("runConfig returned error: %v", err)
	}

	globalEnvPath, err := create.GlobalEnvPath()
	if err != nil {
		t.Fatalf("GlobalEnvPath returned error: %v", err)
	}
	if _, err := os.Stat(globalEnvPath); err != nil {
		t.Fatalf("expected %s to exist: %v", globalEnvPath, err)
	}
}

func TestRunConfigRejectsLegacyInitArgument(t *testing.T) {
	err := runConfig([]string{"init"})
	if err == nil {
		t.Fatal("expected usage error")
	}
	if err.Error() != "usage: toba config" {
		t.Fatalf("unexpected error: %v", err)
	}
}
