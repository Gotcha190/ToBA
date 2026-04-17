package cli

import (
	"os"

	"github.com/gotcha190/toba/internal/create"
	"github.com/gotcha190/toba/internal/create/steps"
)

func RunDoctor() error {
	cwd, err := os.Getwd()
	if err != nil {
		return err
	}

	ctx := create.NewContext(cwd, create.ProjectConfig{}, create.NewConsoleLogger(os.Stdout), create.NoopRunner{})

	pipeline := create.Pipeline{
		Steps: []create.Step{
			steps.NewDoctorStep(),
		},
	}

	return pipeline.Run(ctx)
}
