package steps

import (
	"fmt"

	"github.com/gotcha190/ToBA/internal/create"
	"github.com/gotcha190/ToBA/internal/doctor"
)

type DoctorStep struct{}

func NewDoctorStep() *DoctorStep {
	return &DoctorStep{}
}

func (s *DoctorStep) Name() string {
	return "Doctor check"
}

func (s *DoctorStep) Run(ctx *create.Context) error {
	ctx.Logger.Info("Current milestone requirements:")
	ctx.Logger.Info("No external system dependencies are required for project scaffolding and .lando.yml generation.")
	ctx.Logger.Info("Full create workflow requirements:")

	var missing []string

	for _, result := range doctor.RunChecks(doctor.FullWorkflowChecks()) {
		ctx.Logger.Info("Checking " + result.Check.Name)
		if result.Err != nil {
			ctx.Logger.Warning(fmt.Sprintf("%s not installed", result.Check.Name))
			missing = append(missing, result.Check.Binary)
			continue
		}
		ctx.Logger.Success(result.Check.Name + " installed")
	}

	if len(missing) > 0 {
		return fmt.Errorf("missing tools for full create workflow: %v", missing)
	}

	return nil
}
