package cli

import (
	"bufio"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/gotcha190/toba/internal/create"
)

// cleanupFailedInstall handles post-failure cleanup for a newly created local
// project directory.
//
// Parameters:
// - ctx: create workflow context describing the failed run
// - input: reader used to collect interactive confirmation from the user
//
// Returns:
// - null
//
// Side effects:
// - prompts the user for confirmation
// - may destroy the local Lando app
// - may remove the failed project directory from disk
func cleanupFailedInstall(ctx *create.Context, input io.Reader) {
	if ctx == nil || ctx.DryRun || !ctx.ProjectCreated {
		return
	}

	if _, err := os.Stat(ctx.Paths.Root); err != nil {
		return
	}

	ctx.Logger.Prompt("Delete failed installation at " + ctx.Paths.Root + "? [y/N]: ")
	reader := bufio.NewReader(input)
	answer, err := reader.ReadString('\n')
	if err != nil && answer == "" {
		ctx.Logger.Info("")
		return
	}

	answer = strings.TrimSpace(strings.ToLower(answer))
	if answer != "y" && answer != "yes" {
		return
	}

	if err := destroyLandoApp(ctx); err != nil {
		ctx.Logger.Error("Failed to destroy Lando app in " + ctx.Paths.Root + ": " + err.Error())
		ctx.Logger.Error("Keeping project directory because removing it now could make manual Lando cleanup harder: " + ctx.Paths.Root)
		return
	}

	if err := os.RemoveAll(ctx.Paths.Root); err != nil {
		ctx.Logger.Warning("Failed to remove " + ctx.Paths.Root + ": " + err.Error())
		return
	}

	ctx.Logger.Success("Removed failed installation: " + ctx.Paths.Root)
}

// destroyLandoApp destroys the local Lando app for a failed installation when
// the project root already contains a .lando.yml file.
//
// Parameters:
// - ctx: create workflow context containing the project root and runner
//
// Returns:
// - an error when Lando destruction fails or the marker file cannot be checked
//
// Side effects:
// - runs `lando destroy -y` in the project directory when applicable
func destroyLandoApp(ctx *create.Context) error {
	landoFilePath := filepath.Join(ctx.Paths.Root, ".lando.yml")
	if _, err := os.Stat(landoFilePath); os.IsNotExist(err) {
		return nil
	} else if err != nil {
		return err
	}

	ctx.Logger.Info("Destroying Lando app in " + ctx.Paths.Root)
	return ctx.Runner.Run(ctx.Paths.Root, "lando", "destroy", "-y")
}
