package steps

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/gotcha190/ToBA/internal/create"
)

type restoreTestRunner struct{}

func (r restoreTestRunner) Run(dir string, cmd string, args ...string) error {
	return nil
}

func (r restoreTestRunner) CaptureOutput(dir string, cmd string, args ...string) (string, error) {
	return "", nil
}

func TestImportPluginsStepExtractsMultipleArchives(t *testing.T) {
	ctx := newRestoreTestContext(t)

	if err := NewImportPluginsStep().Run(ctx); err != nil {
		t.Fatalf("ImportPluginsStep returned error: %v", err)
	}

	for _, path := range []string{
		filepath.Join(ctx.Paths.Plugins, "advanced-custom-fields-pro"),
		filepath.Join(ctx.Paths.Plugins, "wp-optimize"),
	} {
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("expected %s to exist: %v", path, err)
		}
	}
}

func TestImportOthersStepExtractsIntoWPContentWithoutNestedExpansion(t *testing.T) {
	ctx := newRestoreTestContext(t)

	if err := NewImportOthersStep().Run(ctx); err != nil {
		t.Fatalf("ImportOthersStep returned error: %v", err)
	}

	if _, err := os.Stat(filepath.Join(ctx.Paths.WPContent, "uploads.zip")); err != nil {
		t.Fatalf("expected uploads.zip to exist: %v", err)
	}
	if _, err := os.Stat(filepath.Join(ctx.Paths.WPContent, "cache", "acorn")); err != nil {
		t.Fatalf("expected cache/acorn to exist: %v", err)
	}
	if _, err := os.Stat(filepath.Join(ctx.Paths.WPContent, "uploads")); !os.IsNotExist(err) {
		t.Fatalf("expected nested uploads archive to stay compressed, got err=%v", err)
	}
}

func TestImportDatabaseStepWritesSQLAndRunsRewrite(t *testing.T) {
	runner := &recordingRunner{}
	ctx := newRestoreTestContext(t)
	ctx.Runner = runner
	ctx.Config.Domain = "demo.lndo.site"

	if err := NewImportDatabaseStep().Run(ctx); err != nil {
		t.Fatalf("ImportDatabaseStep returned error: %v", err)
	}

	if _, err := os.Stat(ctx.Paths.DatabaseSQL); err != nil {
		t.Fatalf("expected %s to exist: %v", ctx.Paths.DatabaseSQL, err)
	}
	if len(runner.commands) != 2 {
		t.Fatalf("expected 2 commands, got %d", len(runner.commands))
	}
	if got := runner.commands[0].args; len(got) < 2 || got[0] != "db-import" || got[1] != "app/database.sql" {
		t.Fatalf("unexpected db-import args: %#v", got)
	}
	if got := runner.commands[1].args; len(got) < 4 || got[2] != "https://starter.tamago-dev.pl" || got[3] != "https://demo.lndo.site" {
		t.Fatalf("unexpected search-replace args: %#v", got)
	}
}

type recordedCommand struct {
	dir  string
	cmd  string
	args []string
}

type recordingRunner struct {
	commands []recordedCommand
}

func (r *recordingRunner) Run(dir string, cmd string, args ...string) error {
	r.commands = append(r.commands, recordedCommand{
		dir:  dir,
		cmd:  cmd,
		args: append([]string(nil), args...),
	})
	return nil
}

func (r *recordingRunner) CaptureOutput(dir string, cmd string, args ...string) (string, error) {
	return "", nil
}

func newRestoreTestContext(t *testing.T) *create.Context {
	t.Helper()

	baseDir := t.TempDir()
	config := create.ProjectConfig{Name: "demo", Domain: "demo.lndo.site"}
	ctx := create.NewContext(baseDir, config, create.ConsoleLogger{}, restoreTestRunner{})

	for _, dir := range []string{ctx.Paths.Root, ctx.Paths.AppDir, ctx.Paths.ConfigDir} {
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatalf("failed to create %s: %v", dir, err)
		}
	}

	return ctx
}
