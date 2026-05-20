package git

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"testing"

	"github.com/gotcha190/toba/internal/create"
)

const testStarterRepo = "git@example.com:company/starter.git"
const testProjectRepo = "git@example.com:company/demo.git"

type recordedCommand struct {
	dir  string
	cmd  string
	args []string
}

type fakeRunner struct {
	mu                     sync.Mutex
	commands               []recordedCommand
	err                    error
	runErrByCommand        map[string]error
	captureOutputByCommand map[string]string
	captureErrByCommand    map[string]error
}

func (r *fakeRunner) Run(dir string, cmd string, args ...string) error {
	r.mu.Lock()
	r.commands = append(r.commands, recordedCommand{
		dir:  dir,
		cmd:  cmd,
		args: append([]string(nil), args...),
	})
	r.mu.Unlock()

	if r.runErrByCommand != nil {
		if err, ok := r.runErrByCommand[cmd+" "+strings.Join(args, " ")]; ok {
			return err
		}
	}
	if r.err != nil {
		return r.err
	}
	if cmd == "git" && len(args) == 3 && args[0] == "clone" {
		return os.MkdirAll(filepath.Join(dir, args[2]), 0755)
	}

	return nil
}

func (r *fakeRunner) CaptureOutput(dir string, cmd string, args ...string) (string, error) {
	r.mu.Lock()
	r.commands = append(r.commands, recordedCommand{
		dir:  dir,
		cmd:  cmd,
		args: append([]string(nil), args...),
	})
	r.mu.Unlock()

	key := cmd + " " + strings.Join(args, " ")
	if r.captureErrByCommand != nil {
		if err, ok := r.captureErrByCommand[key]; ok {
			return "", err
		}
	}
	if r.captureOutputByCommand != nil {
		if output, ok := r.captureOutputByCommand[key]; ok {
			return output, nil
		}
	}

	return "", r.err
}

func TestCloneReturnsMissingRepoError(t *testing.T) {
	_, err := Clone(&fakeRunner{}, t.TempDir(), "", "demo")
	if err == nil {
		t.Fatal("expected missing starter repo error")
	}
	if !strings.Contains(err.Error(), "TOBA_STARTER_REPO") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCloneRunsCloneCommand(t *testing.T) {
	themesDir := t.TempDir()
	runner := &fakeRunner{}

	targetDir, err := Clone(runner, themesDir, testStarterRepo, "demo")
	if err != nil {
		t.Fatalf("Clone returned error: %v", err)
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

func TestCloneFailsWhenTargetExists(t *testing.T) {
	themesDir := t.TempDir()
	targetDir := filepath.Join(themesDir, "demo")
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		t.Fatalf("failed to create target dir: %v", err)
	}

	_, err := Clone(&fakeRunner{}, themesDir, testStarterRepo, "demo")
	assertCodedError(t, err, "THEME_DIR_EXISTS")
}

func TestCloneReturnsCodedErrorWhenCloneFails(t *testing.T) {
	expectedErr := errors.New("clone failed")
	_, err := Clone(&fakeRunner{
		runErrByCommand: map[string]error{
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
	err := DetachRemoteAndCreateBranches(&fakeRunner{
		runErrByCommand: map[string]error{
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
	runner := &fakeRunner{}

	if err := DetachRemoteAndCreateBranches(runner, repoDir); err != nil {
		t.Fatalf("DetachRemoteAndCreateBranches returned error: %v", err)
	}

	expected := []recordedCommand{
		{
			dir:  repoDir,
			cmd:  "git",
			args: []string{"remote", "remove", "origin"},
		},
		{
			dir:  repoDir,
			cmd:  "git",
			args: []string{"branch", "-M", "develop"},
		},
		{
			dir:  repoDir,
			cmd:  "git",
			args: []string{"branch", "-f", "starter", "develop"},
		},
	}

	if !reflect.DeepEqual(runner.commands, expected) {
		t.Fatalf("unexpected commands:\nexpected: %#v\ngot: %#v", expected, runner.commands)
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
	runner := &fakeRunner{}

	result := TrySetupProjectBranches(runner, t.TempDir(), "not-a-repo", "demo")

	if len(result.Warnings) == 0 {
		t.Fatal("expected warning for invalid starter repo")
	}
	assertNoRecordedCommand(t, runner.commands, "git", []string{"ls-remote", testProjectRepo})
	assertNoRecordedGitSubcommand(t, runner.commands, "ls-remote")
	assertNoRecordedGitSubcommand(t, runner.commands, "remote add")
	assertNoRecordedGitSubcommand(t, runner.commands, "push")
}

func TestTrySetupProjectBranchesRemoteRemoveErrorContinuesToBranchDevelop(t *testing.T) {
	runner := &fakeRunner{
		runErrByCommand: map[string]error{
			"git remote remove origin": errors.New("remove failed"),
		},
	}

	result := TrySetupProjectBranches(runner, t.TempDir(), testStarterRepo, "demo")

	if len(result.Warnings) == 0 {
		t.Fatal("expected warning for remote remove error")
	}
	assertRecordedCommand(t, runner.commands, "git", []string{"branch", "-M", "develop"})
}

func TestTrySetupProjectBranchesBranchDevelopErrorContinuesToStarterBranch(t *testing.T) {
	runner := &fakeRunner{
		runErrByCommand: map[string]error{
			"git branch -M develop": errors.New("branch failed"),
		},
	}

	result := TrySetupProjectBranches(runner, t.TempDir(), testStarterRepo, "demo")

	if len(result.Warnings) == 0 {
		t.Fatal("expected warning for branch develop error")
	}
	assertRecordedCommand(t, runner.commands, "git", []string{"branch", "-f", "starter", "develop"})
}

func TestTrySetupProjectBranchesUnavailableProjectRepoSkipsRemoteAddAndPush(t *testing.T) {
	runner := &fakeRunner{
		runErrByCommand: map[string]error{
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
	assertRecordedCommand(t, runner.commands, "git", []string{"ls-remote", testProjectRepo})
	assertNoRecordedGitSubcommand(t, runner.commands, "remote add")
	assertNoRecordedGitSubcommand(t, runner.commands, "push")
}

func TestTrySetupProjectBranchesAvailableProjectRepoPushesBranches(t *testing.T) {
	runner := &fakeRunner{}

	result := TrySetupProjectBranches(runner, t.TempDir(), testStarterRepo, "demo")

	if !result.Pushed {
		t.Fatal("expected Pushed to be true")
	}
	if len(result.Warnings) != 0 {
		t.Fatalf("expected no warnings, got %#v", result.Warnings)
	}
	assertRecordedCommand(t, runner.commands, "git", []string{"remote", "add", "origin", testProjectRepo})
	assertRecordedCommand(t, runner.commands, "git", []string{"push", "-u", "origin", "develop", "starter"})
}

func TestTrySetupProjectBranchesExistingRemoteDevelopSkipsPush(t *testing.T) {
	runner := &fakeRunner{
		captureOutputByCommand: map[string]string{
			"git ls-remote --heads " + testProjectRepo + " develop starter": "abc123\trefs/heads/develop\n",
		},
	}

	result := TrySetupProjectBranches(runner, t.TempDir(), testStarterRepo, "demo")

	if result.Pushed {
		t.Fatal("expected Pushed to be false")
	}
	if !containsWarning(result.Warnings, "already has develop or starter branch") {
		t.Fatalf("expected existing branch warning, got %#v", result.Warnings)
	}
	assertRecordedCommand(t, runner.commands, "git", []string{"ls-remote", "--heads", testProjectRepo, "develop", "starter"})
	assertNoRecordedGitSubcommand(t, runner.commands, "remote add")
	assertNoRecordedGitSubcommand(t, runner.commands, "push")
}

func TestTrySetupProjectBranchesExistingRemoteStarterSkipsPush(t *testing.T) {
	runner := &fakeRunner{
		captureOutputByCommand: map[string]string{
			"git ls-remote --heads " + testProjectRepo + " develop starter": "abc123\trefs/heads/starter\n",
		},
	}

	result := TrySetupProjectBranches(runner, t.TempDir(), testStarterRepo, "demo")

	if result.Pushed {
		t.Fatal("expected Pushed to be false")
	}
	if !containsWarning(result.Warnings, "already has develop or starter branch") {
		t.Fatalf("expected existing branch warning, got %#v", result.Warnings)
	}
	assertNoRecordedGitSubcommand(t, runner.commands, "remote add")
	assertNoRecordedGitSubcommand(t, runner.commands, "push")
}

func TestTrySetupProjectBranchesBranchInspectionErrorSkipsPush(t *testing.T) {
	runner := &fakeRunner{
		captureErrByCommand: map[string]error{
			"git ls-remote --heads " + testProjectRepo + " develop starter": errors.New("inspect failed"),
		},
	}

	result := TrySetupProjectBranches(runner, t.TempDir(), testStarterRepo, "demo")

	if result.Pushed {
		t.Fatal("expected Pushed to be false")
	}
	if !containsWarning(result.Warnings, "Could not inspect project git branches") {
		t.Fatalf("expected inspection warning, got %#v", result.Warnings)
	}
	assertNoRecordedGitSubcommand(t, runner.commands, "remote add")
	assertNoRecordedGitSubcommand(t, runner.commands, "push")
}

func TestTrySetupProjectBranchesRemoteAddErrorSkipsPush(t *testing.T) {
	runner := &fakeRunner{
		runErrByCommand: map[string]error{
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
	assertRecordedCommand(t, runner.commands, "git", []string{"remote", "add", "origin", testProjectRepo})
	assertNoRecordedGitSubcommand(t, runner.commands, "push")
}

func TestTrySetupProjectBranchesPushErrorReportsNotPushed(t *testing.T) {
	runner := &fakeRunner{
		runErrByCommand: map[string]error{
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
	assertRecordedCommand(t, runner.commands, "git", []string{"push", "-u", "origin", "develop", "starter"})
}

func assertRecordedCommand(t *testing.T, commands []recordedCommand, cmd string, args []string) {
	t.Helper()

	for _, command := range commands {
		if command.cmd == cmd && reflect.DeepEqual(command.args, args) {
			return
		}
	}

	t.Fatalf("expected command %s %v, got %#v", cmd, args, commands)
}

func assertNoRecordedCommand(t *testing.T, commands []recordedCommand, cmd string, args []string) {
	t.Helper()

	for _, command := range commands {
		if command.cmd == cmd && reflect.DeepEqual(command.args, args) {
			t.Fatalf("did not expect command %s %v, got %#v", cmd, args, commands)
		}
	}
}

func assertNoRecordedGitSubcommand(t *testing.T, commands []recordedCommand, subcommand string) {
	t.Helper()

	for _, command := range commands {
		if command.cmd != "git" {
			continue
		}
		got := strings.Join(command.args, " ")
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
