package cli

import (
	"archive/zip"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/gotcha190/toba/internal/create"
)

const testStarterRepo = "git@example.com:company/starter.git"
const testRemoteWordPressRoot = "www/example.com"

type recordedCommand struct {
	dir  string
	cmd  string
	args []string
}

type fakeRunner struct {
	mu                  sync.Mutex
	commands            []recordedCommand
	runErr              error
	runErrByCommand     map[string]error
	homeURL             string
	failCmd             string
	captureErr          error
	captureErrByCommand map[string]error
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
	if cmd == "scp" && len(args) >= 3 {
		if err := materializeDownloadedFixture(args[len(args)-2], args[len(args)-1]); err != nil {
			return err
		}
	}
	if r.runErrByCommand != nil {
		if err, ok := r.runErrByCommand[cmd+" "+strings.Join(args, " ")]; ok {
			return err
		}
	}
	if r.runErr != nil && (r.failCmd == "" || r.failCmd == cmd) {
		return r.runErr
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

	if r.captureErrByCommand != nil {
		if err, ok := r.captureErrByCommand[cmd+" "+strings.Join(args, " ")]; ok {
			return "", err
		}
	}
	if r.captureErr != nil {
		return "", r.captureErr
	}
	if r.runErr != nil && (r.failCmd == "" || r.failCmd == cmd) {
		return "", r.runErr
	}
	if cmd == "ssh" && len(args) > 0 && strings.Contains(args[len(args)-1], "wp84 option get home") {
		if r.homeURL == "" {
			return "https://starter.example.test\n", nil
		}
		return r.homeURL + "\n", nil
	}
	if cmd == "lando" && len(args) == 5 && args[0] == "wp" && args[1] == "user" && args[2] == "get" && args[3] == "tamago" && args[4] == "--field=ID" {
		return "1\n", nil
	}
	if cmd == "lando" && len(args) == 3 && args[0] == "wp" && args[1] == "eval" {
		if strings.Contains(args[2], "get_option('stylesheet') ?: get_option('template')") {
			return "toet\n", nil
		}
	}

	return "", nil
}

func TestRunCreateCreatesProjectSkeletonFromSSHStarter(t *testing.T) {
	baseDir := t.TempDir()
	withWorkingDir(t, baseDir)
	runner := &fakeRunner{}

	err := runCreateWithRunner(CreateOptions{
		Name:                "demo",
		PHPVersion:          "8.4",
		StarterRepo:         testStarterRepo,
		SSHTarget:           "user@192.168.0.1 -p 22",
		RemoteWordPressRoot: testRemoteWordPressRoot,
	}, runner)
	if err != nil {
		t.Fatalf("RunCreate returned error: %v", err)
	}

	paths := create.NewProjectPaths(baseDir, "demo")
	assertProjectSkeleton(t, paths)
	assertStaticConfigs(t, paths)

	for _, path := range []string{
		filepath.Join(paths.WPContent, "plugins", "advanced-custom-fields-pro", "acf.php"),
		filepath.Join(paths.WPContent, "plugins", "wp-optimize", "wp-optimize.php"),
		filepath.Join(paths.WPContent, "uploads", "2025", "07", "example.jpg"),
		paths.DatabaseSQL,
		filepath.Join(paths.Themes, "demo"),
	} {
		if _, statErr := os.Stat(path); statErr != nil {
			t.Fatalf("expected %s to exist: %v", path, statErr)
		}
	}

	assertHasCommand(t, runner.commands, "ssh", []string{"-p", "22", "user@192.168.0.1", "dynamic:home"})
	assertHasCommand(t, runner.commands, "scp", []string{"-P", "22", "dynamic:remote-sql", "dynamic:local-sql"})
	assertHasCommand(t, runner.commands, "git", []string{"clone", testStarterRepo, "demo"})
	assertHasCommand(t, runner.commands, "lando", []string{"composer", "install", "--no-interaction", "--prefer-dist", "--optimize-autoloader", "--no-progress"})
	assertHasCommand(t, runner.commands, "npm", []string{"ci", "--no-audit", "--no-fund"})
	assertHasCommand(t, runner.commands, "npm", []string{"run", "build"})
	assertHasCommand(t, runner.commands, "lando", []string{"wp", "theme", "activate", "demo"})
	assertCommandCount(t, runner.commands, "lando", []string{"wp", "acorn", "key:generate"}, 2)
	assertNoCommand(t, runner.commands, "lando", []string{"wp", "eval", "echo get_option('stylesheet') ?: get_option('template');"})
}

func TestRunCreateUsesExistingProjectFolderForLocalBackups(t *testing.T) {
	baseDir := t.TempDir()
	withWorkingDir(t, baseDir)

	projectRoot := filepath.Join(baseDir, "demo")
	writeCreateTestFile(t, filepath.Join(projectRoot, "backup-demo-db.sql"), "# Home URL: https://local-starter.test\nSELECT 1;\n")
	if err := writeZipFixture(filepath.Join(projectRoot, "backup-demo-plugins.zip"), map[string]string{
		"plugins/acf/acf.php": "<?php\n",
	}); err != nil {
		t.Fatalf("failed to write plugins zip: %v", err)
	}
	if err := writeZipFixture(filepath.Join(projectRoot, "backup-demo-uploads.zip"), map[string]string{
		"uploads/2026/example.txt": "uploaded",
	}); err != nil {
		t.Fatalf("failed to write uploads zip: %v", err)
	}
	if err := writeZipFixture(filepath.Join(projectRoot, "backup-demo-themes.zip"), map[string]string{
		"themes/toet/style.css": "/* theme */",
	}); err != nil {
		t.Fatalf("failed to write themes zip: %v", err)
	}
	if err := writeZipFixture(filepath.Join(projectRoot, "backup-demo-others.zip"), map[string]string{
		"mu-plugins/local.php": "<?php\n",
	}); err != nil {
		t.Fatalf("failed to write others zip: %v", err)
	}

	runner := &fakeRunner{}
	if err := runCreateWithRunner(CreateOptions{Name: "demo", SSHTarget: "user@192.168.0.1 -p 22", RemoteWordPressRoot: testRemoteWordPressRoot}, runner); err != nil {
		t.Fatalf("expected local project backup flow to succeed, got: %v", err)
	}

	paths := create.NewProjectPaths(baseDir, "demo")
	assertProjectSkeleton(t, paths)

	for _, path := range []string{
		filepath.Join(paths.WPContent, "plugins", "acf", "acf.php"),
		filepath.Join(paths.WPContent, "uploads", "2026", "example.txt"),
		filepath.Join(paths.WPContent, "themes", "toet", "style.css"),
		filepath.Join(paths.WPContent, "mu-plugins", "local.php"),
		paths.DatabaseSQL,
	} {
		if _, statErr := os.Stat(path); statErr != nil {
			t.Fatalf("expected %s to exist: %v", path, statErr)
		}
	}

	assertNoCommand(t, runner.commands, "ssh", nil)
	assertNoCommand(t, runner.commands, "scp", nil)
	assertNoCommand(t, runner.commands, "git", []string{"clone", testStarterRepo, "demo"})
	assertNoCommand(t, runner.commands, "lando", []string{"composer", "install"})
	assertCommandCount(t, runner.commands, "lando", []string{"wp", "acorn", "key:generate"}, 0)
	assertHasCommand(t, runner.commands, "lando", []string{"wp", "eval", "echo get_option('stylesheet') ?: get_option('template');"})
	assertHasCommand(t, runner.commands, "lando", []string{"wp", "theme", "activate", "toet"})
}

func TestRunCreateFailsWhenExistingProjectFolderHasNoUpdraftBackups(t *testing.T) {
	baseDir := t.TempDir()
	withWorkingDir(t, baseDir)

	if err := os.Mkdir(filepath.Join(baseDir, "demo"), 0755); err != nil {
		t.Fatalf("failed to prepare existing directory: %v", err)
	}

	err := runCreateWithRunner(CreateOptions{Name: "demo", StarterRepo: testStarterRepo, SSHTarget: "user@192.168.0.1 -p 22", RemoteWordPressRoot: testRemoteWordPressRoot}, &fakeRunner{})
	if err == nil {
		t.Fatal("expected RunCreate to fail for empty existing project directory")
	}
	if !strings.Contains(err.Error(), "contains no recognizable Updraft backup files") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRunCreateFailsWhenExistingProjectFolderContainsProjectMarkers(t *testing.T) {
	baseDir := t.TempDir()
	withWorkingDir(t, baseDir)

	projectRoot := filepath.Join(baseDir, "demo")
	writeCreateTestFile(t, filepath.Join(projectRoot, "backup-demo-db.sql"), "# Home URL: https://local-starter.test\nSELECT 1;\n")
	if err := writeZipFixture(filepath.Join(projectRoot, "backup-demo-plugins.zip"), map[string]string{"plugins/acf/acf.php": "<?php\n"}); err != nil {
		t.Fatalf("failed to write plugins zip: %v", err)
	}
	if err := writeZipFixture(filepath.Join(projectRoot, "backup-demo-uploads.zip"), map[string]string{"uploads/2026/example.txt": "uploaded"}); err != nil {
		t.Fatalf("failed to write uploads zip: %v", err)
	}
	if err := writeZipFixture(filepath.Join(projectRoot, "backup-demo-themes.zip"), map[string]string{"themes/toet/style.css": "/* theme */"}); err != nil {
		t.Fatalf("failed to write themes zip: %v", err)
	}
	writeCreateTestFile(t, filepath.Join(projectRoot, ".lando.yml"), "name: demo\n")

	err := runCreateWithRunner(CreateOptions{Name: "demo", SSHTarget: "user@192.168.0.1 -p 22", RemoteWordPressRoot: testRemoteWordPressRoot}, &fakeRunner{})
	if err == nil {
		t.Fatal("expected existing generated project markers to fail")
	}
	if !strings.Contains(err.Error(), ".lando.yml") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRunCreateDryRunDoesNotWriteFiles(t *testing.T) {
	baseDir := t.TempDir()
	withWorkingDir(t, baseDir)
	runner := &fakeRunner{}

	err := runCreateWithRunner(CreateOptions{Name: "demo", StarterRepo: testStarterRepo, SSHTarget: "user@192.168.0.1 -p 22", RemoteWordPressRoot: testRemoteWordPressRoot, DryRun: true}, runner)
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
	err := runCreateWithRunner(CreateOptions{Name: "demo", StarterRepo: testStarterRepo, SSHTarget: "user@192.168.0.1 -p 22", RemoteWordPressRoot: testRemoteWordPressRoot}, &fakeRunner{runErr: expectedErr, failCmd: "lando"})
	if err == nil {
		t.Fatal("expected runCreateWithRunner to return runner error")
	}
	if !strings.Contains(err.Error(), "lando start failed") {
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
		"TOBA_PHP_VERSION=8.4\n" +
		"TOBA_DOMAIN=stale-from-env.lndo.site\n" +
		"TOBA_STARTER_REPO=" + testStarterRepo + "\n" +
		"TOBA_SSH_TARGET=user@192.168.0.1 -p 22\n" +
		"TOBA_REMOTE_WORDPRESS_ROOT=" + testRemoteWordPressRoot + "\n"
	if err := os.MkdirAll(filepath.Dir(globalEnvPath), 0755); err != nil {
		t.Fatalf("failed to create config dir: %v", err)
	}
	if err := os.WriteFile(globalEnvPath, []byte(env), 0644); err != nil {
		t.Fatalf("failed to write global env file: %v", err)
	}

	if err := runCreateWithRunner(CreateOptions{Name: "demo"}, runner); err != nil {
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

func TestRunCreateWritesConfigSourceToProvidedOutput(t *testing.T) {
	baseDir := t.TempDir()
	withWorkingDir(t, baseDir)
	runner := &fakeRunner{}
	output := &strings.Builder{}

	globalEnvPath, err := create.GlobalEnvPath()
	if err != nil {
		t.Fatalf("GlobalEnvPath returned error: %v", err)
	}

	env := "" +
		"TOBA_PHP_VERSION=8.4\n" +
		"TOBA_STARTER_REPO=" + testStarterRepo + "\n" +
		"TOBA_SSH_TARGET=user@192.168.0.1 -p 22\n" +
		"TOBA_REMOTE_WORDPRESS_ROOT=" + testRemoteWordPressRoot + "\n"
	if err := os.MkdirAll(filepath.Dir(globalEnvPath), 0755); err != nil {
		t.Fatalf("failed to create config dir: %v", err)
	}
	if err := os.WriteFile(globalEnvPath, []byte(env), 0644); err != nil {
		t.Fatalf("failed to write global env file: %v", err)
	}

	if err := runCreateWithIO(CreateOptions{Name: "demo"}, runner, strings.NewReader("n\n"), output); err != nil {
		t.Fatalf("runCreateWithIO returned error: %v", err)
	}

	if !strings.Contains(output.String(), "Using config from "+globalEnvPath) {
		t.Fatalf("expected config source log, got %q", output.String())
	}
}

func TestRunCreateRemoteWordPressRootFlagOverridesEnv(t *testing.T) {
	baseDir := t.TempDir()
	withWorkingDir(t, baseDir)
	runner := &fakeRunner{}

	globalEnvPath, err := create.GlobalEnvPath()
	if err != nil {
		t.Fatalf("GlobalEnvPath returned error: %v", err)
	}

	env := "" +
		"TOBA_STARTER_REPO=" + testStarterRepo + "\n" +
		"TOBA_SSH_TARGET=user@192.168.0.1 -p 22\n" +
		"TOBA_REMOTE_WORDPRESS_ROOT=www/stale-from-env.example\n"
	if err := os.MkdirAll(filepath.Dir(globalEnvPath), 0755); err != nil {
		t.Fatalf("failed to create config dir: %v", err)
	}
	if err := os.WriteFile(globalEnvPath, []byte(env), 0644); err != nil {
		t.Fatalf("failed to write global env file: %v", err)
	}

	if err := runCreateWithRunner(CreateOptions{Name: "demo", RemoteWordPressRoot: testRemoteWordPressRoot}, runner); err != nil {
		t.Fatalf("runCreateWithRunner returned error: %v", err)
	}

	assertHasCommand(t, runner.commands, "ssh", []string{"-p", "22", "user@192.168.0.1", "dynamic:home"})
}

func TestRunCreateNormalizesNameAndPrintsFinalSiteURL(t *testing.T) {
	baseDir := t.TempDir()
	withWorkingDir(t, baseDir)
	runner := &fakeRunner{}
	output := &strings.Builder{}

	err := runCreateWithIO(
		CreateOptions{
			Name:                "Test",
			StarterRepo:         testStarterRepo,
			SSHTarget:           "user@192.168.0.1 -p 22",
			RemoteWordPressRoot: testRemoteWordPressRoot,
		},
		runner,
		strings.NewReader("n\n"),
		output,
	)
	if err != nil {
		t.Fatalf("runCreateWithIO returned error: %v", err)
	}

	paths := create.NewProjectPaths(baseDir, "test")
	if _, statErr := os.Stat(paths.Root); statErr != nil {
		t.Fatalf("expected normalized project root %s to exist: %v", paths.Root, statErr)
	}

	assertHasCommand(t, runner.commands, "git", []string{"clone", testStarterRepo, "test"})
	assertHasCommand(t, runner.commands, "lando", []string{"wp", "theme", "activate", "test"})

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
	if installURL != "--url=test.lndo.site" {
		t.Fatalf("expected normalized install url, got %q", installURL)
	}

	if !strings.Contains(output.String(), "Project ready: https://test.lndo.site") {
		t.Fatalf("expected final site URL in output, got %q", output.String())
	}
}

func TestRunCreateDoesNotRunAcornOptimizeClearForSSHStarter(t *testing.T) {
	baseDir := t.TempDir()
	withWorkingDir(t, baseDir)
	runner := &fakeRunner{}

	err := runCreateWithRunner(CreateOptions{
		Name:                "demo",
		StarterRepo:         testStarterRepo,
		SSHTarget:           "user@192.168.0.1 -p 22",
		RemoteWordPressRoot: testRemoteWordPressRoot,
	}, runner)
	if err != nil {
		t.Fatalf("runCreateWithRunner returned error: %v", err)
	}

	assertNoCommand(t, runner.commands, "lando", []string{"wp", "acorn", "optimize:clear"})
}

func TestRunCreateDoesNotRunAcornOptimizeClearForLocalBackup(t *testing.T) {
	baseDir := t.TempDir()
	withWorkingDir(t, baseDir)

	projectRoot := filepath.Join(baseDir, "demo")
	writeCreateTestFile(t, filepath.Join(projectRoot, "backup-demo-db.sql"), "# Home URL: https://local-starter.test\nSELECT 1;\n")
	if err := writeZipFixture(filepath.Join(projectRoot, "backup-demo-plugins.zip"), map[string]string{
		"plugins/acf/acf.php": "<?php\n",
	}); err != nil {
		t.Fatalf("failed to write plugins zip: %v", err)
	}
	if err := writeZipFixture(filepath.Join(projectRoot, "backup-demo-uploads.zip"), map[string]string{
		"uploads/2026/example.txt": "uploaded",
	}); err != nil {
		t.Fatalf("failed to write uploads zip: %v", err)
	}
	if err := writeZipFixture(filepath.Join(projectRoot, "backup-demo-themes.zip"), map[string]string{
		"themes/toet/style.css": "/* theme */",
	}); err != nil {
		t.Fatalf("failed to write themes zip: %v", err)
	}
	if err := writeZipFixture(filepath.Join(projectRoot, "backup-demo-others.zip"), map[string]string{
		"mu-plugins/local.php": "<?php\n",
	}); err != nil {
		t.Fatalf("failed to write others zip: %v", err)
	}

	runner := &fakeRunner{}
	if err := runCreateWithRunner(CreateOptions{Name: "demo", SSHTarget: "user@192.168.0.1 -p 22", RemoteWordPressRoot: testRemoteWordPressRoot}, runner); err != nil {
		t.Fatalf("expected local project backup flow to succeed, got: %v", err)
	}

	assertNoCommand(t, runner.commands, "lando", []string{"wp", "acorn", "optimize:clear"})
}

func TestRunCreateCleansUpFailedInstallWhenConfirmed(t *testing.T) {
	baseDir := t.TempDir()
	withWorkingDir(t, baseDir)

	input := strings.NewReader("y\n")
	output := &strings.Builder{}
	err := runCreateWithIO(
		CreateOptions{Name: "demo", SSHTarget: "user@192.168.0.1 -p 22", RemoteWordPressRoot: testRemoteWordPressRoot},
		&fakeRunner{runErrByCommand: map[string]error{"lando start": fmt.Errorf("lando failed")}},
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

func TestRunCreateKeepsProjectDirWhenLandoDestroyFails(t *testing.T) {
	baseDir := t.TempDir()
	withWorkingDir(t, baseDir)

	input := strings.NewReader("y\n")
	output := &strings.Builder{}
	err := runCreateWithIO(
		CreateOptions{Name: "demo", SSHTarget: "user@192.168.0.1 -p 22", RemoteWordPressRoot: testRemoteWordPressRoot},
		&fakeRunner{runErrByCommand: map[string]error{
			"lando start":      fmt.Errorf("lando failed"),
			"lando destroy -y": fmt.Errorf("destroy failed"),
		}},
		input,
		output,
	)
	if err == nil {
		t.Fatal("expected create to fail")
	}

	projectRoot := filepath.Join(baseDir, "demo")
	if _, statErr := os.Stat(projectRoot); statErr != nil {
		t.Fatalf("expected project root to remain after destroy failure, got err=%v", statErr)
	}
	if !strings.Contains(output.String(), "Failed to destroy Lando app in "+projectRoot) {
		t.Fatalf("expected destroy error, got: %s", output.String())
	}
	if !strings.Contains(output.String(), "Keeping project directory because removing it now could make manual Lando cleanup harder") {
		t.Fatalf("expected keep-directory error, got: %s", output.String())
	}
}

func TestRunCreateFailsWithoutGlobalConfig(t *testing.T) {
	baseDir := t.TempDir()
	withWorkingDir(t, baseDir)

	err := runCreateWithRunner(CreateOptions{}, &fakeRunner{})
	if err == nil {
		t.Fatal("expected missing global config error")
	}
	if !strings.Contains(err.Error(), "toba create <project-name>") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRunCreateFailsWhenStarterRepoIsMissing(t *testing.T) {
	baseDir := t.TempDir()
	withWorkingDir(t, baseDir)

	err := runCreateWithRunner(CreateOptions{Name: "demo", SSHTarget: "user@192.168.0.1 -p 22", RemoteWordPressRoot: testRemoteWordPressRoot}, &fakeRunner{})
	if err == nil {
		t.Fatal("expected missing starter repo error")
	}
	if !strings.Contains(err.Error(), "TOBA_STARTER_REPO") || !strings.Contains(err.Error(), "--starter-repo") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRunCreateFailsWhenSSHTargetIsMissing(t *testing.T) {
	baseDir := t.TempDir()
	withWorkingDir(t, baseDir)

	globalEnvPath, err := create.GlobalEnvPath()
	if err != nil {
		t.Fatalf("GlobalEnvPath returned error: %v", err)
	}
	if err := os.MkdirAll(filepath.Dir(globalEnvPath), 0755); err != nil {
		t.Fatalf("failed to create config dir: %v", err)
	}
	if err := os.WriteFile(globalEnvPath, []byte("TOBA_STARTER_REPO="+testStarterRepo+"\n"), 0644); err != nil {
		t.Fatalf("failed to write global env file: %v", err)
	}

	err = runCreateWithRunner(CreateOptions{Name: "demo"}, &fakeRunner{})
	if err == nil {
		t.Fatal("expected missing SSH target error")
	}
	if !strings.Contains(err.Error(), "TOBA_SSH_TARGET") || !strings.Contains(err.Error(), "--ssh-target") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRunCreateFailsWhenRemoteWordPressRootIsMissing(t *testing.T) {
	baseDir := t.TempDir()
	withWorkingDir(t, baseDir)

	globalEnvPath, err := create.GlobalEnvPath()
	if err != nil {
		t.Fatalf("GlobalEnvPath returned error: %v", err)
	}
	if err := os.MkdirAll(filepath.Dir(globalEnvPath), 0755); err != nil {
		t.Fatalf("failed to create config dir: %v", err)
	}
	if err := os.WriteFile(globalEnvPath, []byte("TOBA_STARTER_REPO="+testStarterRepo+"\nTOBA_SSH_TARGET=user@192.168.0.1 -p 22\n"), 0644); err != nil {
		t.Fatalf("failed to write global env file: %v", err)
	}

	err = runCreateWithRunner(CreateOptions{Name: "demo"}, &fakeRunner{})
	if err == nil {
		t.Fatal("expected missing remote WordPress root error")
	}
	if !strings.Contains(err.Error(), "TOBA_REMOTE_WORDPRESS_ROOT") || !strings.Contains(err.Error(), "--remote-wordpress-root") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRunCreateFailsForProjectNameWithSpaces(t *testing.T) {
	baseDir := t.TempDir()
	withWorkingDir(t, baseDir)

	err := runCreateWithRunner(CreateOptions{Name: "demo project", StarterRepo: testStarterRepo, SSHTarget: "user@192.168.0.1 -p 22", RemoteWordPressRoot: testRemoteWordPressRoot}, &fakeRunner{})
	if err == nil {
		t.Fatal("expected project name validation error")
	}
	if !strings.Contains(err.Error(), "project name cannot contain spaces") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func assertProjectSkeleton(t *testing.T, paths create.ProjectPaths) {
	t.Helper()

	for _, dir := range []string{paths.Root, paths.AppDir, paths.ConfigDir} {
		info, statErr := os.Stat(dir)
		if statErr != nil {
			t.Fatalf("expected %s to exist: %v", dir, statErr)
		}
		if !info.IsDir() {
			t.Fatalf("expected %s to be a directory", dir)
		}
	}
}

func assertStaticConfigs(t *testing.T, paths create.ProjectPaths) {
	t.Helper()

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

	phpINIPath := filepath.Join(paths.ConfigDir, "php.ini")
	phpINI, readPHPINIError := os.ReadFile(phpINIPath)
	if readPHPINIError != nil {
		t.Fatalf("expected %s to exist: %v", phpINIPath, readPHPINIError)
	}
	if !strings.Contains(string(phpINI), "xdebug.mode = profile,debug,develop") {
		t.Fatalf("expected generated php.ini to contain xdebug mode, got:\n%s", string(phpINI))
	}

	wpCLIPath := filepath.Join(paths.Root, "wp-cli.yml")
	wpCLIConfig, readWPCLIError := os.ReadFile(wpCLIPath)
	if readWPCLIError != nil {
		t.Fatalf("expected %s to exist: %v", wpCLIPath, readWPCLIError)
	}
	if string(wpCLIConfig) != "path: app\nserver:\n  docroot: app\napache_modules:\n  - mod_rewrite\n" {
		t.Fatalf("unexpected wp-cli.yml contents:\n%s", string(wpCLIConfig))
	}
}

func assertHasCommand(t *testing.T, commands []recordedCommand, cmd string, args []string) {
	t.Helper()

	for _, command := range commands {
		if command.cmd != cmd {
			continue
		}
		if argsMatch(command.args, args) {
			return
		}
	}

	t.Fatalf("expected command %s %v, got %#v", cmd, args, commands)
}

func assertNoCommand(t *testing.T, commands []recordedCommand, cmd string, args []string) {
	t.Helper()

	for _, command := range commands {
		if command.cmd != cmd {
			continue
		}
		if args == nil || argsMatch(command.args, args) {
			t.Fatalf("did not expect command %s %v, got %#v", cmd, args, commands)
		}
	}
}

func assertCommandCount(t *testing.T, commands []recordedCommand, cmd string, args []string, expected int) {
	t.Helper()

	count := 0
	for _, command := range commands {
		if command.cmd != cmd {
			continue
		}
		if argsMatch(command.args, args) {
			count++
		}
	}

	if count != expected {
		t.Fatalf("expected %d occurrences of %s %v, got %d in %#v", expected, cmd, args, count, commands)
	}
}

func argsMatch(actual []string, expected []string) bool {
	if len(actual) != len(expected) {
		return false
	}

	for i := range actual {
		if strings.HasPrefix(expected[i], "dynamic:") {
			if !matchesDynamicArg(actual[i], strings.TrimPrefix(expected[i], "dynamic:")) {
				return false
			}
			continue
		}
		if actual[i] != expected[i] {
			return false
		}
	}

	return true
}

func matchesDynamicArg(actual string, pattern string) bool {
	switch pattern {
	case "home":
		return strings.Contains(actual, "wp84 option get home") && strings.Contains(actual, "'"+testRemoteWordPressRoot+"'")
	case "remote-sql":
		return strings.HasPrefix(actual, "user@192.168.0.1:"+testRemoteWordPressRoot+"/") && strings.HasSuffix(actual, ".sql")
	case "local-sql":
		return strings.HasSuffix(actual, ".sql")
	default:
		return false
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
		return os.WriteFile(localTarget, []byte("# Home URL: https://starter.example.test\nSELECT 1;\n"), 0644)
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

func writeCreateTestFile(t *testing.T, path string, content string) {
	t.Helper()

	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatalf("failed to create parent dir for %s: %v", path, err)
	}
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write %s: %v", path, err)
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
