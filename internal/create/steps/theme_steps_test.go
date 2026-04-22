package steps

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gotcha190/toba/internal/create"
)

type themeStepRunner struct {
	commands []recordedCommand
	outputs  map[string]string
	runErr   map[string]error
}

func (r *themeStepRunner) Run(dir string, cmd string, args ...string) error {
	r.commands = append(r.commands, recordedCommand{
		dir:  dir,
		cmd:  cmd,
		args: append([]string(nil), args...),
	})
	if r.runErr != nil {
		if err, ok := r.runErr[strings.Join(append([]string{cmd}, args...), " ")]; ok {
			return err
		}
	}
	return nil
}

func (r *themeStepRunner) CaptureOutput(dir string, cmd string, args ...string) (string, error) {
	r.commands = append(r.commands, recordedCommand{
		dir:  dir,
		cmd:  cmd,
		args: append([]string(nil), args...),
	})
	key := strings.Join(append([]string{cmd}, args...), " ")
	return r.outputs[key], nil
}

func TestInstallThemeStepUsesLocalThemesBackup(t *testing.T) {
	ctx := newThemeStepContext(t)
	ctx.StarterData.ThemePaths = []string{
		writeZipFixture(t, ctx.Paths.Root, "starter-themes.zip", map[string]string{
			"themes/sage/style.css": "/* theme */",
		}),
	}

	if err := NewInstallThemeStep().Run(ctx); err != nil {
		t.Fatalf("InstallThemeStep returned error: %v", err)
	}

	if _, err := os.Stat(filepath.Join(ctx.Paths.Themes, "sage", "style.css")); err != nil {
		t.Fatalf("expected restored theme to exist: %v", err)
	}
}

func TestBuildThemeStepSkipsWhenLocalThemeBackupExists(t *testing.T) {
	runner := &themeStepRunner{}
	ctx := newThemeStepContext(t)
	ctx.Runner = runner
	ctx.StarterData.ThemePaths = []string{"starter-themes.zip"}

	if err := NewBuildThemeStep().Run(ctx); err != nil {
		t.Fatalf("BuildThemeStep returned error: %v", err)
	}
	if len(runner.commands) != 0 {
		t.Fatalf("expected no build commands, got %#v", runner.commands)
	}
}

func TestActivateThemeStepUsesDatabaseThemeSlugForLocalThemeBackup(t *testing.T) {
	runner := &themeStepRunner{
		outputs: map[string]string{
			"lando wp eval echo get_option('stylesheet') ?: get_option('template');": "sage\n",
		},
	}
	ctx := newThemeStepContext(t)
	ctx.Runner = runner
	ctx.StarterData.ThemePaths = []string{"starter-themes.zip"}

	if err := NewActivateThemeStep().Run(ctx); err != nil {
		t.Fatalf("ActivateThemeStep returned error: %v", err)
	}

	if len(runner.commands) != 2 {
		t.Fatalf("expected 2 commands, got %#v", runner.commands)
	}
	if got := runner.commands[1].args; len(got) != 4 || got[0] != "wp" || got[1] != "theme" || got[2] != "activate" || got[3] != "sage" {
		t.Fatalf("unexpected activate args: %#v", got)
	}
}

func TestGenerateAcornKeyStepSkipsWhenLocalThemeBackupExists(t *testing.T) {
	runner := &themeStepRunner{}
	ctx := newThemeStepContext(t)
	ctx.Runner = runner
	ctx.StarterData.ThemePaths = []string{"starter-themes.zip"}

	if err := NewGenerateAcornKeyStep().Run(ctx); err != nil {
		t.Fatalf("GenerateAcornKeyStep returned error: %v", err)
	}
	if len(runner.commands) != 0 {
		t.Fatalf("expected no acorn commands, got %#v", runner.commands)
	}
}

func TestRefreshThemeCachesStepRunsExpectedCommands(t *testing.T) {
	runner := &themeStepRunner{
		outputs: map[string]string{
			"lando wp acorn list": "  optimize         Cache the framework bootstrap files\n  cache:clear      Flush the application cache\n  acf:cache        Cache ACF assets\n",
		},
	}
	ctx := newThemeStepContext(t)
	ctx.Runner = runner

	if err := NewRefreshThemeCachesStep().Run(ctx); err != nil {
		t.Fatalf("RefreshThemeCachesStep returned error: %v", err)
	}
	if len(runner.commands) != 2 {
		t.Fatalf("expected 2 cache commands, got %#v", runner.commands)
	}
	if got := runner.commands[0].args; len(got) != 3 || got[0] != "wp" || got[1] != "acorn" || got[2] != "list" {
		t.Fatalf("unexpected list args: %#v", got)
	}
	if got := runner.commands[1].args; len(got) != 5 || got[0] != "ssh" || got[1] != "-s" || got[2] != "appserver" || got[3] != "-c" || got[4] != "cd /app && wp acorn optimize && wp acorn cache:clear && wp acorn acf:cache" {
		t.Fatalf("unexpected batch args: %#v", got)
	}
}

func TestRefreshThemeCachesStepWarnsWhenRefreshFails(t *testing.T) {
	logger := &starterTestLogger{}
	runner := &themeStepRunner{
		outputs: map[string]string{
			"lando wp acorn list": "  optimize:clear   Remove the cached bootstrap files\n",
		},
		runErr: map[string]error{
			"lando ssh -s appserver -c cd /app && wp acorn optimize:clear": errors.New("acorn unavailable"),
		},
	}
	ctx := newThemeStepContext(t)
	ctx.Runner = runner
	ctx.Logger = logger

	if err := NewRefreshThemeCachesStep().Run(ctx); err != nil {
		t.Fatalf("RefreshThemeCachesStep returned error: %v", err)
	}
	if len(logger.warnings) == 0 {
		t.Fatal("expected warning when theme cache refresh fails")
	}
}

func newThemeStepContext(t *testing.T) *create.Context {
	t.Helper()

	baseDir := t.TempDir()
	ctx := create.NewContext(baseDir, create.ProjectConfig{Name: "demo", Domain: "demo.lndo.site", StarterRepo: "git@example.com:company/starter.git"}, create.ConsoleLogger{}, &themeStepRunner{})

	for _, dir := range []string{ctx.Paths.Root, ctx.Paths.AppDir, ctx.Paths.ConfigDir, ctx.Paths.WPContent, ctx.Paths.Themes} {
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatalf("failed to create %s: %v", dir, err)
		}
	}

	return ctx
}
