package theme

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

const testStarterRepo = "git@example.com:company/starter.git"

type recordedCommand struct {
	dir  string
	cmd  string
	args []string
}

type fakeRunner struct {
	commands []recordedCommand
	err      error
}

func (r *fakeRunner) Run(dir string, cmd string, args ...string) error {
	r.commands = append(r.commands, recordedCommand{
		dir:  dir,
		cmd:  cmd,
		args: append([]string(nil), args...),
	})
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

	err := Build(runner, "/tmp/demo-theme")
	if err != nil {
		t.Fatalf("Build returned error: %v", err)
	}

	expected := []recordedCommand{
		{
			dir:  "/tmp/demo-theme",
			cmd:  "lando",
			args: []string{"composer", "install"},
		},
		{
			dir:  "/tmp/demo-theme",
			cmd:  "npm",
			args: []string{"i"},
		},
		{
			dir:  "/tmp/demo-theme",
			cmd:  "npm",
			args: []string{"run", "build"},
		},
	}

	if !reflect.DeepEqual(runner.commands, expected) {
		t.Fatalf("unexpected commands:\nexpected: %#v\ngot: %#v", expected, runner.commands)
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
