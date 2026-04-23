package steps

import "github.com/gotcha190/toba/internal/create"
import "github.com/gotcha190/toba/internal/wordpress"

type ResetAdminPasswordStep struct{}

// NewResetAdminPasswordStep creates the pipeline step that restores the
// default local admin password.
//
// Returns:
// - a configured ResetAdminPasswordStep instance
func NewResetAdminPasswordStep() *ResetAdminPasswordStep {
	return &ResetAdminPasswordStep{}
}

// Name returns the human-readable pipeline label for this step.
//
// Returns:
// - the display name used by pipeline logging
func (s *ResetAdminPasswordStep) Name() string {
	return "Reset admin password"
}

// Run resets the local WordPress admin password to the default development
// value.
//
// Parameters:
// - ctx: shared create context containing project paths and runner access
//
// Returns:
// - an error when the password reset or fallback user creation fails
//
// Side effects:
//   - runs a single `lando wp eval ...` script that resets or creates the local
//     admin account
func (s *ResetAdminPasswordStep) Run(ctx *create.Context) error {
	if ctx.DryRun {
		ctx.Logger.Info("Would run: lando wp eval <reset-or-create tamago admin user script>")
		return nil
	}

	return wordpress.ResetAdminPassword(ctx.Runner, ctx.Paths.Root)
}
