package cli

import (
	"fmt"
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

	fmt.Printf("Copied config from %s to %s\n", sourcePath, targetPath)
	return nil
}
