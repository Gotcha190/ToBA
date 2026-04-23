package steps

import (
	"fmt"

	"github.com/gotcha190/toba/internal/create"
	"github.com/gotcha190/toba/internal/doctor"
)

type DoctorStep struct{}

// NewDoctorStep creates the pipeline step that checks required external tools.
//
// Returns:
// - a configured DoctorStep instance
func NewDoctorStep() *DoctorStep {
	return &DoctorStep{}
}

// Name returns the human-readable pipeline label for this step.
//
// Returns:
// - the display name used by pipeline logging
func (s *DoctorStep) Name() string {
	return "Doctor check"
}

// Run verifies that all binaries needed by the full create workflow are
// available in PATH.
//
// Parameters:
// - ctx: shared create context providing logger access
//
// Returns:
// - an error when any required binary is missing
//
// Side effects:
// - writes one log entry per dependency check
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
