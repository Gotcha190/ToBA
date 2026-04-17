package steps

import "github.com/gotcha190/toba/internal/create"
import "github.com/gotcha190/toba/internal/wordpress"

type ResetAdminPasswordStep struct{}

// NewResetAdminPasswordStep creates the pipeline step that restores the
// default local admin password.
//
// Parameters:
// - none
//
// Returns:
// - a configured ResetAdminPasswordStep instance
func NewResetAdminPasswordStep() *ResetAdminPasswordStep {
	return &ResetAdminPasswordStep{}
}

// Name returns the human-readable pipeline label for this step.
//
// Parameters:
// - none
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
// - an error when the password reset command fails
//
// Side effects:
// - runs `lando wp user update 1 --user_pass=tamago` unless dry-run mode is enabled
func (s *ResetAdminPasswordStep) Run(ctx *create.Context) error {
	if ctx.DryRun {
		ctx.Logger.Info("Would run: lando wp user update 1 --user_pass=tamago")
		return nil
	}

	return wordpress.ResetAdminPassword(ctx.Runner, ctx.Paths.Root)
}
