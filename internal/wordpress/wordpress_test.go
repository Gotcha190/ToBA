package wordpress

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/gotcha190/toba/internal/create"
)

type recordedCommand struct {
	dir  string
	cmd  string
	args []string
}

type fakeRunner struct {
	commands    []recordedCommand
	err         error
	outputs     map[string]string
	captureErrs map[string]error
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
	r.commands = append(r.commands, recordedCommand{
		dir:  dir,
		cmd:  cmd,
		args: append([]string(nil), args...),
	})
	if r.captureErrs != nil {
		if err, ok := r.captureErrs[cmd+" "+strings.Join(args, " ")]; ok {
			return r.outputs[cmd+" "+strings.Join(args, " ")], err
		}
	}
	if r.err != nil {
		return "", r.err
	}
	return r.outputs[cmd+" "+strings.Join(args, " ")], nil
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

func TestBackupSourceURLHandlesVeryLongLines(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "database.sql")
	content := strings.Repeat("x", 128*1024) + "\n" +
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

func TestBackupTablePrefixPrefersMetadata(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "database.sql")
	content := "" +
		"# Table prefix: txxbt_\n" +
		"CREATE TABLE `txxbt_options` (\n"
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write SQL file: %v", err)
	}

	prefix, err := BackupTablePrefix(path)
	if err != nil {
		t.Fatalf("BackupTablePrefix returned error: %v", err)
	}
	if prefix != "txxbt_" {
		t.Fatalf("unexpected table prefix: %q", prefix)
	}
}

func TestBackupTablePrefixFallsBackToDefault(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "database.sql")
	if err := os.WriteFile(path, []byte("SELECT 1;\n"), 0644); err != nil {
		t.Fatalf("failed to write SQL file: %v", err)
	}

	prefix, err := BackupTablePrefix(path)
	if err != nil {
		t.Fatalf("BackupTablePrefix returned error: %v", err)
	}
	if prefix != DefaultTablePrefix {
		t.Fatalf("unexpected table prefix: %q", prefix)
	}
}

func TestBackupTablePrefixHandlesVeryLongLines(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "database.sql")
	content := strings.Repeat("x", 128*1024) + "\n" +
		"CREATE TABLE `txxbt_options` (\n"
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write SQL file: %v", err)
	}

	prefix, err := BackupTablePrefix(path)
	if err != nil {
		t.Fatalf("BackupTablePrefix returned error: %v", err)
	}
	if prefix != "txxbt_" {
		t.Fatalf("unexpected table prefix: %q", prefix)
	}
}

func TestSetConfigTablePrefixUpdatesFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "wp-config.php")
	content := "<?php\n$table_prefix = 'wp_';\n"
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write wp-config: %v", err)
	}

	if err := SetConfigTablePrefix(path, "txxbt_"); err != nil {
		t.Fatalf("SetConfigTablePrefix returned error: %v", err)
	}

	updated, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read wp-config: %v", err)
	}
	if !strings.Contains(string(updated), "$table_prefix = 'txxbt_';") {
		t.Fatalf("expected updated table prefix, got:\n%s", string(updated))
	}
}

func TestSetProjectConfigIncludeUpdatesFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "wp-config.php")
	content := "" +
		"<?php\n" +
		"/* Add any custom values between this line and the \"stop editing\" line. */\n\n" +
		"if ( ! defined( 'WP_DEBUG' ) ) {\n" +
		"\tdefine( 'WP_DEBUG', false );\n" +
		"}\n\n" +
		"/** Sets up WordPress vars and included files. */\n" +
		"require_once ABSPATH . 'wp-settings.php';\n"
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write wp-config: %v", err)
	}

	if err := setProjectConfigInclude(path); err != nil {
		t.Fatalf("setProjectConfigInclude returned error: %v", err)
	}

	updated, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read wp-config: %v", err)
	}
	if !strings.Contains(string(updated), "require_once dirname( __DIR__ ) . '/config.php';") {
		t.Fatalf("expected config include, got:\n%s", string(updated))
	}
	if strings.Index(string(updated), "require_once dirname( __DIR__ ) . '/config.php';") > strings.Index(string(updated), "define( 'WP_DEBUG', false );") {
		t.Fatalf("expected config include before WP_DEBUG default, got:\n%s", string(updated))
	}
}

func TestInstallIncludesProjectConfigWhenPresent(t *testing.T) {
	dir := t.TempDir()
	runner := &fakeRunner{}
	config := create.ProjectConfig{
		Name:   "my-project",
		Domain: "my-project.lndo.site",
	}

	if err := os.MkdirAll(filepath.Join(dir, "app"), 0755); err != nil {
		t.Fatalf("failed to create app dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "config.php"), []byte("<?php\n"), 0644); err != nil {
		t.Fatalf("failed to write config.php: %v", err)
	}
	wpConfigPath := filepath.Join(dir, "app", "wp-config.php")
	wpConfig := "" +
		"<?php\n" +
		"/* Add any custom values between this line and the \"stop editing\" line. */\n\n" +
		"/** Sets up WordPress vars and included files. */\n" +
		"require_once ABSPATH . 'wp-settings.php';\n"
	if err := os.WriteFile(wpConfigPath, []byte(wpConfig), 0644); err != nil {
		t.Fatalf("failed to write wp-config.php: %v", err)
	}

	if err := Install(runner, dir, config); err != nil {
		t.Fatalf("Install returned error: %v", err)
	}

	updated, err := os.ReadFile(wpConfigPath)
	if err != nil {
		t.Fatalf("failed to read wp-config.php: %v", err)
	}
	if !strings.Contains(string(updated), "require_once dirname( __DIR__ ) . '/config.php';") {
		t.Fatalf("expected config include, got:\n%s", string(updated))
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
	runner := &fakeRunner{
		outputs: map[string]string{
			"lando wp user get tamago --field=ID": "16\n",
		},
	}

	if err := ResetAdminPassword(runner, "/tmp/demo"); err != nil {
		t.Fatalf("ResetAdminPassword returned error: %v", err)
	}

	expected := []recordedCommand{
		{
			dir:  "/tmp/demo",
			cmd:  "lando",
			args: []string{"wp", "user", "get", "tamago", "--field=ID"},
		},
		{
			dir:  "/tmp/demo",
			cmd:  "lando",
			args: []string{"wp", "user", "update", "tamago", "--user_pass=tamago"},
		},
	}

	if !reflect.DeepEqual(runner.commands, expected) {
		t.Fatalf("unexpected command sequence:\nexpected: %#v\ngot: %#v", expected, runner.commands)
	}
}

func TestResetAdminPasswordCreatesTamagoWhenMissing(t *testing.T) {
	runner := &fakeRunner{
		outputs: map[string]string{
			"lando wp user get tamago --field=ID": "Error: Invalid user ID, email or login: 'tamago'\n",
		},
		captureErrs: map[string]error{
			"lando wp user get tamago --field=ID": errors.New("missing user"),
		},
	}

	if err := ResetAdminPassword(runner, "/tmp/demo"); err != nil {
		t.Fatalf("ResetAdminPassword returned error: %v", err)
	}

	expected := []recordedCommand{
		{
			dir:  "/tmp/demo",
			cmd:  "lando",
			args: []string{"wp", "user", "get", "tamago", "--field=ID"},
		},
		{
			dir:  "/tmp/demo",
			cmd:  "lando",
			args: []string{"wp", "user", "create", "tamago", "email@email.pl", "--role=administrator", "--user_pass=tamago", "--display_name=tamago"},
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

func TestDetectImportedThemeSlugPrefersStylesheet(t *testing.T) {
	runner := &fakeRunner{
		outputs: map[string]string{
			"lando wp option get stylesheet": "sage\n",
		},
	}

	got, err := DetectImportedThemeSlug(runner, "/tmp/demo")
	if err != nil {
		t.Fatalf("DetectImportedThemeSlug returned error: %v", err)
	}
	if got != "sage" {
		t.Fatalf("unexpected theme slug: %q", got)
	}
}

func TestDetectImportedThemeSlugFallsBackToTemplate(t *testing.T) {
	runner := &fakeRunner{
		outputs: map[string]string{
			"lando wp option get stylesheet": "\n",
			"lando wp option get template":   "sage-fallback\n",
		},
	}

	got, err := DetectImportedThemeSlug(runner, "/tmp/demo")
	if err != nil {
		t.Fatalf("DetectImportedThemeSlug returned error: %v", err)
	}
	if got != "sage-fallback" {
		t.Fatalf("unexpected theme slug: %q", got)
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

func TestRefreshThemeCachesRunsExpectedCommands(t *testing.T) {
	runner := &fakeRunner{
		outputs: map[string]string{
			"lando wp acorn list": "  optimize         Cache the framework bootstrap files\n  cache:clear      Flush the application cache\n  acf:cache        Cache ACF assets\n",
		},
	}

	if err := RefreshThemeCaches(runner, "/tmp/demo"); err != nil {
		t.Fatalf("RefreshThemeCaches returned error: %v", err)
	}

	expected := []recordedCommand{
		{
			dir:  "/tmp/demo",
			cmd:  "lando",
			args: []string{"wp", "acorn", "list"},
		},
		{
			dir:  "/tmp/demo",
			cmd:  "lando",
			args: []string{"wp", "acorn", "optimize"},
		},
		{
			dir:  "/tmp/demo",
			cmd:  "lando",
			args: []string{"wp", "acorn", "cache:clear"},
		},
		{
			dir:  "/tmp/demo",
			cmd:  "lando",
			args: []string{"wp", "acorn", "acf:cache"},
		},
	}

	if !reflect.DeepEqual(runner.commands, expected) {
		t.Fatalf("unexpected command sequence:\nexpected: %#v\ngot: %#v", expected, runner.commands)
	}
}

func TestRefreshThemeCachesSkipsUnavailableCommands(t *testing.T) {
	runner := &fakeRunner{
		outputs: map[string]string{
			"lando wp acorn list": "  optimize:clear   Remove the cached bootstrap files\n",
		},
	}

	if err := RefreshThemeCaches(runner, "/tmp/demo"); err != nil {
		t.Fatalf("RefreshThemeCaches returned error: %v", err)
	}

	expected := []recordedCommand{
		{
			dir:  "/tmp/demo",
			cmd:  "lando",
			args: []string{"wp", "acorn", "list"},
		},
		{
			dir:  "/tmp/demo",
			cmd:  "lando",
			args: []string{"wp", "acorn", "optimize:clear"},
		},
	}

	if !reflect.DeepEqual(runner.commands, expected) {
		t.Fatalf("unexpected command sequence:\nexpected: %#v\ngot: %#v", expected, runner.commands)
	}
}

func TestRefreshThemeCachesFallsBackToConfigClear(t *testing.T) {
	runner := &fakeRunner{
		outputs: map[string]string{
			"lando wp acorn list": "  config:clear     Remove the configuration cache file\n",
		},
	}

	if err := RefreshThemeCaches(runner, "/tmp/demo"); err != nil {
		t.Fatalf("RefreshThemeCaches returned error: %v", err)
	}

	expected := []recordedCommand{
		{
			dir:  "/tmp/demo",
			cmd:  "lando",
			args: []string{"wp", "acorn", "list"},
		},
		{
			dir:  "/tmp/demo",
			cmd:  "lando",
			args: []string{"wp", "acorn", "config:clear"},
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
