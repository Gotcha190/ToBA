package steps

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/gotcha190/ToBA/internal/create"
	"github.com/gotcha190/ToBA/internal/project"
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
	if ctx.StarterData.DatabasePath == "" {
		return fmt.Errorf("database starter file is not prepared")
	}

	targetURL, err := wordpress.LocalHTTPSURL(ctx.Config.Domain)
	if err != nil {
		return err
	}

	if ctx.DryRun {
		ctx.Logger.Info("Would prepare database: " + ctx.StarterData.DatabasePath + " -> " + relativePath(ctx.Paths.Root, ctx.Paths.DatabaseSQL))
		ctx.Logger.Info("Would run: lando db-import " + relativePath(ctx.Paths.Root, ctx.Paths.DatabaseSQL))
		sourceURL := ctx.StarterData.SourceURL
		if sourceURL == "" {
			sourceURL = "<backup-url>"
		}
		ctx.Logger.Info("Would run: lando wp search-replace " + sourceURL + " " + targetURL + " --all-tables-with-prefix --skip-columns=guid")
		return nil
	}

	switch strings.ToLower(filepath.Ext(ctx.StarterData.DatabasePath)) {
	case ".gz":
		if err := project.WriteGzipPath(ctx.StarterData.DatabasePath, ctx.Paths.DatabaseSQL); err != nil {
			return fmt.Errorf("expand %s: %w", ctx.StarterData.DatabasePath, err)
		}
	case ".sql":
		if err := project.CopyFile(ctx.StarterData.DatabasePath, ctx.Paths.DatabaseSQL); err != nil {
			return fmt.Errorf("copy %s: %w", ctx.StarterData.DatabasePath, err)
		}
	default:
		return fmt.Errorf("unsupported database file: %s", ctx.StarterData.DatabasePath)
	}

	sourceURL := ctx.StarterData.SourceURL
	if sourceURL == "" {
		sourceURL, err = wordpress.BackupSourceURL(ctx.Paths.DatabaseSQL)
		if err != nil {
			return err
		}
	}
	ctx.Logger.Info("Prepared database: " + ctx.StarterData.DatabasePath)

	if err := wordpress.ImportDatabase(ctx.Runner, ctx.Paths.Root, ctx.Paths.DatabaseSQL); err != nil {
		return err
	}
	if err := wordpress.SearchReplace(ctx.Runner, ctx.Paths.Root, sourceURL, targetURL); err != nil {
		return err
	}

	return nil
}
