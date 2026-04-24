package steps

import (
	"archive/zip"
	"bytes"
	"compress/gzip"
	"os"
	"path/filepath"
	"testing"

	"github.com/gotcha190/toba/internal/create"
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
		filepath.Join(ctx.Paths.Plugins, "contact-form-7"),
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
	if _, err := os.Stat(filepath.Join(ctx.Paths.WPContent, "mu-plugins", "b.php")); err != nil {
		t.Fatalf("expected mu-plugins/b.php to exist: %v", err)
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
	if got := runner.commands[1].args; len(got) < 4 || got[2] != "https://starter.example.test" || got[3] != "https://demo.lndo.site" {
		t.Fatalf("unexpected search-replace args: %#v", got)
	}
}

func TestImportDatabaseStepUpdatesConfigForCustomPrefix(t *testing.T) {
	runner := &recordingRunner{}
	ctx := newRestoreTestContext(t)
	ctx.Runner = runner
	ctx.Config.Domain = "demo.lndo.site"
	ctx.StarterData.DatabasePath = writeGzipFixture(t, ctx.Paths.Root, "starter-db-custom.gz", ""+
		"# Backup of: https://starter.example.test\n"+
		"# Home URL: https://starter.example.test\n"+
		"# Table prefix: txxbt_\n"+
		"CREATE TABLE `txxbt_options` (\n"+
		"  `option_id` bigint(20) unsigned NOT NULL AUTO_INCREMENT\n"+
		");\n")

	wpConfigPath := filepath.Join(ctx.Paths.AppDir, "wp-config.php")
	if err := os.WriteFile(wpConfigPath, []byte("<?php\n$table_prefix = 'wp_';\n"), 0644); err != nil {
		t.Fatalf("failed to write wp-config.php: %v", err)
	}

	if err := NewImportDatabaseStep().Run(ctx); err != nil {
		t.Fatalf("ImportDatabaseStep returned error: %v", err)
	}

	updated, err := os.ReadFile(wpConfigPath)
	if err != nil {
		t.Fatalf("failed to read wp-config.php: %v", err)
	}
	if !bytes.Contains(updated, []byte("$table_prefix = 'txxbt_';")) {
		t.Fatalf("expected custom table prefix in wp-config.php, got:\n%s", string(updated))
	}
	if len(runner.commands) != 2 {
		t.Fatalf("expected 2 commands, got %d", len(runner.commands))
	}
}

func TestClearImportedCachesStepRemovesCacheDirAndRunsFlushCommands(t *testing.T) {
	runner := &recordingRunner{}
	ctx := newRestoreTestContext(t)
	ctx.Runner = runner

	cacheFile := filepath.Join(ctx.Paths.WPContent, "cache", "acorn", "framework", "cache", "data", "old.txt")
	if err := os.MkdirAll(filepath.Dir(cacheFile), 0755); err != nil {
		t.Fatalf("failed to create cache dir: %v", err)
	}
	if err := os.WriteFile(cacheFile, []byte("stale"), 0644); err != nil {
		t.Fatalf("failed to write cache file: %v", err)
	}

	if err := NewClearImportedCachesStep().Run(ctx); err != nil {
		t.Fatalf("ClearImportedCachesStep returned error: %v", err)
	}

	if _, err := os.Stat(filepath.Join(ctx.Paths.WPContent, "cache")); !os.IsNotExist(err) {
		t.Fatalf("expected cache dir to be removed, got err=%v", err)
	}
	if len(runner.commands) != 1 {
		t.Fatalf("expected 1 command, got %d", len(runner.commands))
	}
	if got := runner.commands[0].args; len(got) != 3 || got[0] != "wp" || got[1] != "cache" || got[2] != "flush" {
		t.Fatalf("unexpected wp cache flush args: %#v", got)
	}
}

type recordedCommand struct {
	dir  string
	cmd  string
	args []string
}

type recordingRunner struct {
	commands        []recordedCommand
	runErrByCommand map[string]error
}

func (r *recordingRunner) Run(dir string, cmd string, args ...string) error {
	r.commands = append(r.commands, recordedCommand{
		dir:  dir,
		cmd:  cmd,
		args: append([]string(nil), args...),
	})
	if r.runErrByCommand != nil {
		if err, ok := r.runErrByCommand[cmd+" "+joinArgs(args)]; ok {
			return err
		}
	}
	return nil
}

func (r *recordingRunner) CaptureOutput(dir string, cmd string, args ...string) (string, error) {
	return "", nil
}

func joinArgs(args []string) string {
	result := ""
	for i, arg := range args {
		if i > 0 {
			result += " "
		}
		result += arg
	}
	return result
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

	ctx.StarterData = create.StarterData{
		Mode: starterDataModeLocal,
		DatabasePath: writeGzipFixture(t, baseDir, "starter-db.gz", ""+
			"# Backup of: https://starter.example.test\n"+
			"# Home URL: https://starter.example.test\n"+
			"SELECT 1;\n"),
		PluginsPaths: []string{
			writeZipFixture(t, baseDir, "starter-plugins-a.zip", map[string]string{
				"plugins/advanced-custom-fields-pro/acf.php": "<?php\n",
			}),
			writeZipFixture(t, baseDir, "starter-plugins-b.zip", map[string]string{
				"plugins/wp-optimize/wp-optimize.php":          "<?php\n",
				"plugins/contact-form-7/wp-contact-form-7.php": "<?php\n",
			}),
		},
		UploadsPaths: []string{
			writeZipFixture(t, baseDir, "starter-uploads-a.zip", map[string]string{
				"uploads/2025/07/example.jpg": "image",
			}),
			writeZipFixture(t, baseDir, "starter-uploads-b.zip", map[string]string{
				"uploads/2026/readme.txt": "uploaded",
			}),
		},
		OthersPaths: []string{
			writeZipFixture(t, baseDir, "starter-others-a.zip", map[string]string{
				"uploads.zip":       "nested",
				"cache/acorn/.keep": "cache",
			}),
			writeZipFixture(t, baseDir, "starter-others-b.zip", map[string]string{
				"mu-plugins/a.php": "<?php\n",
				"mu-plugins/b.php": "<?php\n",
			}),
		},
	}

	return ctx
}

func writeZipFixture(t *testing.T, dir string, name string, files map[string]string) string {
	t.Helper()

	path := filepath.Join(dir, name)
	output, err := os.Create(path)
	if err != nil {
		t.Fatalf("failed to create %s: %v", path, err)
	}
	defer func() {
		_ = output.Close()
	}()

	writer := zip.NewWriter(output)
	for fileName, content := range files {
		entry, err := writer.Create(fileName)
		if err != nil {
			t.Fatalf("failed to create zip entry %s: %v", fileName, err)
		}
		if _, err := entry.Write([]byte(content)); err != nil {
			t.Fatalf("failed to write zip entry %s: %v", fileName, err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("failed to close zip writer: %v", err)
	}

	return path
}

func writeGzipFixture(t *testing.T, dir string, name string, content string) string {
	t.Helper()

	path := filepath.Join(dir, name)
	var buffer bytes.Buffer
	writer := gzip.NewWriter(&buffer)
	if _, err := writer.Write([]byte(content)); err != nil {
		t.Fatalf("failed to write gzip content: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("failed to close gzip writer: %v", err)
	}

	if err := os.WriteFile(path, buffer.Bytes(), 0644); err != nil {
		t.Fatalf("failed to write %s: %v", path, err)
	}

	return path
}
