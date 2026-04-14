package wordpress

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
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

func TestInstallRunsExpectedCommands(t *testing.T) {
	runner := &fakeRunner{}
	config := create.ProjectConfig{
		Name:   "my-project",
		Domain: "my-project.lndo.site",
	}

	err := Install(runner, "/tmp/demo", config)
	if err != nil {
		t.Fatalf("Install returned error: %v", err)
	}

	expected := []recordedCommand{
		{
			dir:  "/tmp/demo",
			cmd:  "lando",
			args: []string{"wp", "core", "download", "--locale=pl_PL"},
		},
		{
			dir:  "/tmp/demo",
			cmd:  "lando",
			args: []string{"wp", "config", "create", "--dbname=wordpress", "--dbuser=wordpress", "--dbpass=wordpress", "--dbhost=database", "--dbcharset=utf8mb4"},
		},
		{
			dir:  "/tmp/demo",
			cmd:  "lando",
			args: []string{"wp", "core", "install", "--url=my-project.lndo.site", "--title=My Project", "--admin_user=tamago", "--admin_email=email@email.pl", "--admin_password=tamago"},
		},
	}

	if !reflect.DeepEqual(runner.commands, expected) {
		t.Fatalf("unexpected command sequence:\nexpected: %#v\ngot: %#v", expected, runner.commands)
	}
}

func TestInstallReturnsCommandError(t *testing.T) {
	expectedErr := errors.New("boom")
	runner := &fakeRunner{err: expectedErr}

	err := Install(runner, "/tmp/demo", create.ProjectConfig{Name: "demo", Domain: "demo.lndo.site"})
	if !errors.Is(err, expectedErr) {
		t.Fatalf("expected %v, got %v", expectedErr, err)
	}
}

func TestProjectTitle(t *testing.T) {
	if got := ProjectTitle("my-project_name"); got != "My Project Name" {
		t.Fatalf("unexpected title: %q", got)
	}
}

func TestImportDatabaseRunsExpectedCommand(t *testing.T) {
	runner := &fakeRunner{}

	if err := ImportDatabase(runner, "/tmp/demo", "/tmp/demo/app/database.sql"); err != nil {
		t.Fatalf("ImportDatabase returned error: %v", err)
	}

	expected := []recordedCommand{
		{
			dir:  "/tmp/demo",
			cmd:  "lando",
			args: []string{"db-import", "app/database.sql"},
		},
	}

	if !reflect.DeepEqual(runner.commands, expected) {
		t.Fatalf("unexpected command sequence:\nexpected: %#v\ngot: %#v", expected, runner.commands)
	}
}

func TestBackupSourceURLPrefersHomeURL(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "database.sql")
	content := "" +
		"# Backup of: https://old.example.com\n" +
		"# Home URL: https://preferred.example.com\n"
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write SQL file: %v", err)
	}

	sourceURL, err := BackupSourceURL(path)
	if err != nil {
		t.Fatalf("BackupSourceURL returned error: %v", err)
	}
	if sourceURL != "https://preferred.example.com" {
		t.Fatalf("unexpected source URL: %q", sourceURL)
	}
}

func TestSearchReplaceRunsExpectedCommand(t *testing.T) {
	runner := &fakeRunner{}

	if err := SearchReplace(runner, "/tmp/demo", "https://old.example.com", "https://demo.lndo.site"); err != nil {
		t.Fatalf("SearchReplace returned error: %v", err)
	}

	expected := []recordedCommand{
		{
			dir: "/tmp/demo",
			cmd: "lando",
			args: []string{
				"wp",
				"search-replace",
				"https://old.example.com",
				"https://demo.lndo.site",
				"--all-tables-with-prefix",
				"--skip-columns=guid",
			},
		},
	}

	if !reflect.DeepEqual(runner.commands, expected) {
		t.Fatalf("unexpected command sequence:\nexpected: %#v\ngot: %#v", expected, runner.commands)
	}
}

func TestResetAdminPasswordRunsExpectedCommand(t *testing.T) {
	runner := &fakeRunner{}

	if err := ResetAdminPassword(runner, "/tmp/demo"); err != nil {
		t.Fatalf("ResetAdminPassword returned error: %v", err)
	}

	expected := []recordedCommand{
		{
			dir:  "/tmp/demo",
			cmd:  "lando",
			args: []string{"wp", "user", "update", "1", "--user_pass=tamago"},
		},
	}

	if !reflect.DeepEqual(runner.commands, expected) {
		t.Fatalf("unexpected command sequence:\nexpected: %#v\ngot: %#v", expected, runner.commands)
	}
}

func TestActivateThemeRunsExpectedCommand(t *testing.T) {
	runner := &fakeRunner{}

	if err := ActivateTheme(runner, "/tmp/demo", "demo"); err != nil {
		t.Fatalf("ActivateTheme returned error: %v", err)
	}

	expected := []recordedCommand{
		{
			dir:  "/tmp/demo",
			cmd:  "lando",
			args: []string{"wp", "theme", "activate", "demo"},
		},
	}

	if !reflect.DeepEqual(runner.commands, expected) {
		t.Fatalf("unexpected command sequence:\nexpected: %#v\ngot: %#v", expected, runner.commands)
	}
}

func TestFlushRewriteRulesRunsExpectedCommand(t *testing.T) {
	runner := &fakeRunner{}

	if err := FlushRewriteRules(runner, "/tmp/demo"); err != nil {
		t.Fatalf("FlushRewriteRules returned error: %v", err)
	}

	expected := []recordedCommand{
		{
			dir:  "/tmp/demo",
			cmd:  "lando",
			args: []string{"wp", "rewrite", "flush", "--hard"},
		},
	}

	if !reflect.DeepEqual(runner.commands, expected) {
		t.Fatalf("unexpected command sequence:\nexpected: %#v\ngot: %#v", expected, runner.commands)
	}
}

func TestLocalHTTPSURLNormalizesScheme(t *testing.T) {
	got, err := LocalHTTPSURL("http://demo.lndo.site")
	if err != nil {
		t.Fatalf("LocalHTTPSURL returned error: %v", err)
	}
	if got != "https://demo.lndo.site" {
		t.Fatalf("unexpected URL: %q", got)
	}
}
