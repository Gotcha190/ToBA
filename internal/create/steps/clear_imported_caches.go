package steps

import (
	"os"
	"path/filepath"

	"github.com/gotcha190/ToBA/internal/create"
)

type ClearImportedCachesStep struct{}

func NewClearImportedCachesStep() *ClearImportedCachesStep {
	return &ClearImportedCachesStep{}
}

func (s *ClearImportedCachesStep) Name() string {
	return "Clear imported caches"
}

func (s *ClearImportedCachesStep) Run(ctx *create.Context) error {
	cacheDir := filepath.Join(ctx.Paths.WPContent, "cache")

	if ctx.DryRun {
		ctx.Logger.Info("Would remove: " + cacheDir)
		ctx.Logger.Info("Would run: lando wp cache flush")
		ctx.Logger.Info("Would run: lando wp acorn optimize:clear")
		return nil
	}

	if err := os.RemoveAll(cacheDir); err != nil {
		return err
	}

	if err := ctx.Runner.Run(ctx.Paths.Root, "lando", "wp", "cache", "flush"); err != nil {
		return err
	}

	if err := ctx.Runner.Run(ctx.Paths.Root, "lando", "wp", "acorn", "optimize:clear"); err != nil {
		ctx.Logger.Warning("Failed to clear Acorn cache: " + err.Error())
	}

	return nil
}
