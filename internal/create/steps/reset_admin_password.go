package steps

import "github.com/gotcha190/toba/internal/create"
import "github.com/gotcha190/toba/internal/wordpress"

type ResetAdminPasswordStep struct{}

func NewResetAdminPasswordStep() *ResetAdminPasswordStep {
	return &ResetAdminPasswordStep{}
}

func (s *ResetAdminPasswordStep) Name() string {
	return "Reset admin password"
}

func (s *ResetAdminPasswordStep) Run(ctx *create.Context) error {
	if ctx.DryRun {
		ctx.Logger.Info("Would run: lando wp user update 1 --user_pass=tamago")
		return nil
	}

	return wordpress.ResetAdminPassword(ctx.Runner, ctx.Paths.Root)
}
