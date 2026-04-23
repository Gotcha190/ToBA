package cli

import (
	"os"

	"github.com/gotcha190/toba/internal/create"
	"github.com/gotcha190/toba/internal/create/steps"
)

// RunDoctor executes the dependency checks required by the full create
// workflow.
//
// Returns:
//   - an error when the current working directory cannot be resolved or one of
//     the required dependencies is missing
//
// Side effects:
// - writes doctor status messages to stdout
//
// Usage:
//
//	toba doctor
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
