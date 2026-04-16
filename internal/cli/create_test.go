package cli

import (
	"archive/zip"
	"fmt"
	"os"
	"path/filepath"
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
	homeURL  string
	failCmd  string
}

func (r *fakeRunner) Run(dir string, cmd string, args ...string) error {
	r.commands = append(r.commands, recordedCommand{
		dir:  dir,
		cmd:  cmd,
		args: append([]string(nil), args...),
	})
	if cmd == "git" && len(args) == 3 && args[0] == "clone" {
		if err := os.MkdirAll(filepath.Join(dir, args[2]), 0755); err != nil {
			return err
		}
	}
	if cmd == "scp" && len(args) >= 3 {
		if err := materializeDownloadedFixture(args[len(args)-2], args[len(args)-1]); err != nil {
			return err
		}
	}
	if r.runErr != nil && (r.failCmd == "" || r.failCmd == cmd) {
		return r.runErr
	}
	return nil
}

func (r *fakeRunner) CaptureOutput(dir string, cmd string, args ...string) (string, error) {
	r.commands = append(r.commands, recordedCommand{
		dir:  dir,
		cmd:  cmd,
		args: append([]string(nil), args...),
	})
	if r.runErr != nil && (r.failCmd == "" || r.failCmd == cmd) {
		return "", r.runErr
	}
	if cmd == "ssh" && len(args) > 0 && strings.Contains(args[len(args)-1], "wp84 option get home") {
		if r.homeURL == "" {
			return "https://starter.tamago-dev.pl\n", nil
		}
		return r.homeURL + "\n", nil
	}
	if cmd == "lando" && len(args) == 4 && args[0] == "wp" && args[1] == "option" && args[2] == "get" {
		switch args[3] {
		case "stylesheet", "template":
			return "toet\n", nil
		}
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
		SSHTarget:   "user@192.168.0.1 -p 22",
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
	if string(wpCLIConfig) != "path: app\nserver:\n  docroot: app\napache_modules:\n  - mod_rewrite\n" {
		t.Fatalf("unexpected wp-cli.yml contents:\n%s", string(wpCLIConfig))
	}
	for _, path := range []string{
		filepath.Join(paths.WPContent, "plugins", "webp-converter-for-media"),
		filepath.Join(paths.WPContent, "plugins", "redis-cache"),
		filepath.Join(paths.WPContent, "uploads", "2025", "07"),
		paths.DatabaseSQL,
	} {
		if _, statErr := os.Stat(path); statErr != nil {
			t.Fatalf("expected %s to exist: %v", path, statErr)
		}
	}
	if _, statErr := os.Stat(filepath.Join(paths.WPContent, "uploads", "2026")); os.IsNotExist(statErr) {
		t.Fatalf("expected regular uploads archive to be extracted")
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
			args: []string{"wp", "config", "create", "--dbname=wordpress", "--dbuser=wordpress", "--dbpass=wordpress", "--dbhost=database", "--dbcharset=utf8mb4"},
		},
		{
			dir:  paths.Root,
			cmd:  "lando",
			args: []string{"wp", "core", "install", "--url=demo.lndo.site", "--title=Demo", "--admin_user=tamago", "--admin_email=email@email.pl", "--admin_password=tamago"},
		},
		{
			dir:  paths.Root,
			cmd:  "lando",
			args: []string{"db-import", "app/database.sql"},
		},
		{
			dir: paths.Root,
			cmd: "lando",
			args: []string{
				"wp",
				"search-replace",
				"https://toet-szablon.lndo.site",
				"https://demo.lndo.site",
				"--all-tables-with-prefix",
				"--skip-columns=guid",
			},
		},
		{
			dir:  paths.Root,
			cmd:  "lando",
			args: []string{"wp", "cache", "flush"},
		},
		{
			dir:  paths.Root,
			cmd:  "lando",
			args: []string{"wp", "acorn", "optimize:clear"},
		},
		{
			dir:  paths.Root,
			cmd:  "lando",
			args: []string{"wp", "user", "update", "1", "--user_pass=tamago"},
		},
		{
			dir:  paths.Root,
			cmd:  "lando",
			args: []string{"wp", "option", "get", "stylesheet"},
		},
		{
			dir:  paths.Root,
			cmd:  "lando",
			args: []string{"wp", "theme", "activate", "toet"},
		},
		{
			dir:  paths.Root,
			cmd:  "lando",
			args: []string{"wp", "rewrite", "flush", "--hard"},
		},
		{
			dir:  paths.Root,
			cmd:  "lando",
			args: []string{"wp", "acorn", "optimize"},
		},
		{
			dir:  paths.Root,
			cmd:  "lando",
			args: []string{"wp", "acorn", "cache:clear"},
		},
		{
			dir:  paths.Root,
			cmd:  "lando",
			args: []string{"wp", "acorn", "acf:cache"},
		},
	}

	if !commandsMatch(runner.commands, expectedCommands) {
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

	err := runCreateWithRunner(CreateOptions{Name: "demo", StarterRepo: testStarterRepo, SSHTarget: "user@192.168.0.1 -p 22"}, &fakeRunner{})
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

	err := runCreateWithRunner(CreateOptions{Name: "demo", StarterRepo: testStarterRepo, SSHTarget: "user@192.168.0.1 -p 22", DryRun: true}, runner)
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
	err := runCreateWithRunner(CreateOptions{Name: "demo", StarterRepo: testStarterRepo, SSHTarget: "user@192.168.0.1 -p 22"}, &fakeRunner{runErr: expectedErr, failCmd: "lando"})
	if err == nil {
		t.Fatal("expected runCreateWithRunner to return runner error")
	}
	if !strings.Contains(err.Error(), "lando start failed") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRunCreateUsesEmbeddedStarterWithoutStarterRepo(t *testing.T) {
	baseDir := t.TempDir()
	withWorkingDir(t, baseDir)

	runner := &fakeRunner{}
	if err := runCreateWithRunner(CreateOptions{Name: "demo", SSHTarget: "user@192.168.0.1 -p 22"}, runner); err != nil {
		t.Fatalf("expected embedded starter override to work without starter repo, got: %v", err)
	}
	for _, command := range runner.commands {
		if command.cmd == "git" {
			t.Fatalf("expected no git clone when embedded theme backup exists, got %#v", runner.commands)
		}
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
		"TOBA_DOMAIN=stale-from-env.lndo.site\n" +
		"TOBA_STARTER_REPO=" + testStarterRepo + "\n" +
		"TOBA_SSH_TARGET=user@192.168.0.1 -p 22\n"
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
	var installURL string
	for _, command := range runner.commands {
		if command.cmd != "lando" || len(command.args) < 4 {
			continue
		}
		if command.args[0] == "wp" && command.args[1] == "core" && command.args[2] == "install" {
			installURL = command.args[3]
			break
		}
	}
	if installURL != "--url=demo.lndo.site" {
		t.Fatalf("expected derived domain --url=demo.lndo.site, got %q", installURL)
	}
}

func TestRunCreateCleansUpFailedInstallWhenConfirmed(t *testing.T) {
	baseDir := t.TempDir()
	withWorkingDir(t, baseDir)

	input := strings.NewReader("y\n")
	output := &strings.Builder{}
	err := runCreateWithIO(
		CreateOptions{Name: "demo", SSHTarget: "user@192.168.0.1 -p 22"},
		&fakeRunner{runErr: fmt.Errorf("lando failed"), failCmd: "lando"},
		input,
		output,
	)
	if err == nil {
		t.Fatal("expected create to fail")
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

func materializeDownloadedFixture(remoteSource string, localTarget string) error {
	switch {
	case strings.HasSuffix(remoteSource, ".sql"):
		return os.WriteFile(localTarget, []byte("# Home URL: https://starter.tamago-dev.pl\nSELECT 1;\n"), 0644)
	case strings.HasSuffix(remoteSource, "-plugins.zip"):
		return writeZipFixture(localTarget, map[string]string{
			"plugins/advanced-custom-fields-pro/acf.php": "<?php\n",
			"plugins/wp-optimize/wp-optimize.php":        "<?php\n",
		})
	case strings.HasSuffix(remoteSource, "-uploads.zip"):
		return writeZipFixture(localTarget, map[string]string{
			"uploads/2025/07/example.jpg": "image",
			"uploads/2026/readme.txt":     "uploaded",
		})
	default:
		return nil
	}
}

func writeZipFixture(path string, files map[string]string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}

	output, err := os.Create(path)
	if err != nil {
		return err
	}
	defer output.Close()

	writer := zip.NewWriter(output)
	for fileName, content := range files {
		entry, err := writer.Create(fileName)
		if err != nil {
			return err
		}
		if _, err := entry.Write([]byte(content)); err != nil {
			return err
		}
	}

	return writer.Close()
}

func commandsMatch(actual []recordedCommand, expected []recordedCommand) bool {
	if len(actual) != len(expected) {
		return false
	}

	for i := range actual {
		if actual[i].dir != expected[i].dir || actual[i].cmd != expected[i].cmd || len(actual[i].args) != len(expected[i].args) {
			return false
		}
		for j := range actual[i].args {
			expectedArg := expected[i].args[j]
			if strings.HasPrefix(expectedArg, "dynamic:") {
				if !matchesDynamicArg(actual[i].args[j], strings.TrimPrefix(expectedArg, "dynamic:")) {
					return false
				}
				continue
			}
			if actual[i].args[j] != expectedArg {
				return false
			}
		}
	}

	return true
}

func matchesDynamicArg(actual string, pattern string) bool {
	switch pattern {
	case "db-export":
		return strings.HasPrefix(actual, "cd 'www/toba.tamago-dev.pl' && wp84 db export ") && strings.HasSuffix(actual, ".sql'")
	case "zip-plugins":
		return strings.HasPrefix(actual, "cd 'www/toba.tamago-dev.pl/wp-content' && zip -rq ../") && strings.Contains(actual, "-plugins.zip") && strings.HasSuffix(actual, " plugins")
	case "zip-uploads":
		return strings.HasPrefix(actual, "cd 'www/toba.tamago-dev.pl/wp-content' && zip -rq ../") && strings.Contains(actual, "-uploads.zip") && strings.HasSuffix(actual, " uploads")
	case "remote-sql":
		return strings.HasPrefix(actual, "user@192.168.0.1:www/toba.tamago-dev.pl/") && strings.HasSuffix(actual, ".sql")
	case "remote-plugins":
		return strings.HasPrefix(actual, "user@192.168.0.1:www/toba.tamago-dev.pl/") && strings.HasSuffix(actual, "-plugins.zip")
	case "remote-uploads":
		return strings.HasPrefix(actual, "user@192.168.0.1:www/toba.tamago-dev.pl/") && strings.HasSuffix(actual, "-uploads.zip")
	case "local-sql":
		return strings.HasSuffix(actual, ".sql")
	case "local-plugins":
		return strings.HasSuffix(actual, "-plugins.zip")
	case "local-uploads":
		return strings.HasSuffix(actual, "-uploads.zip")
	case "cleanup":
		return strings.HasPrefix(actual, "cd 'www/toba.tamago-dev.pl' && rm -f 'toba-create-") && strings.Contains(actual, ".sql'") && strings.Contains(actual, "-plugins.zip'") && strings.Contains(actual, "-uploads.zip'")
	default:
		return false
	}
}

func matchDynamicRemote(suffix string) string {
	switch suffix {
	case ".sql":
		return "dynamic:remote-sql"
	case "-plugins.zip":
		return "dynamic:remote-plugins"
	case "-uploads.zip":
		return "dynamic:remote-uploads"
	default:
		return ""
	}
}

func matchDynamicLocal(suffix string) string {
	switch suffix {
	case ".sql":
		return "dynamic:local-sql"
	case "-plugins.zip":
		return "dynamic:local-plugins"
	case "-uploads.zip":
		return "dynamic:local-uploads"
	default:
		return ""
	}
}
