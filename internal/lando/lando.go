package lando

import (
	"bytes"
	"text/template"

	"github.com/gotcha190/toba/internal/create"
	"github.com/gotcha190/toba/internal/templates"
)

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

func Start(runner create.CommandRunner, projectDir string) error {
	if err := runner.Run(projectDir, "lando", "start"); err != nil {
		return create.NewCodedError("LANDO_START_FAILED", "lando start failed", err)
	}

	return nil
}
