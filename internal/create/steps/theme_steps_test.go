package steps

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/gotcha190/toba/internal/create"
	"github.com/gotcha190/toba/internal/testutil"
)

func TestCloneStarterThemeStepUsesLocalThemesBackup(t *testing.T) {
	ctx := newThemeStepContext(t)
	ctx.StarterData.ThemePaths = []string{
		writeZipFixture(t, ctx.Paths.Root, "starter-themes.zip", map[string]string{
			"themes/sage/style.css": "/* theme */",
		}),
	}

	if err := NewCloneStarterThemeStep().Run(ctx); err != nil {
		t.Fatalf("CloneStarterThemeStep returned error: %v", err)
	}

	if _, err := os.Stat(filepath.Join(ctx.Paths.Themes, "sage", "style.css")); err != nil {
		t.Fatalf("expected restored theme to exist: %v", err)
	}
}

func TestCloneStarterThemeStepDryRunLogsClone(t *testing.T) {
	logger := &starterTestLogger{}
	runner := &testutil.RecordingRunner{}
	ctx := newThemeStepContext(t)
	ctx.DryRun = true
	ctx.Logger = logger
	ctx.Runner = runner

	if err := NewCloneStarterThemeStep().Run(ctx); err != nil {
		t.Fatalf("CloneStarterThemeStep returned error: %v", err)
	}
	if len(runner.Commands) != 0 {
		t.Fatalf("expected no dry-run commands, got %#v", runner.Commands)
	}

	expected := []string{
		"Would run: git clone git@example.com:company/starter.git demo",
	}
	for _, message := range expected {
		if !containsString(logger.infos, message) {
			t.Fatalf("expected dry-run log %q in %#v", message, logger.infos)
		}
	}
}

func TestSetupThemeGitStepDryRunLogsGitSetup(t *testing.T) {
	logger := &starterTestLogger{}
	runner := &testutil.RecordingRunner{}
	ctx := newThemeStepContext(t)
	ctx.DryRun = true
	ctx.Logger = logger
	ctx.Runner = runner

	if err := NewSetupThemeGitStep().Run(ctx); err != nil {
		t.Fatalf("SetupThemeGitStep returned error: %v", err)
	}
	if len(runner.Commands) != 0 {
		t.Fatalf("expected no dry-run commands, got %#v", runner.Commands)
	}

	expected := []string{
		"Would run in " + filepath.Join(ctx.Paths.Themes, "demo") + ": git remote remove origin",
		"Would run in " + filepath.Join(ctx.Paths.Themes, "demo") + ": git branch -M develop",
		"Would run in " + filepath.Join(ctx.Paths.Themes, "demo") + ": git branch -f starter develop",
		"Would check project repo derived from starter repo: git@example.com:company/demo.git",
		"Would add origin and push develop/starter if project repo exists",
	}
	for _, message := range expected {
		if !containsString(logger.infos, message) {
			t.Fatalf("expected dry-run log %q in %#v", message, logger.infos)
		}
	}
}

func TestSetupThemeGitStepDryRunWarnsForInvalidStarterRepo(t *testing.T) {
	logger := &starterTestLogger{}
	runner := &testutil.RecordingRunner{}
	ctx := newThemeStepContext(t)
	ctx.DryRun = true
	ctx.Logger = logger
	ctx.Runner = runner
	ctx.Config.StarterRepo = "not-a-repo"

	if err := NewSetupThemeGitStep().Run(ctx); err != nil {
		t.Fatalf("SetupThemeGitStep returned error: %v", err)
	}
	if len(runner.Commands) != 0 {
		t.Fatalf("expected no dry-run commands, got %#v", runner.Commands)
	}
	if !containsString(logger.warnings, "Could not derive project git repo from starter repo; skipping branch push") {
		t.Fatalf("expected invalid starter repo warning in %#v", logger.warnings)
	}
}

func TestSetupThemeGitStepWarnsAndContinuesWhenGitCommandsFail(t *testing.T) {
	logger := &starterTestLogger{}
	runner := &testutil.RecordingRunner{
		RunErrByCommand: map[string]error{
			"git remote remove origin":                       errors.New("remove failed"),
			"git branch -M develop":                          errors.New("branch failed"),
			"git ls-remote git@example.com:company/demo.git": errors.New("repo missing"),
		},
	}
	ctx := newThemeStepContext(t)
	ctx.Logger = logger
	ctx.Runner = runner

	if err := NewSetupThemeGitStep().Run(ctx); err != nil {
		t.Fatalf("SetupThemeGitStep returned error: %v", err)
	}
	if len(logger.warnings) == 0 {
		t.Fatal("expected warnings when git setup fails")
	}
	if !themeStepHasCommand(runner.Commands, "git", []string{"branch", "-f", "starter", "develop"}) {
		t.Fatalf("expected setup to continue through starter branch command, got %#v", runner.Commands)
	}
}

func TestSetupThemeGitStepSkipsLocalThemesBackup(t *testing.T) {
	runner := &testutil.RecordingRunner{}
	ctx := newThemeStepContext(t)
	ctx.Runner = runner
	ctx.StarterData.ThemePaths = []string{"starter-themes.zip"}

	if err := NewSetupThemeGitStep().Run(ctx); err != nil {
		t.Fatalf("SetupThemeGitStep returned error: %v", err)
	}
	if len(runner.Commands) != 0 {
		t.Fatalf("expected no git setup commands, got %#v", runner.Commands)
	}
}

func TestBuildThemeStepSkipsWhenLocalThemeBackupExists(t *testing.T) {
	runner := &testutil.RecordingRunner{}
	ctx := newThemeStepContext(t)
	ctx.Runner = runner
	ctx.StarterData.ThemePaths = []string{"starter-themes.zip"}

	if err := NewBuildThemeStep().Run(ctx); err != nil {
		t.Fatalf("BuildThemeStep returned error: %v", err)
	}
	if len(runner.Commands) != 0 {
		t.Fatalf("expected no build commands, got %#v", runner.Commands)
	}
}

func TestActivateThemeStepUsesDatabaseThemeSlugForLocalThemeBackup(t *testing.T) {
	runner := &testutil.RecordingRunner{
		Outputs: map[string]string{
			"lando wp eval echo get_option('stylesheet') ?: get_option('template');": "sage\n",
		},
	}
	ctx := newThemeStepContext(t)
	ctx.Runner = runner
	ctx.StarterData.ThemePaths = []string{"starter-themes.zip"}

	if err := NewActivateThemeStep().Run(ctx); err != nil {
		t.Fatalf("ActivateThemeStep returned error: %v", err)
	}

	if len(runner.Commands) != 2 {
		t.Fatalf("expected 2 commands, got %#v", runner.Commands)
	}
	if got := runner.Commands[1].Args; len(got) != 4 || got[0] != "wp" || got[1] != "theme" || got[2] != "activate" || got[3] != "sage" {
		t.Fatalf("unexpected activate args: %#v", got)
	}
}

func TestGenerateAcornKeyStepSkipsWhenLocalThemeBackupExists(t *testing.T) {
	runner := &testutil.RecordingRunner{}
	ctx := newThemeStepContext(t)
	ctx.Runner = runner
	ctx.StarterData.ThemePaths = []string{"starter-themes.zip"}

	if err := NewGenerateAcornKeyStep().Run(ctx); err != nil {
		t.Fatalf("GenerateAcornKeyStep returned error: %v", err)
	}
	if len(runner.Commands) != 0 {
		t.Fatalf("expected no acorn commands, got %#v", runner.Commands)
	}
}

func TestRefreshThemeCachesStepRunsExpectedCommands(t *testing.T) {
	runner := &testutil.RecordingRunner{
		Outputs: map[string]string{
			"lando wp acorn list": "  optimize         Cache the framework bootstrap files\n  cache:clear      Flush the application cache\n  acf:cache        Cache ACF assets\n",
		},
	}
	ctx := newThemeStepContext(t)
	ctx.Runner = runner

	if err := NewRefreshThemeCachesStep().Run(ctx); err != nil {
		t.Fatalf("RefreshThemeCachesStep returned error: %v", err)
	}
	if len(runner.Commands) != 2 {
		t.Fatalf("expected 2 cache commands, got %#v", runner.Commands)
	}
	if got := runner.Commands[0].Args; len(got) != 3 || got[0] != "wp" || got[1] != "acorn" || got[2] != "list" {
		t.Fatalf("unexpected list args: %#v", got)
	}
	if got := runner.Commands[1].Args; len(got) != 5 || got[0] != "ssh" || got[1] != "-s" || got[2] != "appserver" || got[3] != "-c" || got[4] != "cd /app && wp acorn optimize && wp acorn cache:clear && wp acorn acf:cache" {
		t.Fatalf("unexpected batch args: %#v", got)
	}
}

func TestRefreshThemeCachesStepWarnsWhenRefreshFails(t *testing.T) {
	logger := &starterTestLogger{}
	runner := &testutil.RecordingRunner{
		Outputs: map[string]string{
			"lando wp acorn list": "  optimize:clear   Remove the cached bootstrap files\n",
		},
		RunErrByCommand: map[string]error{
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
	ctx := create.NewContext(baseDir, create.ProjectConfig{Name: "demo", Domain: "demo.lndo.site", StarterRepo: "git@example.com:company/starter.git"}, create.ConsoleLogger{}, &testutil.RecordingRunner{})

	for _, dir := range []string{ctx.Paths.Root, ctx.Paths.AppDir, ctx.Paths.ConfigDir, ctx.Paths.WPContent, ctx.Paths.Themes} {
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatalf("failed to create %s: %v", dir, err)
		}
	}

	return ctx
}

func themeStepHasCommand(commands []testutil.RecordedCommand, cmd string, args []string) bool {
	for _, command := range commands {
		if command.Cmd != cmd {
			continue
		}
		if len(command.Args) != len(args) {
			continue
		}
		matches := true
		for i := range args {
			if command.Args[i] != args[i] {
				matches = false
				break
			}
		}
		if matches {
			return true
		}
	}

	return false
}
