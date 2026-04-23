package create

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestExecRunnerCaptureOutputUsesWorkingDirectory(t *testing.T) {
	runner := ExecRunner{}
	dir := t.TempDir()

	output, err := runner.CaptureOutput(dir, "pwd")
	if err != nil {
		t.Fatalf("CaptureOutput returned error: %v", err)
	}

	if got := strings.TrimSpace(output); got != filepath.Clean(dir) {
		t.Fatalf("expected pwd output %q, got %q", filepath.Clean(dir), got)
	}
}

func TestExecRunnerCaptureOutputDoesNotUseShellByDefault(t *testing.T) {
	runner := ExecRunner{}

	output, err := runner.CaptureOutput("", "printf", "%s", "$HOME")
	if err != nil {
		t.Fatalf("CaptureOutput returned error: %v", err)
	}

	if output != "$HOME" {
		t.Fatalf("expected literal shell token, got %q", output)
	}
}

func TestExecRunnerRunSupportsExplicitShellCommands(t *testing.T) {
	runner := ExecRunner{}
	dir := t.TempDir()
	targetPath := filepath.Join(dir, "shell.txt")

	if err := runner.Run("", "bash", "-lc", "printf '%s' \"$HOME\" > "+shellQuoteForTest(targetPath)); err != nil {
		t.Fatalf("Run returned error: %v", err)
	}

	content, err := os.ReadFile(targetPath)
	if err != nil {
		t.Fatalf("failed to read output file: %v", err)
	}
	if strings.TrimSpace(string(content)) == "$HOME" || len(strings.TrimSpace(string(content))) == 0 {
		t.Fatalf("expected shell-expanded HOME value, got %q", string(content))
	}
}

func TestExecRunnerRunCapturesCommandOutputInErrors(t *testing.T) {
	runner := ExecRunner{}

	err := runner.Run("", "bash", "-lc", "printf 'stdout-message\\n'; printf 'stderr-message\\n' >&2; exit 1")
	if err == nil {
		t.Fatal("expected command failure")
	}
	message := err.Error()
	if !strings.Contains(message, "stderr-message") {
		t.Fatalf("expected stderr output in error, got %q", message)
	}
}

func shellQuoteForTest(value string) string {
	return "'" + strings.ReplaceAll(value, "'", `'\''`) + "'"
}
