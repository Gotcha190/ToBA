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
		Steps: []create.Step{
			steps.NewCollectConfigStep(),
			steps.NewPrepareStarterDataStep(),
			steps.NewProjectDirStep(),
			steps.NewGenerateLandoConfigStep(),
			steps.NewStartLandoStep(),
			steps.NewInstallWordPressStep(),
			steps.NewInstallThemeStep(),
			steps.NewBuildThemeStep(),
			steps.NewImportPluginsStep(),
			steps.NewImportUploadsStep(),
			steps.NewImportOthersStep(),
			steps.NewImportDatabaseStep(),
			steps.NewResetAdminPasswordStep(),
			steps.NewActivateThemeStep(),
			steps.NewGenerateAcornKeyStep(),
			steps.NewClearImportedCachesStep(),
			steps.NewFlushRewriteRulesStep(),
			steps.NewRefreshThemeCachesStep(),
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
