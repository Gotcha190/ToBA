package cli

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/gotcha190/ToBA/internal/create"
	"github.com/gotcha190/ToBA/internal/create/steps"
)

type CreateOptions struct {
	Name        string
	PHPVersion  string
	Domain      string
	StarterRepo string
	DryRun      bool
}

func RunCreate(opts CreateOptions) error {
	return runCreateWithIO(opts, create.ExecRunner{}, os.Stdin, os.Stdout)
}

func runCreateWithRunner(opts CreateOptions, runner create.CommandRunner) error {
	return runCreateWithIO(opts, runner, strings.NewReader("n\n"), io.Discard)
}

func runCreateWithIO(opts CreateOptions, runner create.CommandRunner, input io.Reader, output io.Writer) error {
	config, envPath, _, err := create.ResolveEnvConfig()
	if err != nil {
		return err
	}
	if envPath != "" {
		fmt.Fprintf(output, "Using config from %s\n", envPath)
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

	ctx := create.NewContext(cwd, config, create.ConsoleLogger{}, runner)

	pipeline := create.Pipeline{
		Steps: []create.Step{
			steps.NewCollectConfigStep(),
			steps.NewProjectDirStep(),
			steps.NewGenerateLandoConfigStep(),
			steps.NewStartLandoStep(),
			steps.NewInstallWordPressStep(),
			steps.NewInstallThemeStep(),
			steps.NewBuildThemeStep(),
		},
	}

	err = pipeline.Run(ctx)
	if err != nil {
		maybeCleanupFailedInstall(ctx, input, output)
		return err
	}

	return nil
}

func maybeCleanupFailedInstall(ctx *create.Context, input io.Reader, output io.Writer) {
	if ctx == nil || ctx.DryRun || !ctx.ProjectCreated {
		return
	}

	if _, err := os.Stat(ctx.Paths.Root); err != nil {
		return
	}

	fmt.Fprintf(output, "Delete failed installation at %s? [y/N]: ", ctx.Paths.Root)
	reader := bufio.NewReader(input)
	answer, err := reader.ReadString('\n')
	if err != nil && answer == "" {
		fmt.Fprintln(output)
		return
	}

	answer = strings.TrimSpace(strings.ToLower(answer))
	if answer != "y" && answer != "yes" {
		return
	}

	if err := bestEffortDestroyLandoApp(ctx, output); err != nil {
		fmt.Fprintf(output, "Failed to destroy Lando app in %s: %v\n", ctx.Paths.Root, err)
	}

	if err := os.RemoveAll(ctx.Paths.Root); err != nil {
		fmt.Fprintf(output, "Failed to remove %s: %v\n", ctx.Paths.Root, err)
		return
	}

	fmt.Fprintf(output, "Removed failed installation: %s\n", ctx.Paths.Root)
}

func bestEffortDestroyLandoApp(ctx *create.Context, output io.Writer) error {
	landoFilePath := filepath.Join(ctx.Paths.Root, ".lando.yml")
	if _, err := os.Stat(landoFilePath); os.IsNotExist(err) {
		return nil
	} else if err != nil {
		return err
	}

	fmt.Fprintf(output, "Destroying Lando app in %s\n", ctx.Paths.Root)
	return ctx.Runner.Run(ctx.Paths.Root, "lando", "destroy", "-y")
}
