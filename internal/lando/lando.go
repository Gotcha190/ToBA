package lando

import (
	"bytes"
	"text/template"

	"github.com/gotcha190/toba/internal/create"
	"github.com/gotcha190/toba/internal/templates"
)

// RenderConfig renders the embedded .lando.yml template using the provided
// project configuration.
//
// Parameters:
// - config: normalized project configuration used to fill the Lando template
//
// Returns:
// - the rendered `.lando.yml` content
// - an error when the template cannot be read, parsed, or executed
func RenderConfig(config create.ProjectConfig) ([]byte, error) {
	templateBody, err := templates.Read("lando/.lando.yml")
	if err != nil {
		return nil, err
	}

	tpl, err := template.New(".lando.yml").Option("missingkey=error").Parse(string(templateBody))
	if err != nil {
		return nil, err
	}

	var rendered bytes.Buffer
	if err := tpl.Execute(&rendered, config); err != nil {
		return nil, err
	}

	return rendered.Bytes(), nil
}

// Start runs `lando start` in the project directory and wraps failures with a
// coded error.
//
// Parameters:
// - runner: command runner used to launch Lando
// - projectDir: local project root in which `lando start` should run
//
// Returns:
// - an error when the Lando start command fails
//
// Side effects:
// - starts the local Lando app
func Start(runner create.CommandRunner, projectDir string) error {
	if err := runner.Run(projectDir, "lando", "start"); err != nil {
		return create.NewCodedError("LANDO_START_FAILED", "lando start failed", err)
	}

	return nil
}
