package steps

import (
	"archive/zip"
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gotcha190/ToBA/internal/create"
)

type starterTestLogger struct {
	warnings []string
}

func (l *starterTestLogger) Step(string)    {}
func (l *starterTestLogger) Info(string)    {}
func (l *starterTestLogger) Success(string) {}
func (l *starterTestLogger) Error(string)   {}
func (l *starterTestLogger) Warning(msg string) {
	l.warnings = append(l.warnings, msg)
}

type starterTestRunner struct {
	commands        []starterRecordedCommand
	captureOutput   string
	cleanupRunError error
}

func (r *starterTestRunner) Run(dir string, cmd string, args ...string) error {
	r.commands = append(r.commands, starterRecordedCommand{dir: dir, cmd: cmd, args: append([]string(nil), args...)})
	if cmd == "scp" && len(args) >= 3 {
		if err := writeStarterFixture(args[len(args)-2], args[len(args)-1]); err != nil {
			return err
		}
	}
	if cmd == "ssh" && len(args) > 0 && strings.Contains(args[len(args)-1], "rm -f") {
		return r.cleanupRunError
	}
	return nil
}

func (r *starterTestRunner) CaptureOutput(dir string, cmd string, args ...string) (string, error) {
	r.commands = append(r.commands, starterRecordedCommand{dir: dir, cmd: cmd, args: append([]string(nil), args...)})
	return r.captureOutput, nil
}

func TestPrepareStarterDataUsesLocalProjectBackupsWhenComplete(t *testing.T) {
	baseDir := t.TempDir()
	ctx := create.NewContext(baseDir, create.ProjectConfig{Name: "demo"}, &starterTestLogger{}, &starterTestRunner{})

	writeStarterProjectFile(t, filepath.Join(ctx.Paths.Root, "backup-demo-db.gz"), "db")
	writeStarterProjectFile(t, filepath.Join(ctx.Paths.Root, "backup-demo-plugins.zip"), "plugins")
	writeStarterProjectFile(t, filepath.Join(ctx.Paths.Root, "backup-demo-uploads.zip"), "uploads")
	writeStarterProjectFile(t, filepath.Join(ctx.Paths.Root, "backup-demo-themes.zip"), "themes")
	writeStarterProjectFile(t, filepath.Join(ctx.Paths.Root, "backup-demo-others.zip"), "others")

	if err := NewPrepareStarterDataStep().Run(ctx); err != nil {
		t.Fatalf("PrepareStarterDataStep returned error: %v", err)
	}

	if ctx.StarterData.Mode != starterDataModeLocal {
		t.Fatalf("unexpected starter mode: %q", ctx.StarterData.Mode)
	}
	if !ctx.UseExistingProjectDir {
		t.Fatal("expected existing project dir mode to be enabled")
	}
	for _, path := range append([]string{ctx.StarterData.DatabasePath}, append(append(append(ctx.StarterData.PluginsPaths, ctx.StarterData.UploadsPaths...), ctx.StarterData.OthersPaths...), ctx.StarterData.ThemePaths...)...) {
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("expected %s to exist: %v", path, err)
		}
		if strings.HasPrefix(path, ctx.Paths.Root) {
			t.Fatalf("expected %s to be copied to a temp dir", path)
		}
	}
}

func TestPrepareStarterDataUsesCategorizedLocalProjectBackups(t *testing.T) {
	baseDir := t.TempDir()
	ctx := create.NewContext(baseDir, create.ProjectConfig{Name: "demo"}, &starterTestLogger{}, &starterTestRunner{})

	writeStarterProjectFile(t, filepath.Join(ctx.Paths.Root, "database", "backup-db.gz"), "db")
	writeStarterProjectFile(t, filepath.Join(ctx.Paths.Root, "plugins", "plugins-a.zip"), "plugins")
	writeStarterProjectFile(t, filepath.Join(ctx.Paths.Root, "uploads", "uploads-a.zip"), "uploads")
	writeStarterProjectFile(t, filepath.Join(ctx.Paths.Root, "themes", "themes-a.zip"), "themes")

	if err := NewPrepareStarterDataStep().Run(ctx); err != nil {
		t.Fatalf("PrepareStarterDataStep returned error: %v", err)
	}

	if ctx.StarterData.Mode != starterDataModeLocal {
		t.Fatalf("unexpected starter mode: %q", ctx.StarterData.Mode)
	}
}

func TestPrepareStarterDataFetchesOverSSHWhenLocalProjectFolderMissing(t *testing.T) {
	logger := &starterTestLogger{}
	runner := &starterTestRunner{captureOutput: "https://starter.tamago-dev.pl\n"}
	ctx := create.NewContext(t.TempDir(), create.ProjectConfig{
		Name:      "demo",
		SSHTarget: "user@192.168.0.1 -p 22",
	}, logger, runner)

	if err := NewPrepareStarterDataStep().Run(ctx); err != nil {
		t.Fatalf("PrepareStarterDataStep returned error: %v", err)
	}

	if ctx.StarterData.Mode != starterDataModeRemote {
		t.Fatalf("unexpected starter mode: %q", ctx.StarterData.Mode)
	}
	if ctx.StarterData.SourceURL != "https://starter.tamago-dev.pl" {
		t.Fatalf("unexpected source URL: %q", ctx.StarterData.SourceURL)
	}
	for _, path := range append([]string{ctx.StarterData.DatabasePath}, append(ctx.StarterData.PluginsPaths, ctx.StarterData.UploadsPaths...)...) {
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("expected %s to exist: %v", path, err)
		}
	}
}

func TestPrepareStarterDataRejectsEmptyExistingProjectFolder(t *testing.T) {
	baseDir := t.TempDir()
	ctx := create.NewContext(baseDir, create.ProjectConfig{Name: "demo"}, &starterTestLogger{}, &starterTestRunner{})

	if err := os.MkdirAll(ctx.Paths.Root, 0755); err != nil {
		t.Fatalf("failed to create root: %v", err)
	}

	err := NewPrepareStarterDataStep().Run(ctx)
	if err == nil {
		t.Fatal("expected empty existing project folder to fail")
	}
	if !strings.Contains(err.Error(), "contains no recognizable Updraft backup files") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestPrepareStarterDataRejectsPartialLocalProjectBackup(t *testing.T) {
	baseDir := t.TempDir()
	ctx := create.NewContext(baseDir, create.ProjectConfig{Name: "demo"}, &starterTestLogger{}, &starterTestRunner{})

	writeStarterProjectFile(t, filepath.Join(ctx.Paths.Root, "backup-demo-db.gz"), "db")
	writeStarterProjectFile(t, filepath.Join(ctx.Paths.Root, "backup-demo-plugins.zip"), "plugins")
	writeStarterProjectFile(t, filepath.Join(ctx.Paths.Root, "backup-demo-uploads.zip"), "uploads")

	err := NewPrepareStarterDataStep().Run(ctx)
	if err == nil {
		t.Fatal("expected partial local backup error")
	}
	if !strings.Contains(err.Error(), "incomplete") || !strings.Contains(err.Error(), "themes") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestPrepareStarterDataRejectsMultipleLocalDatabases(t *testing.T) {
	baseDir := t.TempDir()
	ctx := create.NewContext(baseDir, create.ProjectConfig{Name: "demo"}, &starterTestLogger{}, &starterTestRunner{})

	writeStarterProjectFile(t, filepath.Join(ctx.Paths.Root, "backup-demo-db.gz"), "db")
	writeStarterProjectFile(t, filepath.Join(ctx.Paths.Root, "database", "backup-db.sql"), "db2")

	err := NewPrepareStarterDataStep().Run(ctx)
	if err == nil {
		t.Fatal("expected multiple database error")
	}
	if !strings.Contains(err.Error(), "expected exactly 1 database backup") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestPrepareStarterDataRejectsInvalidSSHTarget(t *testing.T) {
	ctx := create.NewContext(t.TempDir(), create.ProjectConfig{Name: "demo", SSHTarget: "bad-target"}, &starterTestLogger{}, &starterTestRunner{})
	err := NewPrepareStarterDataStep().Run(ctx)
	if err == nil {
		t.Fatal("expected invalid ssh target error")
	}
	if !strings.Contains(err.Error(), "expected format: user@host -p port") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestPrepareStarterDataWarnsWhenRemoteCleanupFails(t *testing.T) {
	logger := &starterTestLogger{}
	ctx := create.NewContext(t.TempDir(), create.ProjectConfig{
		Name:      "demo",
		SSHTarget: "user@192.168.0.1 -p 22",
	}, logger, &starterTestRunner{
		captureOutput:   "https://starter.tamago-dev.pl\n",
		cleanupRunError: os.ErrPermission,
	})

	if err := NewPrepareStarterDataStep().Run(ctx); err != nil {
		t.Fatalf("PrepareStarterDataStep returned error: %v", err)
	}
	if len(logger.warnings) == 0 {
		t.Fatal("expected cleanup warning")
	}
}

func writeStarterProjectFile(t *testing.T, path string, content string) {
	t.Helper()

	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatalf("failed to create parent dir for %s: %v", path, err)
	}
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write %s: %v", path, err)
	}
}

func writeStarterFixture(remoteSource string, localTarget string) error {
	if err := os.MkdirAll(filepath.Dir(localTarget), 0755); err != nil {
		return err
	}

	switch {
	case strings.HasSuffix(remoteSource, ".sql"):
		return os.WriteFile(localTarget, []byte("# Home URL: https://starter.tamago-dev.pl\nSELECT 1;\n"), 0644)
	case strings.HasSuffix(remoteSource, "-plugins.zip"):
		return os.WriteFile(localTarget, zippedBytes(nil, map[string]string{
			"plugins/example/plugin.php": "<?php\n",
		}), 0644)
	case strings.HasSuffix(remoteSource, "-uploads.zip"):
		return os.WriteFile(localTarget, zippedBytes(nil, map[string]string{
			"uploads/2026/file.txt": "uploaded",
		}), 0644)
	default:
		return nil
	}
}

func zippedBytes(t *testing.T, files map[string]string) []byte {
	if t != nil {
		t.Helper()
	}

	var buffer bytes.Buffer
	writer := zip.NewWriter(&buffer)
	for name, content := range files {
		entry, err := writer.Create(name)
		if err != nil {
			if t != nil {
				t.Fatalf("failed to create zip entry %s: %v", name, err)
			}
			panic(err)
		}
		if _, err := entry.Write([]byte(content)); err != nil {
			if t != nil {
				t.Fatalf("failed to write zip entry %s: %v", name, err)
			}
			panic(err)
		}
	}
	if err := writer.Close(); err != nil {
		if t != nil {
			t.Fatalf("failed to close zip writer: %v", err)
		}
		panic(err)
	}

	return buffer.Bytes()
}

type starterRecordedCommand struct {
	dir  string
	cmd  string
	args []string
}
