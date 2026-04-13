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

	for _, check := range doctor.FullWorkflowChecks() {
		ctx.Logger.Info("Checking " + check.Name)
		if err := doctor.RunCheck(check); err != nil {
			ctx.Logger.Warning(fmt.Sprintf("%s not installed", check.Name))
			missing = append(missing, check.Binary)
			continue
		}
		ctx.Logger.Success(check.Name + " installed")
	}

	if len(missing) > 0 {
		return fmt.Errorf("missing tools for full create workflow: %v", missing)
	}

	return nil
}
