package cli

import (
	"bufio"
	"io"
	"os"
	"strings"

	"github.com/gotcha190/toba/internal/create"
)

func RunConfig() error {
	return runConfigWithIO(os.Stdin, os.Stdout)
}

func runConfigWithIO(input io.Reader, output io.Writer) error {
	cwd, err := os.Getwd()
	if err != nil {
		return err
	}

	content, sourcePath, targetPath, fromTemplate, err := create.ResolveGlobalEnvInitialization(cwd)
	if err != nil {
		return err
	}

	logger := create.NewConsoleLogger(output)
	if _, err := os.Stat(targetPath); err == nil {
		logger.Prompt("Overwrite existing global config at " + targetPath + "? [y/N]: ")
		reader := bufio.NewReader(input)
		answer, readErr := reader.ReadString('\n')
		if readErr != nil && answer == "" {
			logger.Info("No confirmation received; skipped updating global config: " + targetPath)
			return nil
		}

		answer = strings.TrimSpace(strings.ToLower(answer))
		if answer != "y" && answer != "yes" {
			logger.Info("Skipped updating global config: " + targetPath)
			return nil
		}
	} else if !os.IsNotExist(err) {
		return err
	}

	if err := create.WriteGlobalEnv(targetPath, content); err != nil {
		return err
	}

	if fromTemplate {
		if sourcePath == "embedded:.env.example" {
			logger.Info("Created global config from embedded .env.example: " + targetPath)
		} else {
			logger.Info("Created global config from " + sourcePath + ": " + targetPath)
		}
		logger.Info("Fill in the required values in " + targetPath)
		return nil
	}

	logger.Info("Copied config from " + sourcePath + " to " + targetPath)
	return nil
}
