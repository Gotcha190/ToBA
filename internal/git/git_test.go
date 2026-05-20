package git

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/gotcha190/toba/internal/create"
	"github.com/gotcha190/toba/internal/testutil"
)

const testStarterRepo = "git@example.com:company/starter.git"
const testProjectRepo = "git@example.com:company/demo.git"

func TestCloneReturnsMissingRepoError(t *testing.T) {
	_, err := Clone(&testutil.RecordingRunner{}, t.TempDir(), "", "demo")
	if err == nil {
		t.Fatal("expected missing starter repo error")
	}
	if !strings.Contains(err.Error(), "TOBA_STARTER_REPO") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCloneRunsCloneCommand(t *testing.T) {
	themesDir := t.TempDir()
	runner := &testutil.RecordingRunner{}

	targetDir, err := Clone(runner, themesDir, testStarterRepo, "demo")
	if err != nil {
		t.Fatalf("Clone returned error: %v", err)
	}

	expectedTarget := filepath.Join(themesDir, "demo")
	if targetDir != expectedTarget {
		t.Fatalf("expected target dir %q, got %q", expectedTarget, targetDir)
	}

	expected := []testutil.RecordedCommand{
		{
			Dir:  themesDir,
			Cmd:  "git",
			Args: []string{"clone", testStarterRepo, "demo"},
		},
	}

	if !reflect.DeepEqual(runner.Commands, expected) {
		t.Fatalf("unexpected commands:\nexpected: %#v\ngot: %#v", expected, runner.Commands)
	}
}

func TestCloneFailsWhenTargetExists(t *testing.T) {
	themesDir := t.TempDir()
	targetDir := filepath.Join(themesDir, "demo")
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		t.Fatalf("failed to create target dir: %v", err)
	}

	_, err := Clone(&testutil.RecordingRunner{}, themesDir, testStarterRepo, "demo")
	assertCodedError(t, err, "THEME_DIR_EXISTS")
}

func TestCloneReturnsCodedErrorWhenCloneFails(t *testing.T) {
	expectedErr := errors.New("clone failed")
	_, err := Clone(&testutil.RecordingRunner{
		RunErrByCommand: map[string]error{
			"git clone " + testStarterRepo + " demo": expectedErr,
		},
	}, t.TempDir(), testStarterRepo, "demo")

	assertCodedError(t, err, "THEME_CLONE_FAILED")
	if !errors.Is(err, expectedErr) {
		t.Fatalf("expected wrapped error %v, got %v", expectedErr, err)
	}
}

func TestDetachRemoteAndCreateBranchesReturnsCodedErrorWhenSetupFails(t *testing.T) {
	expectedErr := errors.New("remote remove failed")
	err := DetachRemoteAndCreateBranches(&testutil.RecordingRunner{
		RunErrByCommand: map[string]error{
			"git remote remove origin": expectedErr,
		},
	}, t.TempDir())

	assertCodedError(t, err, "THEME_GIT_SETUP_FAILED")
	if !errors.Is(err, expectedErr) {
		t.Fatalf("expected wrapped error %v, got %v", expectedErr, err)
	}
}

func TestDetachRemoteAndCreateBranchesRunsGitSetupCommands(t *testing.T) {
	repoDir := t.TempDir()
	runner := &testutil.RecordingRunner{}

	if err := DetachRemoteAndCreateBranches(runner, repoDir); err != nil {
		t.Fatalf("DetachRemoteAndCreateBranches returned error: %v", err)
	}

	expected := []testutil.RecordedCommand{
		{
			Dir:  repoDir,
			Cmd:  "git",
			Args: []string{"remote", "remove", "origin"},
		},
		{
			Dir:  repoDir,
			Cmd:  "git",
			Args: []string{"branch", "-M", "develop"},
		},
		{
			Dir:  repoDir,
			Cmd:  "git",
			Args: []string{"branch", "-f", "starter", "develop"},
		},
	}

	if !reflect.DeepEqual(runner.Commands, expected) {
		t.Fatalf("unexpected commands:\nexpected: %#v\ngot: %#v", expected, runner.Commands)
	}
}

func TestProjectRepoURLFromStarterSSH(t *testing.T) {
	got, err := ProjectRepoURLFromStarter(testStarterRepo, "demo")
	if err != nil {
		t.Fatalf("ProjectRepoURLFromStarter returned error: %v", err)
	}
	if got != testProjectRepo {
		t.Fatalf("expected SSH project repo URL, got %q", got)
	}
}

func TestProjectRepoURLFromStarterHTTPS(t *testing.T) {
	got, err := ProjectRepoURLFromStarter("https://github.com/company/starter.git", "demo")
	if err != nil {
		t.Fatalf("ProjectRepoURLFromStarter returned error: %v", err)
	}
	if got != "https://github.com/company/demo.git" {
		t.Fatalf("expected HTTPS project repo URL, got %q", got)
	}
}

func TestTrySetupProjectBranchesInvalidStarterRepoSkipsPush(t *testing.T) {
	runner := &testutil.RecordingRunner{}

	result := TrySetupProjectBranches(runner, t.TempDir(), "not-a-repo", "demo")

	if len(result.Warnings) == 0 {
		t.Fatal("expected warning for invalid starter repo")
	}
	assertNoRecordedCommand(t, runner.Commands, "git", []string{"ls-remote", testProjectRepo})
	assertNoRecordedGitSubcommand(t, runner.Commands, "ls-remote")
	assertNoRecordedGitSubcommand(t, runner.Commands, "remote add")
	assertNoRecordedGitSubcommand(t, runner.Commands, "push")
}

func TestTrySetupProjectBranchesRemoteRemoveErrorContinuesToBranchDevelop(t *testing.T) {
	runner := &testutil.RecordingRunner{
		RunErrByCommand: map[string]error{
			"git remote remove origin": errors.New("remove failed"),
		},
	}

	result := TrySetupProjectBranches(runner, t.TempDir(), testStarterRepo, "demo")

	if len(result.Warnings) == 0 {
		t.Fatal("expected warning for remote remove error")
	}
	assertRecordedCommand(t, runner.Commands, "git", []string{"branch", "-M", "develop"})
}

func TestTrySetupProjectBranchesBranchDevelopErrorContinuesToStarterBranch(t *testing.T) {
	runner := &testutil.RecordingRunner{
		RunErrByCommand: map[string]error{
			"git branch -M develop": errors.New("branch failed"),
		},
	}

	result := TrySetupProjectBranches(runner, t.TempDir(), testStarterRepo, "demo")

	if len(result.Warnings) == 0 {
		t.Fatal("expected warning for branch develop error")
	}
	assertRecordedCommand(t, runner.Commands, "git", []string{"branch", "-f", "starter", "develop"})
}

func TestTrySetupProjectBranchesUnavailableProjectRepoSkipsRemoteAddAndPush(t *testing.T) {
	runner := &testutil.RecordingRunner{
		RunErrByCommand: map[string]error{
			"git ls-remote " + testProjectRepo: errors.New("repo missing"),
		},
	}

	result := TrySetupProjectBranches(runner, t.TempDir(), testStarterRepo, "demo")

	if result.Pushed {
		t.Fatal("expected Pushed to be false")
	}
	if len(result.Warnings) == 0 {
		t.Fatal("expected warning for unavailable project repo")
	}
	assertRecordedCommand(t, runner.Commands, "git", []string{"ls-remote", testProjectRepo})
	assertNoRecordedGitSubcommand(t, runner.Commands, "remote add")
	assertNoRecordedGitSubcommand(t, runner.Commands, "push")
}

func TestTrySetupProjectBranchesAvailableProjectRepoPushesBranches(t *testing.T) {
	runner := &testutil.RecordingRunner{}

	result := TrySetupProjectBranches(runner, t.TempDir(), testStarterRepo, "demo")

	if !result.Pushed {
		t.Fatal("expected Pushed to be true")
	}
	if len(result.Warnings) != 0 {
		t.Fatalf("expected no warnings, got %#v", result.Warnings)
	}
	assertRecordedCommand(t, runner.Commands, "git", []string{"remote", "add", "origin", testProjectRepo})
	assertRecordedCommand(t, runner.Commands, "git", []string{"push", "-u", "origin", "develop", "starter"})
}

func TestTrySetupProjectBranchesSkipsPushAfterRemoteBranchGuard(t *testing.T) {
	branchCheckCommand := "git ls-remote --heads " + testProjectRepo + " develop starter"
	tests := []struct {
		name            string
		output          string
		err             error
		expectedWarning string
	}{
		{
			name:            "existing develop",
			output:          "abc123\trefs/heads/develop\n",
			expectedWarning: "already has develop or starter branch",
		},
		{
			name:            "existing starter",
			output:          "abc123\trefs/heads/starter\n",
			expectedWarning: "already has develop or starter branch",
		},
		{
			name:            "inspection error",
			err:             errors.New("inspect failed"),
			expectedWarning: "Could not inspect project git branches",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			runner := &testutil.RecordingRunner{
				Outputs: map[string]string{
					branchCheckCommand: tt.output,
				},
				CaptureErrByCommand: map[string]error{},
			}
			if tt.err != nil {
				runner.CaptureErrByCommand[branchCheckCommand] = tt.err
			}

			result := TrySetupProjectBranches(runner, t.TempDir(), testStarterRepo, "demo")

			if result.Pushed {
				t.Fatal("expected Pushed to be false")
			}
			if !containsWarning(result.Warnings, tt.expectedWarning) {
				t.Fatalf("expected warning %q, got %#v", tt.expectedWarning, result.Warnings)
			}
			assertRecordedCommand(t, runner.Commands, "git", []string{"ls-remote", "--heads", testProjectRepo, "develop", "starter"})
			assertNoRecordedGitSubcommand(t, runner.Commands, "remote add")
			assertNoRecordedGitSubcommand(t, runner.Commands, "push")
		})
	}
}

func TestTrySetupProjectBranchesRemoteAddErrorSkipsPush(t *testing.T) {
	runner := &testutil.RecordingRunner{
		RunErrByCommand: map[string]error{
			"git remote add origin " + testProjectRepo: errors.New("add failed"),
		},
	}

	result := TrySetupProjectBranches(runner, t.TempDir(), testStarterRepo, "demo")

	if result.Pushed {
		t.Fatal("expected Pushed to be false")
	}
	if len(result.Warnings) == 0 {
		t.Fatal("expected warning for remote add error")
	}
	assertRecordedCommand(t, runner.Commands, "git", []string{"remote", "add", "origin", testProjectRepo})
	assertNoRecordedGitSubcommand(t, runner.Commands, "push")
}

func TestTrySetupProjectBranchesPushErrorReportsNotPushed(t *testing.T) {
	runner := &testutil.RecordingRunner{
		RunErrByCommand: map[string]error{
			"git push -u origin develop starter": errors.New("push failed"),
		},
	}

	result := TrySetupProjectBranches(runner, t.TempDir(), testStarterRepo, "demo")

	if result.Pushed {
		t.Fatal("expected Pushed to be false")
	}
	if len(result.Warnings) == 0 {
		t.Fatal("expected warning for push error")
	}
	assertRecordedCommand(t, runner.Commands, "git", []string{"push", "-u", "origin", "develop", "starter"})
}

func assertRecordedCommand(t *testing.T, commands []testutil.RecordedCommand, cmd string, args []string) {
	t.Helper()

	for _, command := range commands {
		if command.Cmd == cmd && reflect.DeepEqual(command.Args, args) {
			return
		}
	}

	t.Fatalf("expected command %s %v, got %#v", cmd, args, commands)
}

func assertNoRecordedCommand(t *testing.T, commands []testutil.RecordedCommand, cmd string, args []string) {
	t.Helper()

	for _, command := range commands {
		if command.Cmd == cmd && reflect.DeepEqual(command.Args, args) {
			t.Fatalf("did not expect command %s %v, got %#v", cmd, args, commands)
		}
	}
}

func assertNoRecordedGitSubcommand(t *testing.T, commands []testutil.RecordedCommand, subcommand string) {
	t.Helper()

	for _, command := range commands {
		if command.Cmd != "git" {
			continue
		}
		got := strings.Join(command.Args, " ")
		if got == subcommand || strings.HasPrefix(got, subcommand+" ") {
			t.Fatalf("did not expect git %s, got %#v", subcommand, commands)
		}
	}
}

func containsWarning(warnings []string, fragment string) bool {
	for _, warning := range warnings {
		if strings.Contains(warning, fragment) {
			return true
		}
	}

	return false
}

func assertCodedError(t *testing.T, err error, expectedCode string) {
	t.Helper()

	if err == nil {
		t.Fatalf("expected coded error %s", expectedCode)
	}

	var coded interface {
		Code() string
	}
	if !errors.As(err, &coded) {
		t.Fatalf("expected coded error %s, got %T: %v", expectedCode, err, err)
	}
	if coded.Code() != expectedCode {
		t.Fatalf("expected code %s, got %s", expectedCode, coded.Code())
	}

	var createErr create.CodedError
	if !errors.As(err, &createErr) {
		t.Fatalf("expected create.CodedError, got %T: %v", err, err)
	}
}
