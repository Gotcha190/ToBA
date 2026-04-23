package steps

import (
	"fmt"

	"github.com/gotcha190/toba/internal/create"
	"github.com/gotcha190/toba/internal/updraft"
)

// prepareLocalStarterData validates an existing project directory as a local
// Updraft backup source and copies its files into a temp workspace.
//
// Parameters:
// - ctx: shared create context containing project paths and mutable starter-data state
//
// Returns:
// - an error when the local backup set is empty, invalid, incomplete, or cannot be copied
//
// Side effects:
// - marks the run as using an existing project directory
// - populates ctx.StarterData with local backup paths
func prepareLocalStarterData(ctx *create.Context) error {
	selection, err := updraft.ScanProjectDir(ctx.Paths.Root)
	if err != nil {
		return fmt.Errorf("local project backup in %s is invalid: %w", ctx.Paths.Root, err)
	}
	if !selection.HasRecognizedFiles() {
		return fmt.Errorf("project directory %s exists but contains no recognizable Updraft backup files", ctx.Paths.Root)
	}
	if err := selection.ValidateLocalProjectSet(); err != nil {
		return fmt.Errorf("local project backup in %s is incomplete: %w", ctx.Paths.Root, err)
	}

	ctx.UseExistingProjectDir = true
	ctx.Logger.Info("Using local project backup folder: " + ctx.Paths.Root)

	ctx.StarterData = create.StarterData{
		Mode:         starterDataModeLocal,
		DatabasePath: selection.Database,
		PluginsPaths: append([]string(nil), selection.Plugins...),
		UploadsPaths: append([]string(nil), selection.Uploads...),
		OthersPaths:  append([]string(nil), selection.Others...),
		ThemePaths:   append([]string(nil), selection.Themes...),
	}

	return nil
}
