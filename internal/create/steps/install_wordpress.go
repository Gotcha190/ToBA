package steps

import (
	"github.com/gotcha190/toba/internal/create"
	"github.com/gotcha190/toba/internal/wordpress"
)

type InstallWordPressStep struct{}

// NewInstallWordPressStep creates the pipeline step that downloads and installs
// WordPress inside the local Lando project.
//
// Parameters:
// - none
//
// Returns:
// - a configured InstallWordPressStep instance
func NewInstallWordPressStep() *InstallWordPressStep {
	return &InstallWordPressStep{}
}

// Name returns the human-readable pipeline label for this step.
//
// Parameters:
// - none
//
// Returns:
// - the display name used by pipeline logging
func (s *InstallWordPressStep) Name() string {
	return "Install WordPress"
}

// Run downloads WordPress, creates wp-config.php, and performs the initial
// site installation.
//
// Parameters:
// - ctx: shared create context containing project paths, config, and runner access
//
// Returns:
// - an error when WordPress download, configuration, or installation fails
//
// Side effects:
// - may write WordPress core files and wp-config.php through WP-CLI
// - batches the bootstrap commands through Lando unless dry-run mode is enabled
func (s *InstallWordPressStep) Run(ctx *create.Context) error {
	if ctx.DryRun {
		ctx.Logger.Info("Would run batched bootstrap: lando ssh -s appserver -c cd /app && wp core download --locale='pl_PL' && wp config create --dbname='wordpress' --dbuser='wordpress' --dbpass='wordpress' --dbhost='database' --dbcharset='utf8mb4'")
		ctx.Logger.Info("Would run: lando wp core install --url=" + ctx.Config.Domain + " --title=" + wordpress.ProjectTitle(ctx.Config.Name) + " --admin_user=tamago --admin_email=email@email.pl --admin_password=tamago")
		return nil
	}

	return wordpress.Install(ctx.Runner, ctx.Paths.Root, ctx.Config)
}
