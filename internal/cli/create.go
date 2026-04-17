package cli

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/gotcha190/ToBA/internal/create"
	"github.com/gotcha190/ToBA/internal/create/steps"
)

type CreateOptions struct {
	Name        string
	PHPVersion  string
	Domain      string
	StarterRepo string
	SSHTarget   string
	DryRun      bool
}

func RunCreate(opts CreateOptions) error {
	return runCreateWithIO(opts, create.ExecRunner{}, os.Stdin, os.Stdout)
}

func runCreateWithRunner(opts CreateOptions, runner create.CommandRunner) error {
	return runCreateWithIO(opts, runner, strings.NewReader("n\n"), io.Discard)
}

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
	if opts.Domain != "" {
		config.Domain = opts.Domain
	}
	if opts.StarterRepo != "" {
		config.StarterRepo = opts.StarterRepo
	}
	if opts.SSHTarget != "" {
		config.SSHTarget = opts.SSHTarget
	}
	config.DryRun = opts.DryRun

	if strings.TrimSpace(config.Name) == "" {
		if envPath != "" {
			return fmt.Errorf("project name cannot be empty; set TOBA_PROJECT_NAME in %s or pass project name as argument", envPath)
		}
		globalPath, pathErr := create.GlobalEnvPath()
		if pathErr != nil {
			return fmt.Errorf("project name cannot be empty")
		}
		return fmt.Errorf("project name cannot be empty; global config not found at %s. Run 'ToBA config init' in the ToBA repository first or pass project name as argument", globalPath)
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
			steps.NewClearImportedCachesStep(),
			steps.NewResetAdminPasswordStep(),
			steps.NewActivateThemeStep(),
			steps.NewGenerateAcornKeyStep(),
			steps.NewFlushRewriteRulesStep(),
			steps.NewRefreshThemeCachesStep(),
		},
	}

	err = pipeline.Run(ctx)
	if err != nil {
		cleanupFailedInstall(ctx, input)
		return err
	}

	return nil
}
