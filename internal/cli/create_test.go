package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/gotcha190/ToBA/internal/create"
)

type recordedCommand struct {
	dir  string
	cmd  string
	args []string
}

type fakeRunner struct {
	commands []recordedCommand
	runErr   error
}

func (r *fakeRunner) Run(dir string, cmd string, args ...string) error {
	r.commands = append(r.commands, recordedCommand{
		dir:  dir,
		cmd:  cmd,
		args: append([]string(nil), args...),
	})
	return r.runErr
}

func (r *fakeRunner) CaptureOutput(dir string, cmd string, args ...string) (string, error) {
	if err := r.Run(dir, cmd, args...); err != nil {
		return "", err
	}
	return "", nil
}

func TestRunCreateCreatesProjectSkeleton(t *testing.T) {
	baseDir := t.TempDir()
	withWorkingDir(t, baseDir)
	runner := &fakeRunner{}

	err := runCreateWithRunner(CreateOptions{
		Name:       "demo",
		PHPVersion: "8.4",
		Domain:     "demo.lndo.site",
		Database:   "demo_db",
	}, runner)
	if err != nil {
		t.Fatalf("RunCreate returned error: %v", err)
	}

	paths := create.NewProjectPaths(baseDir, "demo")
	for _, dir := range []string{paths.Root, paths.AppDir, paths.ConfigDir} {
		info, statErr := os.Stat(dir)
		if statErr != nil {
			t.Fatalf("expected %s to exist: %v", dir, statErr)
		}
		if !info.IsDir() {
			t.Fatalf("expected %s to be a directory", dir)
		}
	}

	landoConfigPath := filepath.Join(paths.Root, ".lando.yml")
	landoConfig, readErr := os.ReadFile(landoConfigPath)
	if readErr != nil {
		t.Fatalf("expected %s to exist: %v", landoConfigPath, readErr)
	}

	output := string(landoConfig)
	if !strings.Contains(output, "name: demo") {
		t.Fatalf("expected rendered .lando.yml to include project name, got:\n%s", output)
	}
	if !strings.Contains(output, `php: "8.4"`) {
		t.Fatalf("expected rendered .lando.yml to include php version, got:\n%s", output)
	}
	if !strings.Contains(output, "type: node:18") {
		t.Fatalf("expected rendered .lando.yml to include theme service, got:\n%s", output)
	}
	if _, statErr := os.Stat(paths.WPContent); !os.IsNotExist(statErr) {
		t.Fatalf("expected %s not to exist before WordPress install, got err=%v", paths.WPContent, statErr)
	}

	phpINIPath := filepath.Join(paths.ConfigDir, "php.ini")
	phpINI, readPHPINIError := os.ReadFile(phpINIPath)
	if readPHPINIError != nil {
		t.Fatalf("expected %s to exist: %v", phpINIPath, readPHPINIError)
	}
	if !strings.Contains(string(phpINI), "xdebug.mode = profile,debug,develop") {
		t.Fatalf("expected generated php.ini to contain xdebug mode, got:\n%s", string(phpINI))
	}
	if !strings.Contains(string(phpINI), "xdebug.client_host = ${LANDO_HOST_IP}") {
		t.Fatalf("expected generated php.ini to contain default settings, got:\n%s", string(phpINI))
	}

	wpCLIPath := filepath.Join(paths.Root, "wp-cli.yml")
	wpCLIConfig, readWPCLIError := os.ReadFile(wpCLIPath)
	if readWPCLIError != nil {
		t.Fatalf("expected %s to exist: %v", wpCLIPath, readWPCLIError)
	}
	if string(wpCLIConfig) != "path: app\nserver:\n  docroot: app\n" {
		t.Fatalf("unexpected wp-cli.yml contents:\n%s", string(wpCLIConfig))
	}

	expectedCommands := []recordedCommand{
		{
			dir:  paths.Root,
			cmd:  "lando",
			args: []string{"start"},
		},
		{
			dir:  paths.Root,
			cmd:  "lando",
			args: []string{"wp", "core", "download", "--locale=pl_PL"},
		},
		{
			dir:  paths.Root,
			cmd:  "lando",
			args: []string{"wp", "config", "create", "--dbname=wordpress", "--dbuser=wordpress", "--dbpass=wordpress", "--dbhost=database"},
		},
		{
			dir:  paths.Root,
			cmd:  "lando",
			args: []string{"wp", "core", "install", "--url=demo.lndo.site", "--title=Demo", "--admin_user=tamago", "--admin_email=email@email.pl", "--admin_password=tamago"},
		},
	}

	if !reflect.DeepEqual(runner.commands, expectedCommands) {
		t.Fatalf("unexpected command sequence:\nexpected: %#v\ngot: %#v", expectedCommands, runner.commands)
	}
}

func TestRunCreateFailsWhenDirectoryExists(t *testing.T) {
	baseDir := t.TempDir()
	withWorkingDir(t, baseDir)

	existingRoot := filepath.Join(baseDir, "demo")
	if err := os.Mkdir(existingRoot, 0755); err != nil {
		t.Fatalf("failed to prepare existing directory: %v", err)
	}

	err := runCreateWithRunner(CreateOptions{Name: "demo"}, &fakeRunner{})
	if err == nil {
		t.Fatal("expected RunCreate to fail when project directory already exists")
	}
	if !strings.Contains(err.Error(), "directory already exists") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRunCreateDryRunDoesNotWriteFiles(t *testing.T) {
	baseDir := t.TempDir()
	withWorkingDir(t, baseDir)
	runner := &fakeRunner{}

	err := runCreateWithRunner(CreateOptions{Name: "demo", DryRun: true}, runner)
	if err != nil {
		t.Fatalf("RunCreate returned error in dry-run mode: %v", err)
	}

	projectRoot := filepath.Join(baseDir, "demo")
	if _, statErr := os.Stat(projectRoot); !os.IsNotExist(statErr) {
		t.Fatalf("expected %s not to exist after dry-run, got err=%v", projectRoot, statErr)
	}
	if len(runner.commands) != 0 {
		t.Fatalf("expected dry-run not to execute commands, got: %#v", runner.commands)
	}
}

func TestRunCreateReturnsRunnerError(t *testing.T) {
	baseDir := t.TempDir()
	withWorkingDir(t, baseDir)

	expectedErr := fmt.Errorf("lando failed")
	err := runCreateWithRunner(CreateOptions{Name: "demo"}, &fakeRunner{runErr: expectedErr})
	if err == nil {
		t.Fatal("expected runCreateWithRunner to return runner error")
	}
	if !strings.Contains(err.Error(), "lando start failed") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func withWorkingDir(t *testing.T, dir string) {
	t.Helper()

	original, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get working directory: %v", err)
	}

	if err := os.Chdir(dir); err != nil {
		t.Fatalf("failed to chdir to temp dir: %v", err)
	}

	t.Cleanup(func() {
		if err := os.Chdir(original); err != nil {
			t.Fatalf("failed to restore working directory: %v", err)
		}
	})
}
