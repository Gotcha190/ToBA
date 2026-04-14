package steps

import (
	"fmt"

	"github.com/gotcha190/ToBA/internal/create"
	"github.com/gotcha190/ToBA/internal/project"
	"github.com/gotcha190/ToBA/internal/templates"
	"github.com/gotcha190/ToBA/internal/wordpress"
)

type ImportDatabaseStep struct{}

func NewImportDatabaseStep() *ImportDatabaseStep {
	return &ImportDatabaseStep{}
}

func (s *ImportDatabaseStep) Name() string {
	return "Import database"
}

func (s *ImportDatabaseStep) Run(ctx *create.Context) error {
	archives, err := listTemplateFiles("wordpress/database", ".gz")
	if err != nil {
		return err
	}
	if len(archives) != 1 {
		return fmt.Errorf("expected 1 database archive in wordpress/database, found %d", len(archives))
	}

	targetURL, err := wordpress.LocalHTTPSURL(ctx.Config.Domain)
	if err != nil {
		return err
	}

	if ctx.DryRun {
		ctx.Logger.Info("Would extract: " + archives[0] + " -> " + relativePath(ctx.Paths.Root, ctx.Paths.DatabaseSQL))
		ctx.Logger.Info("Would run: lando db-import " + relativePath(ctx.Paths.Root, ctx.Paths.DatabaseSQL))
		ctx.Logger.Info("Would run: lando wp search-replace <backup-url> " + targetURL + " --all-tables-with-prefix --skip-columns=guid")
		return nil
	}

	content, err := templates.Read(archives[0])
	if err != nil {
		return err
	}
	if err := project.WriteGzipFile(content, ctx.Paths.DatabaseSQL); err != nil {
		return fmt.Errorf("expand %s: %w", archives[0], err)
	}

	sourceURL, err := wordpress.BackupSourceURL(ctx.Paths.DatabaseSQL)
	if err != nil {
		return err
	}
	ctx.Logger.Info("Expanded: " + archives[0])

	if err := wordpress.ImportDatabase(ctx.Runner, ctx.Paths.Root, ctx.Paths.DatabaseSQL); err != nil {
		return err
	}
	if err := wordpress.SearchReplace(ctx.Runner, ctx.Paths.Root, sourceURL, targetURL); err != nil {
		return err
	}

	return nil
}
