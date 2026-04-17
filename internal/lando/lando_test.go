package lando

import (
	"strings"
	"testing"

	"github.com/gotcha190/toba/internal/create"
)

func TestRenderConfig(t *testing.T) {
	rendered, err := RenderConfig(create.ProjectConfig{
		Name:       "demo",
		PHPVersion: "8.4",
		Domain:     "demo.lndo.site",
	})
	if err != nil {
		t.Fatalf("RenderConfig returned error: %v", err)
	}

	output := string(rendered)
	for _, expected := range []string{
		"database: mysql:8.0",
		"name: demo",
		`php: "8.4"`,
		"webroot: ./app",
		"php: config/php.ini",
		"PHP_IDE_CONFIG: projectName=demo",
		"type: node:18",
		"service: theme",
	} {
		if !strings.Contains(output, expected) {
			t.Fatalf("expected rendered config to contain %q, got:\n%s", expected, output)
		}
	}
	for _, unexpected := range []string{
		"type: mysql:8.0",
		"composer:",
		"wp:",
	} {
		if strings.Contains(output, unexpected) {
			t.Fatalf("expected rendered config not to contain %q, got:\n%s", unexpected, output)
		}
	}
}
