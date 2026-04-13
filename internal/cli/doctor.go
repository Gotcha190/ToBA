package cli

import (
	"os"

	"github.com/gotcha190/ToBA/internal/create"
	"github.com/gotcha190/ToBA/internal/create/steps"
)

func RunDoctor() error {
	cwd, err := os.Getwd()
	if err != nil {
		return err
	}

	ctx := create.NewContext(cwd, create.ProjectConfig{}, create.ConsoleLogger{}, create.NoopRunner{})

	pipeline := create.Pipeline{
		Steps: []create.Step{
			steps.NewDoctorStep(),
		},
	}

	return pipeline.Run(ctx)
}
