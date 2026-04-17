package cli

import (
	"os"

	"github.com/gotcha190/ToBA/internal/create"
)

func RunConfigInit() error {
	cwd, err := os.Getwd()
	if err != nil {
		return err
	}

	sourcePath, targetPath, err := create.CopyLocalEnvToGlobal(cwd)
	if err != nil {
		return err
	}

	create.NewConsoleLogger(os.Stdout).Info("Copied config from " + sourcePath + " to " + targetPath)
	return nil
}
