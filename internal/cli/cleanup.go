package cli

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/gotcha190/ToBA/internal/create"
)

func cleanupFailedInstall(ctx *create.Context, input io.Reader, output io.Writer) {
	if ctx == nil || ctx.DryRun || !ctx.ProjectCreated {
		return
	}

	if _, err := os.Stat(ctx.Paths.Root); err != nil {
		return
	}

	fmt.Fprintf(output, "Delete failed installation at %s? [y/N]: ", ctx.Paths.Root)
	reader := bufio.NewReader(input)
	answer, err := reader.ReadString('\n')
	if err != nil && answer == "" {
		fmt.Fprintln(output)
		return
	}

	answer = strings.TrimSpace(strings.ToLower(answer))
	if answer != "y" && answer != "yes" {
		return
	}

	if err := destroyLandoApp(ctx, output); err != nil {
		fmt.Fprintf(output, "Failed to destroy Lando app in %s: %v\n", ctx.Paths.Root, err)
	}

	if err := os.RemoveAll(ctx.Paths.Root); err != nil {
		fmt.Fprintf(output, "Failed to remove %s: %v\n", ctx.Paths.Root, err)
		return
	}

	fmt.Fprintf(output, "Removed failed installation: %s\n", ctx.Paths.Root)
}

func destroyLandoApp(ctx *create.Context, output io.Writer) error {
	landoFilePath := filepath.Join(ctx.Paths.Root, ".lando.yml")
	if _, err := os.Stat(landoFilePath); os.IsNotExist(err) {
		return nil
	} else if err != nil {
		return err
	}

	fmt.Fprintf(output, "Destroying Lando app in %s\n", ctx.Paths.Root)
	return ctx.Runner.Run(ctx.Paths.Root, "lando", "destroy", "-y")
}
