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

func TestPrepareStarterDataUsesEmbeddedOverrideWhenComplete(t *testing.T) {
	restoreTemplateStubs(t,
		map[string][]string{
			"database|.sql": {"wordpress/database/starter.sql"},
			"database|.gz":  nil,
			"plugins|.zip":  {"wordpress/plugins/starter-plugins.zip"},
			"uploads|.zip":  {"wordpress/uploads/starter-uploads.zip"},
			"others|.zip":   {"wordpress/others/starter-others.zip"},
		},
		map[string][]byte{
			"wordpress/database/starter.sql": []byte("# Home URL: https://starter.tamago-dev.pl\nSELECT 1;\n"),
			"wordpress/plugins/starter-plugins.zip": zippedBytes(t, map[string]string{
				"plugins/a/a.php": "<?php\n",
			}),
			"wordpress/uploads/starter-uploads.zip": zippedBytes(t, map[string]string{
				"uploads/2026/file.txt": "ok",
			}),
			"wordpress/others/starter-others.zip": zippedBytes(t, map[string]string{
				"cache/acorn/.keep": "ok",
			}),
		},
	)

	logger := &starterTestLogger{}
	ctx := create.NewContext(t.TempDir(), create.ProjectConfig{Name: "demo"}, logger, &starterTestRunner{})

	if err := NewPrepareStarterDataStep().Run(ctx); err != nil {
		t.Fatalf("PrepareStarterDataStep returned error: %v", err)
	}

	if ctx.StarterData.Mode != starterDataModeEmbedded {
		t.Fatalf("unexpected starter mode: %q", ctx.StarterData.Mode)
	}
	for _, path := range append([]string{
		ctx.StarterData.DatabasePath,
	}, append(append(ctx.StarterData.PluginsPaths, ctx.StarterData.UploadsPaths...), ctx.StarterData.OthersPaths...)...) {
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("expected %s to exist: %v", path, err)
		}
	}
}

func TestPrepareStarterDataFetchesOverSSHWhenEmbeddedOverrideMissing(t *testing.T) {
	restoreTemplateStubs(t,
		map[string][]string{
			"database|.sql": nil,
			"database|.gz":  nil,
			"plugins|.zip":  nil,
			"uploads|.zip":  nil,
			"others|.zip":   nil,
		},
		nil,
	)

	logger := &starterTestLogger{}
	runner := &starterTestRunner{captureOutput: "https://starter.tamago-dev.pl\n"}
	ctx := create.NewContext(t.TempDir(), create.ProjectConfig{
		Name:      "demo",
		SSHTarget: "toba@185.238.75.243 -p 22666",
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

func TestPrepareStarterDataAllowsMultiplePluginAndUploadOverrides(t *testing.T) {
	restoreTemplateStubs(t,
		map[string][]string{
			"database|.sql": {"wordpress/database/starter.sql"},
			"database|.gz":  nil,
			"plugins|.zip": {
				"wordpress/plugins/plugins-part-1.zip",
				"wordpress/plugins/plugins-part-2.zip",
			},
			"uploads|.zip": {
				"wordpress/uploads/uploads-part-1.zip",
				"wordpress/uploads/uploads-part-2.zip",
			},
			"others|.zip": nil,
		},
		map[string][]byte{
			"wordpress/database/starter.sql": []byte("# Home URL: https://starter.tamago-dev.pl\nSELECT 1;\n"),
			"wordpress/plugins/plugins-part-1.zip": zippedBytes(t, map[string]string{
				"plugins/a/a.php": "<?php\n",
			}),
			"wordpress/plugins/plugins-part-2.zip": zippedBytes(t, map[string]string{
				"plugins/b/b.php": "<?php\n",
			}),
			"wordpress/uploads/uploads-part-1.zip": zippedBytes(t, map[string]string{
				"uploads/2026/a.txt": "a",
			}),
			"wordpress/uploads/uploads-part-2.zip": zippedBytes(t, map[string]string{
				"uploads/2026/b.txt": "b",
			}),
		},
	)

	ctx := create.NewContext(t.TempDir(), create.ProjectConfig{Name: "demo"}, &starterTestLogger{}, &starterTestRunner{})

	if err := NewPrepareStarterDataStep().Run(ctx); err != nil {
		t.Fatalf("PrepareStarterDataStep returned error: %v", err)
	}

	if len(ctx.StarterData.PluginsPaths) != 2 {
		t.Fatalf("expected 2 plugin archives, got %d", len(ctx.StarterData.PluginsPaths))
	}
	if len(ctx.StarterData.UploadsPaths) != 2 {
		t.Fatalf("expected 2 upload archives, got %d", len(ctx.StarterData.UploadsPaths))
	}
}

func TestPrepareStarterDataAllowsMultipleOthersOverrides(t *testing.T) {
	restoreTemplateStubs(t,
		map[string][]string{
			"database|.sql": {"wordpress/database/starter.sql"},
			"database|.gz":  nil,
			"plugins|.zip":  {"wordpress/plugins/starter-plugins.zip"},
			"uploads|.zip":  {"wordpress/uploads/starter-uploads.zip"},
			"others|.zip": {
				"wordpress/others/others-part-1.zip",
				"wordpress/others/others-part-2.zip",
			},
		},
		map[string][]byte{
			"wordpress/database/starter.sql": []byte("# Home URL: https://starter.tamago-dev.pl\nSELECT 1;\n"),
			"wordpress/plugins/starter-plugins.zip": zippedBytes(t, map[string]string{
				"plugins/a/a.php": "<?php\n",
			}),
			"wordpress/uploads/starter-uploads.zip": zippedBytes(t, map[string]string{
				"uploads/2026/file.txt": "ok",
			}),
			"wordpress/others/others-part-1.zip": zippedBytes(t, map[string]string{
				"cache/acorn/.keep": "ok",
			}),
			"wordpress/others/others-part-2.zip": zippedBytes(t, map[string]string{
				"mu-plugins/a.php": "<?php\n",
			}),
		},
	)

	ctx := create.NewContext(t.TempDir(), create.ProjectConfig{Name: "demo"}, &starterTestLogger{}, &starterTestRunner{})

	if err := NewPrepareStarterDataStep().Run(ctx); err != nil {
		t.Fatalf("PrepareStarterDataStep returned error: %v", err)
	}

	if len(ctx.StarterData.OthersPaths) != 2 {
		t.Fatalf("expected 2 others archives, got %d", len(ctx.StarterData.OthersPaths))
	}
}

func TestPrepareStarterDataAllowsThemeOverrides(t *testing.T) {
	restoreTemplateStubs(t,
		map[string][]string{
			"database|.sql": {"wordpress/database/starter.sql"},
			"database|.gz":  nil,
			"plugins|.zip":  {"wordpress/plugins/starter-plugins.zip"},
			"uploads|.zip":  {"wordpress/uploads/starter-uploads.zip"},
			"others|.zip":   nil,
			"themes|.zip": {
				"wordpress/themes/starter-themes.zip",
				"wordpress/themes/starter-themes2.zip",
			},
		},
		map[string][]byte{
			"wordpress/database/starter.sql": []byte("# Home URL: https://starter.tamago-dev.pl\nSELECT 1;\n"),
			"wordpress/plugins/starter-plugins.zip": zippedBytes(t, map[string]string{
				"plugins/a/a.php": "<?php\n",
			}),
			"wordpress/uploads/starter-uploads.zip": zippedBytes(t, map[string]string{
				"uploads/2026/file.txt": "ok",
			}),
			"wordpress/themes/starter-themes.zip": zippedBytes(t, map[string]string{
				"themes/sage/style.css": "/* theme */",
			}),
			"wordpress/themes/starter-themes2.zip": zippedBytes(t, map[string]string{
				"themes/twentytwenty/style.css": "/* theme */",
			}),
		},
	)

	ctx := create.NewContext(t.TempDir(), create.ProjectConfig{Name: "demo"}, &starterTestLogger{}, &starterTestRunner{})

	if err := NewPrepareStarterDataStep().Run(ctx); err != nil {
		t.Fatalf("PrepareStarterDataStep returned error: %v", err)
	}

	if len(ctx.StarterData.ThemePaths) != 2 {
		t.Fatalf("expected 2 theme archives, got %d", len(ctx.StarterData.ThemePaths))
	}
}

func TestPrepareStarterDataRejectsPartialEmbeddedOverride(t *testing.T) {
	restoreTemplateStubs(t,
		map[string][]string{
			"database|.sql": {"wordpress/database/starter.sql"},
			"database|.gz":  nil,
			"plugins|.zip":  {"wordpress/plugins/starter-plugins.zip"},
			"uploads|.zip":  nil,
			"others|.zip":   nil,
		},
		nil,
	)

	ctx := create.NewContext(t.TempDir(), create.ProjectConfig{Name: "demo"}, &starterTestLogger{}, &starterTestRunner{})
	err := NewPrepareStarterDataStep().Run(ctx)
	if err == nil {
		t.Fatal("expected partial override error")
	}
	if !strings.Contains(err.Error(), "incomplete embedded starter override") || !strings.Contains(err.Error(), "uploads") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestPrepareStarterDataRejectsInvalidSSHTarget(t *testing.T) {
	restoreTemplateStubs(t,
		map[string][]string{
			"database|.sql": nil,
			"database|.gz":  nil,
			"plugins|.zip":  nil,
			"uploads|.zip":  nil,
			"others|.zip":   nil,
		},
		nil,
	)

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
	restoreTemplateStubs(t,
		map[string][]string{
			"database|.sql": nil,
			"database|.gz":  nil,
			"plugins|.zip":  nil,
			"uploads|.zip":  nil,
			"others|.zip":   nil,
		},
		nil,
	)

	logger := &starterTestLogger{}
	ctx := create.NewContext(t.TempDir(), create.ProjectConfig{
		Name:      "demo",
		SSHTarget: "toba@185.238.75.243 -p 22666",
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

func restoreTemplateStubs(t *testing.T, listing map[string][]string, contents map[string][]byte) {
	t.Helper()

	originalList := starterTemplateFiles
	originalRead := starterTemplateReader
	starterTemplateFiles = func(category string, suffix string) ([]string, error) {
		return append([]string(nil), listing[category+"|"+suffix]...), nil
	}
	starterTemplateReader = func(path string) ([]byte, error) {
		content, ok := contents[path]
		if !ok {
			return nil, os.ErrNotExist
		}
		return append([]byte(nil), content...), nil
	}

	t.Cleanup(func() {
		starterTemplateFiles = originalList
		starterTemplateReader = originalRead
	})
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
