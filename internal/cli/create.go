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

	pipeline := buildCreatePipeline(cwd, config)

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

// buildCreatePipeline constructs the create workflow dependency graph.
//
// Parameters:
// - baseDir: directory in which the project will be created
// - config: project configuration used to decide safe parallelization
//
// Returns:
// - the configured create pipeline
func buildCreatePipeline(baseDir string, config create.ProjectConfig) create.Pipeline {
	remoteBootstrapParallel := canParallelizeRemoteBootstrap(baseDir, config)

	prepareStarterDataStep := create.Step(steps.NewPrepareStarterDataStep())
	prepareStarterDataDeps := []string{"collect-config"}
	projectDirDeps := []string{"prepare-starter-data"}
	installThemeDeps := []string{"project-dir", "prepare-starter-data"}
	importPluginsDeps := []string{"project-dir", "prepare-starter-data"}
	importUploadsDeps := []string{"project-dir", "prepare-starter-data"}
	importOthersDeps := []string{"project-dir", "prepare-starter-data"}
	importDatabaseDeps := []string{"install-wordpress", "prepare-starter-data"}

	if remoteBootstrapParallel {
		prepareStarterDataStep = forcedRemotePrepareStarterDataStep{base: steps.NewPrepareStarterDataStep()}
		projectDirDeps = []string{"collect-config"}
		installThemeDeps = []string{"project-dir"}
	}

	return create.Pipeline{
		Nodes: []create.StepNode{
			{ID: "collect-config", Step: steps.NewCollectConfigStep()},
			{ID: "prepare-starter-data", Step: prepareStarterDataStep, DependsOn: prepareStarterDataDeps},
			{ID: "project-dir", Step: steps.NewProjectDirStep(), DependsOn: projectDirDeps},
			{ID: "generate-lando-config", Step: steps.NewGenerateLandoConfigStep(), DependsOn: []string{"project-dir"}},
			{ID: "start-lando", Step: steps.NewStartLandoStep(), DependsOn: []string{"generate-lando-config"}},
			{ID: "install-theme", Step: steps.NewInstallThemeStep(), DependsOn: installThemeDeps},
			{ID: "import-plugins", Step: steps.NewImportPluginsStep(), DependsOn: importPluginsDeps},
			{ID: "import-uploads", Step: steps.NewImportUploadsStep(), DependsOn: importUploadsDeps},
			{ID: "import-others", Step: steps.NewImportOthersStep(), DependsOn: importOthersDeps},
			{ID: "install-wordpress", Step: steps.NewInstallWordPressStep(), DependsOn: []string{"start-lando"}},
			{ID: "build-theme", Step: steps.NewBuildThemeStep(), DependsOn: []string{"start-lando", "install-theme", "import-plugins"}},
			{ID: "import-database", Step: steps.NewImportDatabaseStep(), DependsOn: importDatabaseDeps},
			{ID: "reset-admin-password", Step: steps.NewResetAdminPasswordStep(), DependsOn: []string{"import-database"}},
			{ID: "activate-theme", Step: steps.NewActivateThemeStep(), DependsOn: []string{"import-database", "install-theme", "build-theme"}},
			{ID: "generate-acorn-key", Step: steps.NewGenerateAcornKeyStep(), DependsOn: []string{"activate-theme"}},
			{ID: "clear-imported-caches", Step: steps.NewClearImportedCachesStep(), DependsOn: []string{"import-database"}},
			{ID: "flush-rewrite-rules", Step: steps.NewFlushRewriteRulesStep(), DependsOn: []string{"activate-theme"}},
			{ID: "refresh-theme-caches", Step: steps.NewRefreshThemeCachesStep(), DependsOn: []string{"generate-acorn-key", "clear-imported-caches", "flush-rewrite-rules", "import-others"}},
		},
	}
}

// canParallelizeRemoteBootstrap reports whether remote starter preparation can
// overlap with local project directory creation.
//
// Parameters:
// - baseDir: directory in which the project will be created
// - config: project configuration to inspect
//
// Returns:
// - true when remote starter data can be prepared in parallel with project setup
func canParallelizeRemoteBootstrap(baseDir string, config create.ProjectConfig) bool {
	normalized := config
	if err := normalized.Normalize(); err != nil {
		return false
	}

	if strings.TrimSpace(normalized.SSHTarget) == "" || strings.TrimSpace(normalized.RemoteWordPressRoot) == "" {
		return false
	}

	paths := create.NewProjectPaths(baseDir, normalized.Name)
	info, err := os.Stat(paths.Root)
	if err == nil {
		return !info.IsDir()
	}

	return os.IsNotExist(err)
}

type forcedRemotePrepareStarterDataStep struct {
	base create.Step
}

// Name returns the wrapped prepare starter data step name.
//
// Returns:
// - the step name
func (s forcedRemotePrepareStarterDataStep) Name() string {
	return s.base.Name()
}

// Run forces remote starter data mode before running the wrapped step.
//
// Parameters:
// - ctx: shared create workflow context
//
// Returns:
// - an error when the wrapped step fails
//
// Side effects:
// - updates ctx.StarterData.Mode before delegating execution
func (s forcedRemotePrepareStarterDataStep) Run(ctx *create.Context) error {
	ctx.StarterData.Mode = "remote"
	return s.base.Run(ctx)
}
