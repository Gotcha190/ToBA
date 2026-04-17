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
	Name        string
	PHPVersion  string
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
	if opts.StarterRepo != "" {
		config.StarterRepo = opts.StarterRepo
	}
	if opts.SSHTarget != "" {
		config.SSHTarget = opts.SSHTarget
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
