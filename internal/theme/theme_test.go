package theme

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"testing"
)

const testStarterRepo = "git@example.com:company/starter.git"

type recordedCommand struct {
	dir  string
	cmd  string
	args []string
}

type fakeRunner struct {
	mu       sync.Mutex
	commands []recordedCommand
	err      error
}

func (r *fakeRunner) Run(dir string, cmd string, args ...string) error {
	r.mu.Lock()
	r.commands = append(r.commands, recordedCommand{
		dir:  dir,
		cmd:  cmd,
		args: append([]string(nil), args...),
	})
	r.mu.Unlock()
	if cmd == "git" && len(args) == 3 && args[0] == "clone" {
		if err := os.MkdirAll(filepath.Join(dir, args[2]), 0755); err != nil {
			return err
		}
	}
	return r.err
}

func (r *fakeRunner) CaptureOutput(dir string, cmd string, args ...string) (string, error) {
	return "", r.err
}

func TestInstallReturnsMissingRepoError(t *testing.T) {
	_, err := Install(&fakeRunner{}, t.TempDir(), "", "demo")
	if err == nil {
		t.Fatal("expected missing starter repo error")
	}
	if !strings.Contains(err.Error(), "TOBA_STARTER_REPO") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestInstallRunsCloneCommand(t *testing.T) {
	themesDir := t.TempDir()
	runner := &fakeRunner{}

	targetDir, err := Install(runner, themesDir, testStarterRepo, "demo")
	if err != nil {
		t.Fatalf("Install returned error: %v", err)
	}

	expectedTarget := filepath.Join(themesDir, "demo")
	if targetDir != expectedTarget {
		t.Fatalf("expected target dir %q, got %q", expectedTarget, targetDir)
	}

	expected := []recordedCommand{
		{
			dir:  themesDir,
			cmd:  "git",
			args: []string{"clone", testStarterRepo, "demo"},
		},
	}

	if !reflect.DeepEqual(runner.commands, expected) {
		t.Fatalf("unexpected commands:\nexpected: %#v\ngot: %#v", expected, runner.commands)
	}
}

func TestBuildRunsExpectedCommands(t *testing.T) {
	runner := &fakeRunner{}
	themeDir := "/tmp/demo-theme"

	err := Build(runner, themeDir)
	if err != nil {
		t.Fatalf("Build returned error: %v", err)
	}

	expectedCommands := []recordedCommand{
		{
			dir:  themeDir,
			cmd:  "lando",
			args: []string{"composer", "install", "--no-interaction", "--prefer-dist", "--optimize-autoloader", "--no-progress"},
		},
		{
			dir:  themeDir,
			cmd:  "npm",
			args: []string{"ci", "--no-audit", "--no-fund"},
		},
	}

	if len(runner.commands) != 3 {
		t.Fatalf("expected 3 commands, got %#v", runner.commands)
	}

	for _, expectedCommand := range expectedCommands {
		if !containsRecordedCommand(runner.commands[:2], expectedCommand) {
			t.Fatalf("missing install command %#v in %#v", expectedCommand, runner.commands)
		}
	}

	expectedBuild := recordedCommand{
		dir:  themeDir,
		cmd:  "npm",
		args: []string{"run", "build"},
	}
	if !reflect.DeepEqual(runner.commands[2], expectedBuild) {
		t.Fatalf("unexpected build command:\nexpected: %#v\ngot: %#v", expectedBuild, runner.commands[2])
	}
}

func TestBuildReturnsCommandError(t *testing.T) {
	expectedErr := errors.New("boom")
	runner := &fakeRunner{err: expectedErr}

	err := Build(runner, "/tmp/demo-theme")
	if !errors.Is(err, expectedErr) {
		t.Fatalf("expected %v, got %v", expectedErr, err)
	}
}

func TestGenerateAcornKeyRunsTwice(t *testing.T) {
	runner := &fakeRunner{}

	err := GenerateAcornKey(runner, "/tmp/demo")
	if err != nil {
		t.Fatalf("GenerateAcornKey returned error: %v", err)
	}

	expected := []recordedCommand{
		{
			dir:  "/tmp/demo",
			cmd:  "lando",
			args: []string{"wp", "acorn", "key:generate"},
		},
		{
			dir:  "/tmp/demo",
			cmd:  "lando",
			args: []string{"wp", "acorn", "key:generate"},
		},
	}

	if !reflect.DeepEqual(runner.commands, expected) {
		t.Fatalf("unexpected commands:\nexpected: %#v\ngot: %#v", expected, runner.commands)
	}
}

func TestInstallFailsWhenTargetExists(t *testing.T) {
	themesDir := t.TempDir()
	targetDir := filepath.Join(themesDir, "demo")
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		t.Fatalf("failed to create target dir: %v", err)
	}

	_, err := Install(&fakeRunner{}, themesDir, testStarterRepo, "demo")
	if err == nil {
		t.Fatal("expected install to fail when theme dir exists")
	}
}

func containsRecordedCommand(commands []recordedCommand, expected recordedCommand) bool {
	for _, command := range commands {
		if reflect.DeepEqual(command, expected) {
			return true
		}
	}

	return false
}
