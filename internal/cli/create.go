package cli

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/gotcha190/toba/internal/create"
	"github.com/gotcha190/toba/internal/create/steps"
	"github.com/gotcha190/toba/internal/wordpress"
)

type CreateOptions struct {
	Name                string
	PHPVersion          string
	StarterRepo         string
	SSHTarget           string
	RemoteWordPressRoot string
	DryRun              bool
}

// RunCreate runs the full ToBA project creation pipeline.
//
// Parameters:
// - opts: parsed options for the `toba create` command
//
// Returns:
// - an error when configuration resolution fails or any pipeline step fails
//
// Side effects:
// - reads global configuration
// - writes logs to stdout
// - executes local commands and writes project files through the pipeline
//
// Usage:
//
//	toba create demo --php=8.4 --starter-repo=git@github.com:org/repo.git --ssh-target='user@host -p 22' --remote-wordpress-root='www/example.com'
func RunCreate(opts CreateOptions) error {
	return runCreateWithIO(opts, create.ExecRunner{}, os.Stdin, os.Stdout)
}

// runCreateWithRunner runs the create flow with an injected command runner for
// tests and non-interactive execution.
//
// Parameters:
// - opts: parsed create options
// - runner: command runner used instead of the default OS-backed runner
//
// Returns:
// - an error when pipeline execution fails
//
// Side effects:
// - uses a default negative answer for cleanup prompts
// - discards user-facing output
func runCreateWithRunner(opts CreateOptions, runner create.CommandRunner) error {
	return runCreateWithIO(opts, runner, strings.NewReader("n\n"), io.Discard)
}

// runCreateWithIO resolves configuration, prepares the runtime context, and
// executes the create pipeline using the provided IO streams.
//
// Parameters:
// - opts: parsed create options coming from the CLI layer
// - runner: command runner used for all shell execution during the run
// - input: source used for cleanup confirmation after failures
// - output: destination used for create logs
//
// Returns:
// - an error when config resolution, context setup, or any pipeline step fails
//
// Side effects:
// - reads the global config file
// - may create, modify, or remove files and directories through pipeline steps
// - may execute local shell commands and remote SSH commands through the runner
// - removes the temporary starter-data directory on exit
func runCreateWithIO(opts CreateOptions, runner create.CommandRunner, input io.Reader, output io.Writer) error {
	logger := create.NewConsoleLogger(output)

	config, envPath, _, err := create.ResolveEnvConfig()
	if err != nil {
		return err
	}
	if envPath != "" {
		logger.Info("Using config from " + envPath)
	}

	if opts.Name != "" {
		config.Name = opts.Name
	}
	if opts.PHPVersion != "" {
		config.PHPVersion = opts.PHPVersion
	}
	if opts.StarterRepo != "" {
		config.StarterRepo = opts.StarterRepo
	}
	if opts.SSHTarget != "" {
		config.SSHTarget = opts.SSHTarget
	}
	if opts.RemoteWordPressRoot != "" {
		config.RemoteWordPressRoot = opts.RemoteWordPressRoot
	}
	config.DryRun = opts.DryRun

	if strings.TrimSpace(config.Name) == "" {
		return fmt.Errorf("project name is required; pass it as argument: toba create <project-name>")
	}

	cwd, err := os.Getwd()
	if err != nil {
		return err
	}

	ctx := create.NewContext(cwd, config, logger, runner)
	defer func() {
		if ctx.StarterData.TempDir == "" {
			return
		}
		if err := os.RemoveAll(ctx.StarterData.TempDir); err != nil {
			ctx.Logger.Warning("Failed to remove starter temp dir: " + ctx.StarterData.TempDir)
		}
	}()

	pipeline := create.Pipeline{
		Nodes: []create.StepNode{
			{ID: "collect-config", Step: steps.NewCollectConfigStep()},
			{ID: "prepare-starter-data", Step: steps.NewPrepareStarterDataStep(), DependsOn: []string{"collect-config"}},
			{ID: "project-dir", Step: steps.NewProjectDirStep(), DependsOn: []string{"prepare-starter-data"}},
			{ID: "generate-lando-config", Step: steps.NewGenerateLandoConfigStep(), DependsOn: []string{"project-dir"}},
			{ID: "start-lando", Step: steps.NewStartLandoStep(), DependsOn: []string{"generate-lando-config"}},
			{ID: "install-theme", Step: steps.NewInstallThemeStep(), DependsOn: []string{"project-dir"}},
			{ID: "import-plugins", Step: steps.NewImportPluginsStep(), DependsOn: []string{"project-dir"}},
			{ID: "import-uploads", Step: steps.NewImportUploadsStep(), DependsOn: []string{"project-dir"}},
			{ID: "import-others", Step: steps.NewImportOthersStep(), DependsOn: []string{"install-theme", "import-plugins", "import-uploads"}},
			{ID: "install-wordpress", Step: steps.NewInstallWordPressStep(), DependsOn: []string{"start-lando"}},
			{ID: "build-theme", Step: steps.NewBuildThemeStep(), DependsOn: []string{"start-lando", "install-theme"}},
			{ID: "import-database", Step: steps.NewImportDatabaseStep(), DependsOn: []string{"install-wordpress"}},
			{ID: "reset-admin-password", Step: steps.NewResetAdminPasswordStep(), DependsOn: []string{"import-database"}},
			{ID: "activate-theme", Step: steps.NewActivateThemeStep(), DependsOn: []string{"import-database", "install-theme", "build-theme"}},
			{ID: "generate-acorn-key", Step: steps.NewGenerateAcornKeyStep(), DependsOn: []string{"activate-theme"}},
			{ID: "clear-imported-caches", Step: steps.NewClearImportedCachesStep(), DependsOn: []string{"import-database"}},
			{ID: "flush-rewrite-rules", Step: steps.NewFlushRewriteRulesStep(), DependsOn: []string{"activate-theme"}},
			{ID: "refresh-theme-caches", Step: steps.NewRefreshThemeCachesStep(), DependsOn: []string{"generate-acorn-key", "clear-imported-caches", "flush-rewrite-rules", "import-others"}},
		},
	}

	err = pipeline.Run(ctx)
	if err != nil {
		cleanupFailedInstall(ctx, input)
		return err
	}

	siteURL, err := wordpress.LocalHTTPSURL(ctx.Config.Domain)
	if err != nil {
		logger.Warning("Project created, but could not determine site URL: " + err.Error())
		return nil
	}

	logger.Success("Project ready: " + siteURL)

	return nil
}
