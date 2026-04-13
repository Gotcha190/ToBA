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

const testStarterRepo = "git@example.com:company/starter.git"

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
		Name:        "demo",
		PHPVersion:  "8.4",
		Domain:      "demo.lndo.site",
		StarterRepo: testStarterRepo,
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
	if info, statErr := os.Stat(paths.Themes); statErr != nil {
		t.Fatalf("expected %s to exist after theme install, got err=%v", paths.Themes, statErr)
	} else if !info.IsDir() {
		t.Fatalf("expected %s to be a directory", paths.Themes)
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
		{
			dir:  paths.Themes,
			cmd:  "git",
			args: []string{"clone", testStarterRepo, "demo"},
		},
		{
			dir:  filepath.Join(paths.Themes, "demo"),
			cmd:  "lando",
			args: []string{"composer", "install"},
		},
		{
			dir:  filepath.Join(paths.Themes, "demo"),
			cmd:  "npm",
			args: []string{"i"},
		},
		{
			dir:  filepath.Join(paths.Themes, "demo"),
			cmd:  "npm",
			args: []string{"run", "build"},
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

	err := runCreateWithRunner(CreateOptions{Name: "demo", StarterRepo: testStarterRepo}, &fakeRunner{})
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

	err := runCreateWithRunner(CreateOptions{Name: "demo", StarterRepo: testStarterRepo, DryRun: true}, runner)
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
	err := runCreateWithRunner(CreateOptions{Name: "demo", StarterRepo: testStarterRepo}, &fakeRunner{runErr: expectedErr})
	if err == nil {
		t.Fatal("expected runCreateWithRunner to return runner error")
	}
	if !strings.Contains(err.Error(), "lando start failed") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRunCreateFailsWhenStarterRepoMissing(t *testing.T) {
	baseDir := t.TempDir()
	withWorkingDir(t, baseDir)

	err := runCreateWithRunner(CreateOptions{Name: "demo"}, &fakeRunner{})
	if err == nil {
		t.Fatal("expected missing starter repo error")
	}
	if !strings.Contains(err.Error(), "TOBA_STARTER_REPO") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRunCreateUsesEnvConfig(t *testing.T) {
	baseDir := t.TempDir()
	withWorkingDir(t, baseDir)
	runner := &fakeRunner{}

	globalEnvPath, err := create.GlobalEnvPath()
	if err != nil {
		t.Fatalf("GlobalEnvPath returned error: %v", err)
	}

	env := "" +
		"TOBA_PROJECT_NAME=demo\n" +
		"TOBA_PHP_VERSION=8.4\n" +
		"TOBA_DOMAIN=demo.lndo.site\n" +
		"TOBA_STARTER_REPO=" + testStarterRepo + "\n"
	if err := os.MkdirAll(filepath.Dir(globalEnvPath), 0755); err != nil {
		t.Fatalf("failed to create config dir: %v", err)
	}
	if err := os.WriteFile(globalEnvPath, []byte(env), 0644); err != nil {
		t.Fatalf("failed to write global env file: %v", err)
	}

	if err := runCreateWithRunner(CreateOptions{}, runner); err != nil {
		t.Fatalf("runCreateWithRunner returned error: %v", err)
	}

	if len(runner.commands) == 0 {
		t.Fatal("expected commands to be executed from env config")
	}
}

func TestRunCreateCleansUpFailedInstallWhenConfirmed(t *testing.T) {
	baseDir := t.TempDir()
	withWorkingDir(t, baseDir)

	input := strings.NewReader("y\n")
	output := &strings.Builder{}
	err := runCreateWithIO(CreateOptions{Name: "demo"}, &fakeRunner{}, input, output)
	if err == nil {
		t.Fatal("expected missing starter repo error")
	}

	projectRoot := filepath.Join(baseDir, "demo")
	if _, statErr := os.Stat(projectRoot); !os.IsNotExist(statErr) {
		t.Fatalf("expected project root to be removed, got err=%v", statErr)
	}
	if !strings.Contains(output.String(), "Delete failed installation") {
		t.Fatalf("expected cleanup prompt, got: %s", output.String())
	}
}

func TestRunCreateFailsWithoutGlobalConfig(t *testing.T) {
	baseDir := t.TempDir()
	withWorkingDir(t, baseDir)

	err := runCreateWithRunner(CreateOptions{}, &fakeRunner{})
	if err == nil {
		t.Fatal("expected missing global config error")
	}
	if !strings.Contains(err.Error(), "ToBA config init") {
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

	originalConfigHome, hadConfigHome := os.LookupEnv("XDG_CONFIG_HOME")
	configHome := filepath.Join(dir, ".config-home")
	if err := os.Setenv("XDG_CONFIG_HOME", configHome); err != nil {
		t.Fatalf("failed to set XDG_CONFIG_HOME: %v", err)
	}

	t.Cleanup(func() {
		if err := os.Chdir(original); err != nil {
			t.Fatalf("failed to restore working directory: %v", err)
		}
		if hadConfigHome {
			if err := os.Setenv("XDG_CONFIG_HOME", originalConfigHome); err != nil {
				t.Fatalf("failed to restore XDG_CONFIG_HOME: %v", err)
			}
			return
		}
		if err := os.Unsetenv("XDG_CONFIG_HOME"); err != nil {
			t.Fatalf("failed to unset XDG_CONFIG_HOME: %v", err)
		}
	})
}
