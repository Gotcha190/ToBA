package steps

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/gotcha190/ToBA/internal/create"
	"github.com/gotcha190/ToBA/internal/updraft"
)

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

	if ctx.DryRun {
		tempDir := filepath.Join(os.TempDir(), "toba-starter-dry-run")
		ctx.StarterData = create.StarterData{
			Mode:         starterDataModeLocal,
			TempDir:      tempDir,
			DatabasePath: tempPathFromSource(tempDir, "database", selection.Database),
			PluginsPaths: tempPathsFromSources(tempDir, "plugins", selection.Plugins),
			UploadsPaths: tempPathsFromSources(tempDir, "uploads", selection.Uploads),
			OthersPaths:  tempPathsFromSources(tempDir, "others", selection.Others),
			ThemePaths:   tempPathsFromSources(tempDir, "themes", selection.Themes),
		}
		return nil
	}

	tempDir, err := makeStarterTempDir()
	if err != nil {
		return err
	}

	databasePath, err := copyLocalFileToTemp(tempDir, "database", selection.Database)
	if err != nil {
		return err
	}
	pluginsPaths, err := copyLocalFilesToTemp(tempDir, "plugins", selection.Plugins)
	if err != nil {
		return err
	}
	uploadsPaths, err := copyLocalFilesToTemp(tempDir, "uploads", selection.Uploads)
	if err != nil {
		return err
	}
	othersPaths, err := copyLocalFilesToTemp(tempDir, "others", selection.Others)
	if err != nil {
		return err
	}
	themePaths, err := copyLocalFilesToTemp(tempDir, "themes", selection.Themes)
	if err != nil {
		return err
	}

	ctx.StarterData = create.StarterData{
		Mode:         starterDataModeLocal,
		TempDir:      tempDir,
		DatabasePath: databasePath,
		PluginsPaths: pluginsPaths,
		UploadsPaths: uploadsPaths,
		OthersPaths:  othersPaths,
		ThemePaths:   themePaths,
	}

	return nil
}
