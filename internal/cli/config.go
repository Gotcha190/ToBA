package cli

import (
	"bufio"
	"io"
	"os"
	"strings"

	"github.com/gotcha190/toba/internal/create"
)

// RunConfig initializes or refreshes the shared global ToBA configuration.
//
// Returns:
//   - an error when the configuration source cannot be resolved, the target
//     path cannot be written, or user interaction fails unexpectedly
//
// Side effects:
// - may prompt before overwriting the global config file
// - writes the global config file in the user's config directory
//
// Usage:
//
//	toba config
func RunConfig() error {
	return runConfigWithIO(os.Stdin, os.Stdout)
}

// runConfigWithIO executes the config bootstrap flow using injectable streams
// for prompting and logging.
//
// Parameters:
// - input: source used to read overwrite confirmations
// - output: destination used for informational logs and prompts
//
// Returns:
// - an error when config resolution or file writing fails
//
// Side effects:
// - reads from input when the global config already exists
// - writes prompts and status messages to output
// - writes the global config file on disk
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
