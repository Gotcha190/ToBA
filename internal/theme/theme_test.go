package theme

import (
	"errors"
	"reflect"
	"testing"

	"github.com/gotcha190/toba/internal/testutil"
)

func TestBuildRunsExpectedCommands(t *testing.T) {
	runner := &testutil.RecordingRunner{}
	themeDir := "/tmp/demo-theme"

	err := Build(runner, themeDir)
	if err != nil {
		t.Fatalf("Build returned error: %v", err)
	}

	expected := []testutil.RecordedCommand{
		{
			Dir:  themeDir,
			Cmd:  "lando",
			Args: []string{"composer", "install", "--no-interaction", "--prefer-dist", "--optimize-autoloader", "--no-progress"},
		},
		{
			Dir:  themeDir,
			Cmd:  "npm",
			Args: []string{"ci", "--no-audit", "--no-fund"},
		},
		{
			Dir:  themeDir,
			Cmd:  "npm",
			Args: []string{"run", "build"},
		},
	}

	if len(runner.Commands) != len(expected) {
		t.Fatalf("unexpected commands count:\nexpected: %d\ngot: %d\ncommands: %#v", len(expected), len(runner.Commands), runner.Commands)
	}
	for _, command := range expected {
		if !containsRecordedCommand(runner.Commands, command) {
			t.Fatalf("expected command %#v in %#v", command, runner.Commands)
		}
	}
	if !reflect.DeepEqual(runner.Commands[len(runner.Commands)-1], expected[2]) {
		t.Fatalf("unexpected final command:\nexpected: %#v\ngot: %#v", expected[2], runner.Commands[len(runner.Commands)-1])
	}
}

func TestBuildFallsBackToNpmInstallWhenNpmCiFails(t *testing.T) {
	runner := &testutil.RecordingRunner{
		RunErrByCommand: map[string]error{
			"npm ci --no-audit --no-fund": errors.New("broken lockfile"),
		},
	}
	themeDir := "/tmp/demo-theme"

	if err := Build(runner, themeDir); err != nil {
		t.Fatalf("Build returned error: %v", err)
	}

	expectedFallback := testutil.RecordedCommand{
		Dir:  themeDir,
		Cmd:  "npm",
		Args: []string{"install", "--no-audit", "--no-fund"},
	}
	if !containsRecordedCommand(runner.Commands, expectedFallback) {
		t.Fatalf("expected fallback command %#v in %#v", expectedFallback, runner.Commands)
	}

	expectedBuild := testutil.RecordedCommand{
		Dir:  themeDir,
		Cmd:  "npm",
		Args: []string{"run", "build"},
	}
	if !reflect.DeepEqual(runner.Commands[len(runner.Commands)-1], expectedBuild) {
		t.Fatalf("unexpected final build command:\nexpected: %#v\ngot: %#v", expectedBuild, runner.Commands[len(runner.Commands)-1])
	}
}

func TestBuildReturnsCommandError(t *testing.T) {
	expectedErr := errors.New("boom")
	runner := &testutil.RecordingRunner{Err: expectedErr}

	err := Build(runner, "/tmp/demo-theme")
	if !errors.Is(err, expectedErr) {
		t.Fatalf("expected %v, got %v", expectedErr, err)
	}
}

func TestBuildReturnsFallbackErrorWhenNpmCiAndInstallFail(t *testing.T) {
	ciErr := errors.New("ci failed")
	installErr := errors.New("install failed")
	runner := &testutil.RecordingRunner{
		RunErrByCommand: map[string]error{
			"npm ci --no-audit --no-fund":      ciErr,
			"npm install --no-audit --no-fund": installErr,
		},
	}

	err := Build(runner, "/tmp/demo-theme")
	if !errors.Is(err, installErr) {
		t.Fatalf("expected %v, got %v", installErr, err)
	}
}

func containsRecordedCommand(commands []testutil.RecordedCommand, expected testutil.RecordedCommand) bool {
	for _, command := range commands {
		if reflect.DeepEqual(command, expected) {
			return true
		}
	}

	return false
}

func TestGenerateAcornKeyRunsTwice(t *testing.T) {
	runner := &testutil.RecordingRunner{}

	err := GenerateAcornKey(runner, "/tmp/demo")
	if err != nil {
		t.Fatalf("GenerateAcornKey returned error: %v", err)
	}

	expected := []testutil.RecordedCommand{
		{
			Dir:  "/tmp/demo",
			Cmd:  "lando",
			Args: []string{"wp", "acorn", "key:generate"},
		},
		{
			Dir:  "/tmp/demo",
			Cmd:  "lando",
			Args: []string{"wp", "acorn", "key:generate"},
		},
	}

	if !reflect.DeepEqual(runner.Commands, expected) {
		t.Fatalf("unexpected commands:\nexpected: %#v\ngot: %#v", expected, runner.Commands)
	}
}
