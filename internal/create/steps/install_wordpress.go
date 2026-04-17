package steps

import (
	"github.com/gotcha190/toba/internal/create"
	"github.com/gotcha190/toba/internal/wordpress"
)

type InstallWordPressStep struct{}

func NewInstallWordPressStep() *InstallWordPressStep {
	return &InstallWordPressStep{}
}

func (s *InstallWordPressStep) Name() string {
	return "Install WordPress"
}

func (s *InstallWordPressStep) Run(ctx *create.Context) error {
	if ctx.DryRun {
		ctx.Logger.Info("Would run: lando wp core download --locale=pl_PL")
		ctx.Logger.Info("Would run: lando wp config create --dbname=wordpress --dbuser=wordpress --dbpass=wordpress --dbhost=database")
		ctx.Logger.Info("Would run: lando wp core install --url=" + ctx.Config.Domain + " --title=" + wordpress.ProjectTitle(ctx.Config.Name) + " --admin_user=tamago --admin_email=email@email.pl --admin_password=tamago")
		return nil
	}

	return wordpress.Install(ctx.Runner, ctx.Paths.Root, ctx.Config)
}
