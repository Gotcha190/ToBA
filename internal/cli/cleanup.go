package cli

import (
	"bufio"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/gotcha190/toba/internal/create"
)

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
