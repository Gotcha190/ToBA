package steps

import (
	"fmt"

	"github.com/gotcha190/toba/internal/create"
	"github.com/gotcha190/toba/internal/doctor"
)

type DoctorStep struct{}

func NewDoctorStep() *DoctorStep {
	return &DoctorStep{}
}

func (s *DoctorStep) Name() string {
	return "Doctor check"
}

func (s *DoctorStep) Run(ctx *create.Context) error {

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
