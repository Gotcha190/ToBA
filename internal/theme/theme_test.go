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
	runErr   map[string]error
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
	if r.runErr != nil {
		key := strings.Join(append([]string{cmd}, args...), " ")
		if err, ok := r.runErr[key]; ok {
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

	expected := []recordedCommand{
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
		{
			dir:  themeDir,
			cmd:  "npm",
			args: []string{"run", "build"},
		},
	}

	if len(runner.commands) != len(expected) {
		t.Fatalf("unexpected commands count:\nexpected: %d\ngot: %d\ncommands: %#v", len(expected), len(runner.commands), runner.commands)
	}
	for _, command := range expected {
		if !containsRecordedCommand(runner.commands, command) {
			t.Fatalf("expected command %#v in %#v", command, runner.commands)
		}
	}
	if !reflect.DeepEqual(runner.commands[len(runner.commands)-1], expected[2]) {
		t.Fatalf("unexpected final command:\nexpected: %#v\ngot: %#v", expected[2], runner.commands[len(runner.commands)-1])
	}
}

func TestBuildFallsBackToNpmInstallWhenNpmCiFails(t *testing.T) {
	runner := &fakeRunner{
		runErr: map[string]error{
			"npm ci --no-audit --no-fund": errors.New("broken lockfile"),
		},
	}
	themeDir := "/tmp/demo-theme"

	if err := Build(runner, themeDir); err != nil {
		t.Fatalf("Build returned error: %v", err)
	}

	expectedFallback := recordedCommand{
		dir:  themeDir,
		cmd:  "npm",
		args: []string{"install", "--no-audit", "--no-fund"},
	}
	if !containsRecordedCommand(runner.commands, expectedFallback) {
		t.Fatalf("expected fallback command %#v in %#v", expectedFallback, runner.commands)
	}

	expectedBuild := recordedCommand{
		dir:  themeDir,
		cmd:  "npm",
		args: []string{"run", "build"},
	}
	if !reflect.DeepEqual(runner.commands[len(runner.commands)-1], expectedBuild) {
		t.Fatalf("unexpected final build command:\nexpected: %#v\ngot: %#v", expectedBuild, runner.commands[len(runner.commands)-1])
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

func TestBuildReturnsFallbackErrorWhenNpmCiAndInstallFail(t *testing.T) {
	ciErr := errors.New("ci failed")
	installErr := errors.New("install failed")
	runner := &fakeRunner{
		runErr: map[string]error{
			"npm ci --no-audit --no-fund":      ciErr,
			"npm install --no-audit --no-fund": installErr,
		},
	}

	err := Build(runner, "/tmp/demo-theme")
	if !errors.Is(err, installErr) {
		t.Fatalf("expected %v, got %v", installErr, err)
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
