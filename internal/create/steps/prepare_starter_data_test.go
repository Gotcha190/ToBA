package steps

import (
	"archive/zip"
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gotcha190/toba/internal/create"
)

type starterTestLogger struct {
	infos    []string
	warnings []string
}

func (l *starterTestLogger) Step(string)              {}
func (l *starterTestLogger) Info(msg string)          { l.infos = append(l.infos, msg) }
func (l *starterTestLogger) Prompt(string)            {}
func (l *starterTestLogger) Success(string)           {}
func (l *starterTestLogger) Error(string)             {}
func (l *starterTestLogger) ErrorCode(string, string) {}
func (l *starterTestLogger) Warning(msg string) {
	l.warnings = append(l.warnings, msg)
}

type starterTestRunner struct {
	commands        []starterRecordedCommand
	captureOutput   string
	captureErr      error
	runErrByCommand map[string]error
	runErrContains  string
	runErr          error
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
	if r.runErrByCommand != nil {
		if err, ok := r.runErrByCommand[cmd+" "+strings.Join(args, " ")]; ok {
			return err
		}
	}
	if r.runErrContains != "" && strings.Contains(cmd+" "+strings.Join(args, " "), r.runErrContains) {
		return r.runErr
	}
	return nil
}

func (r *starterTestRunner) CaptureOutput(dir string, cmd string, args ...string) (string, error) {
	r.commands = append(r.commands, starterRecordedCommand{dir: dir, cmd: cmd, args: append([]string(nil), args...)})
	if r.captureErr != nil {
		return "", r.captureErr
	}
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

func TestPrepareStarterDataIgnoresCategorizedLocalProjectBackups(t *testing.T) {
	baseDir := t.TempDir()
	ctx := create.NewContext(baseDir, create.ProjectConfig{Name: "demo"}, &starterTestLogger{}, &starterTestRunner{})

	writeStarterProjectFile(t, filepath.Join(ctx.Paths.Root, "database", "backup-db.gz"), "db")
	writeStarterProjectFile(t, filepath.Join(ctx.Paths.Root, "plugins", "plugins-a.zip"), "plugins")
	writeStarterProjectFile(t, filepath.Join(ctx.Paths.Root, "uploads", "uploads-a.zip"), "uploads")
	writeStarterProjectFile(t, filepath.Join(ctx.Paths.Root, "themes", "themes-a.zip"), "themes")

	err := NewPrepareStarterDataStep().Run(ctx)
	if err == nil {
		t.Fatal("expected categorized backups to be ignored")
	}
	if !strings.Contains(err.Error(), "contains no recognizable Updraft backup files") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestPrepareStarterDataFetchesOverSSHWhenLocalProjectFolderMissing(t *testing.T) {
	logger := &starterTestLogger{}
	runner := &starterTestRunner{captureOutput: "https://starter.example.test\n"}
	ctx := create.NewContext(t.TempDir(), create.ProjectConfig{
		Name:                "demo",
		SSHTarget:           "user@192.168.0.1 -p 22",
		RemoteWordPressRoot: "www/example.com",
	}, logger, runner)

	if err := NewPrepareStarterDataStep().Run(ctx); err != nil {
		t.Fatalf("PrepareStarterDataStep returned error: %v", err)
	}

	if ctx.StarterData.Mode != starterDataModeRemote {
		t.Fatalf("unexpected starter mode: %q", ctx.StarterData.Mode)
	}
	if ctx.StarterData.SourceURL != "https://starter.example.test" {
		t.Fatalf("unexpected source URL: %q", ctx.StarterData.SourceURL)
	}
	for _, path := range append([]string{ctx.StarterData.DatabasePath}, append(ctx.StarterData.PluginsPaths, ctx.StarterData.UploadsPaths...)...) {
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("expected %s to exist: %v", path, err)
		}
	}
	for _, expected := range []string{
		"No local project backup folder found; using SSH starter data",
		"Preparing starter files on SSH host user@192.168.0.1 -p 22",
		"Downloading starter database over SSH",
		"Downloading starter plugins over SSH",
		"Downloading starter uploads over SSH",
	} {
		if !containsString(logger.infos, expected) {
			t.Fatalf("expected info log %q, got %#v", expected, logger.infos)
		}
	}
	if len(logger.infos) < 2 {
		t.Fatalf("expected multiple info logs, got %#v", logger.infos)
	}
	if logger.infos[0] != "No local project backup folder found; using SSH starter data" {
		t.Fatalf("expected SSH source info first, got %#v", logger.infos)
	}
	if logger.infos[1] != "Preparing starter files on SSH host user@192.168.0.1 -p 22" {
		t.Fatalf("expected SSH preparation info second, got %#v", logger.infos)
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
	writeStarterProjectFile(t, filepath.Join(ctx.Paths.Root, "backup-demo-db.sql"), "db2")

	err := NewPrepareStarterDataStep().Run(ctx)
	if err == nil {
		t.Fatal("expected multiple database error")
	}
	if !strings.Contains(err.Error(), "expected exactly 1 database backup") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestPrepareStarterDataRejectsInvalidSSHTarget(t *testing.T) {
	ctx := create.NewContext(t.TempDir(), create.ProjectConfig{Name: "demo", SSHTarget: "bad-target", RemoteWordPressRoot: "www/example.com"}, &starterTestLogger{}, &starterTestRunner{})
	err := NewPrepareStarterDataStep().Run(ctx)
	if err == nil {
		t.Fatal("expected invalid ssh target error")
	}
	if !strings.Contains(err.Error(), "expected format: user@host -p port") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestPrepareStarterDataRejectsMissingRemoteWordPressRoot(t *testing.T) {
	ctx := create.NewContext(t.TempDir(), create.ProjectConfig{
		Name:      "demo",
		SSHTarget: "user@192.168.0.1 -p 22",
	}, &starterTestLogger{}, &starterTestRunner{})

	err := NewPrepareStarterDataStep().Run(ctx)
	if err == nil {
		t.Fatal("expected missing remote WordPress root error")
	}
	if !strings.Contains(err.Error(), "TOBA_REMOTE_WORDPRESS_ROOT") || !strings.Contains(err.Error(), "--remote-wordpress-root") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestPrepareStarterDataWarnsWhenRemoteCleanupFails(t *testing.T) {
	logger := &starterTestLogger{}
	ctx := create.NewContext(t.TempDir(), create.ProjectConfig{
		Name:                "demo",
		SSHTarget:           "user@192.168.0.1 -p 22",
		RemoteWordPressRoot: "www/example.com",
	}, logger, &starterTestRunner{
		captureOutput:   "https://starter.example.test\n",
		cleanupRunError: os.ErrPermission,
	})

	if err := NewPrepareStarterDataStep().Run(ctx); err != nil {
		t.Fatalf("PrepareStarterDataStep returned error: %v", err)
	}
	if len(logger.warnings) == 0 {
		t.Fatal("expected cleanup warning")
	}
}

func TestPrepareStarterDataWrapsSSHConnectionErrors(t *testing.T) {
	ctx := create.NewContext(t.TempDir(), create.ProjectConfig{
		Name:                "demo",
		SSHTarget:           "user@192.168.0.1 -p 22",
		RemoteWordPressRoot: "www/example.com",
	}, &starterTestLogger{}, &starterTestRunner{
		captureErr: os.ErrDeadlineExceeded,
	})

	err := NewPrepareStarterDataStep().Run(ctx)
	if err == nil {
		t.Fatal("expected SSH connection error")
	}
	if !strings.Contains(err.Error(), "failed to connect to SSH starter host user@192.168.0.1:22") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestPrepareStarterDataCleansRemoteArtifactsWhenRemoteZipFails(t *testing.T) {
	runner := &starterTestRunner{
		captureOutput:  "https://starter.example.test\n",
		runErrContains: "zip -rq ../",
		runErr:         os.ErrInvalid,
	}
	ctx := create.NewContext(t.TempDir(), create.ProjectConfig{
		Name:                "demo",
		SSHTarget:           "user@192.168.0.1 -p 22",
		RemoteWordPressRoot: "www/example.com",
	}, &starterTestLogger{}, runner)

	err := NewPrepareStarterDataStep().Run(ctx)
	if err == nil {
		t.Fatal("expected zip failure")
	}

	foundCleanup := false
	for _, command := range runner.commands {
		if command.cmd != "ssh" || len(command.args) == 0 {
			continue
		}
		if strings.Contains(command.args[len(command.args)-1], "rm -f") {
			foundCleanup = true
			break
		}
	}
	if !foundCleanup {
		t.Fatalf("expected cleanup command after remote zip failure, got %#v", runner.commands)
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
		return os.WriteFile(localTarget, []byte("# Home URL: https://starter.example.test\nSELECT 1;\n"), 0644)
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

func containsString(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}

	return false
}

type starterRecordedCommand struct {
	dir  string
	cmd  string
	args []string
}
