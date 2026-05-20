package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gotcha190/toba/internal/create"
)

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
	assertHasCommandInDir(t, runner.commands, filepath.Join(paths.Themes, "demo"), "git", []string{"remote", "remove", "origin"})
	assertHasCommandInDir(t, runner.commands, filepath.Join(paths.Themes, "demo"), "git", []string{"branch", "-M", "develop"})
	assertHasCommandInDir(t, runner.commands, filepath.Join(paths.Themes, "demo"), "git", []string{"branch", "-f", "starter", "develop"})
	assertHasCommand(t, runner.commands, "lando", []string{"composer", "install", "--no-interaction", "--prefer-dist", "--optimize-autoloader", "--no-progress"})
	assertHasCommand(t, runner.commands, "npm", []string{"ci", "--no-audit", "--no-fund"})
	assertHasCommand(t, runner.commands, "npm", []string{"run", "build"})
	assertHasCommand(t, runner.commands, "lando", []string{"wp", "theme", "activate", "demo"})
	assertCommandCount(t, runner.commands, "lando", []string{"wp", "acorn", "key:generate"}, 2)
	assertNoCommand(t, runner.commands, "lando", []string{"wp", "eval", "echo get_option('stylesheet') ?: get_option('template');"})
}

func TestRunCreateSucceedsWhenProjectGitRepoUnavailable(t *testing.T) {
	baseDir := t.TempDir()
	withWorkingDir(t, baseDir)
	runner := &fakeRunner{
		runErrByCommand: map[string]error{
			"git ls-remote git@example.com:company/demo.git": fmt.Errorf("repo missing"),
		},
	}

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
	assertHasCommandInDir(t, runner.commands, filepath.Join(paths.Themes, "demo"), "git", []string{"ls-remote", "git@example.com:company/demo.git"})
	assertNoCommand(t, runner.commands, "git", []string{"push", "-u", "origin", "develop", "starter"})
}

func TestRunCreateSucceedsWhenProjectGitPushFails(t *testing.T) {
	baseDir := t.TempDir()
	withWorkingDir(t, baseDir)
	runner := &fakeRunner{
		runErrByCommand: map[string]error{
			"git push -u origin develop starter": fmt.Errorf("push failed"),
		},
	}

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
	assertHasCommandInDir(t, runner.commands, filepath.Join(paths.Themes, "demo"), "git", []string{"push", "-u", "origin", "develop", "starter"})
}

func TestRunCreateNoUploadsUsesSSHFallback(t *testing.T) {
	baseDir := t.TempDir()
	withWorkingDir(t, baseDir)
	runner := &fakeRunner{}

	err := runCreateWithRunner(CreateOptions{
		Name:                "demo",
		PHPVersion:          "8.4",
		StarterRepo:         testStarterRepo,
		SSHTarget:           "user@192.168.0.1 -p 22",
		RemoteWordPressRoot: testRemoteWordPressRoot,
		NoUploads:           true,
	}, runner)
	if err != nil {
		t.Fatalf("RunCreate returned error: %v", err)
	}

	paths := create.NewProjectPaths(baseDir, "demo")
	assertProjectSkeleton(t, paths)
	if _, statErr := os.Stat(filepath.Join(paths.WPContent, "uploads", "2025", "07", "example.jpg")); !os.IsNotExist(statErr) {
		t.Fatalf("expected remote uploads fixture not to be extracted, stat error=%v", statErr)
	}

	htaccess, err := os.ReadFile(filepath.Join(paths.AppDir, ".htaccess"))
	if err != nil {
		t.Fatalf("expected htaccess fallback file: %v", err)
	}
	for _, expected := range []string{
		"# BEGIN ToBA Uploads Fallback",
		"RewriteCond %{HTTP_HOST} !^starter\\.example\\.test$ [NC]",
		"RewriteRule ^wp-content/uploads/(.*)$ https://starter.example.test/wp-content/uploads/$1 [R=302,NE,L]",
	} {
		if !strings.Contains(string(htaccess), expected) {
			t.Fatalf("expected htaccess to contain %q, got:\n%s", expected, string(htaccess))
		}
	}

	assertHasCommand(t, runner.commands, "ssh", []string{"-p", "22", "user@192.168.0.1", "dynamic:no-uploads-home"})
	assertNoCommand(t, runner.commands, "scp", []string{"-P", "22", "dynamic:remote-uploads", "dynamic:local-uploads"})
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

func TestRunCreateNoUploadsRejectsLocalBackupMode(t *testing.T) {
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

	err := runCreateWithRunner(CreateOptions{Name: "demo", NoUploads: true}, &fakeRunner{})
	if err == nil {
		t.Fatal("expected no-uploads local mode error")
	}
	if !strings.Contains(err.Error(), "--no-uploads is only supported with SSH starter data") {
		t.Fatalf("unexpected error: %v", err)
	}
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
	assertHasCommandInDir(t, runner.commands, filepath.Join(paths.Themes, "test"), "git", []string{"remote", "remove", "origin"})
	assertHasCommandInDir(t, runner.commands, filepath.Join(paths.Themes, "test"), "git", []string{"branch", "-M", "develop"})
	assertHasCommandInDir(t, runner.commands, filepath.Join(paths.Themes, "test"), "git", []string{"branch", "-f", "starter", "develop"})
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
	if !strings.Contains(output.String(), "[INFO] Total create time: ") {
		t.Fatalf("expected total create time in output, got %q", output.String())
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

func TestRunCreateWithIOLogsSequentialModeWhenEnabled(t *testing.T) {
	baseDir := t.TempDir()
	withWorkingDir(t, baseDir)
	runner := &fakeRunner{}
	output := &strings.Builder{}

	err := runCreateWithIO(
		CreateOptions{
			Name:                "demo",
			StarterRepo:         testStarterRepo,
			SSHTarget:           "user@192.168.0.1 -p 22",
			RemoteWordPressRoot: testRemoteWordPressRoot,
			Sequential:          true,
		},
		runner,
		strings.NewReader("n\n"),
		output,
	)
	if err != nil {
		t.Fatalf("runCreateWithIO returned error: %v", err)
	}

	if !strings.Contains(output.String(), "Sequential create mode enabled") {
		t.Fatalf("expected sequential mode log, got %q", output.String())
	}
	if !strings.Contains(output.String(), "[INFO] Total create time: ") {
		t.Fatalf("expected total create time in output, got %q", output.String())
	}
}
