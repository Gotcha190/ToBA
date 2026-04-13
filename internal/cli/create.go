package cli

import (
	"os"

	"github.com/gotcha190/ToBA/internal/create"
	"github.com/gotcha190/ToBA/internal/create/steps"
)

type CreateOptions struct {
	Name       string
	PHPVersion string
	Domain     string
	Database   string
	DryRun     bool
}

func RunCreate(opts CreateOptions) error {
	return runCreateWithRunner(opts, create.ExecRunner{})
}

func runCreateWithRunner(opts CreateOptions, runner create.CommandRunner) error {
	cwd, err := os.Getwd()
	if err != nil {
		return err
	}

	config := create.ProjectConfig{
		Name:       opts.Name,
		PHPVersion: opts.PHPVersion,
		Domain:     opts.Domain,
		Database:   opts.Database,
		DryRun:     opts.DryRun,
	}

	ctx := create.NewContext(cwd, config, create.ConsoleLogger{}, runner)

	pipeline := create.Pipeline{
		Steps: []create.Step{
			steps.NewCollectConfigStep(),
			steps.NewProjectDirStep(),
			steps.NewGenerateLandoConfigStep(),
			steps.NewStartLandoStep(),
			steps.NewInstallWordPressStep(),
		},
	}

	return pipeline.Run(ctx)
}
