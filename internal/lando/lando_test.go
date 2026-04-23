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

type landoTestRunner struct {
	runCalls     []string
	captureCalls []string
	captureErr   error
}

func (r *landoTestRunner) Run(dir string, cmd string, args ...string) error {
	r.runCalls = append(r.runCalls, dir+"|"+cmd+" "+strings.Join(args, " "))
	return nil
}

func (r *landoTestRunner) CaptureOutput(dir string, cmd string, args ...string) (string, error) {
	r.captureCalls = append(r.captureCalls, dir+"|"+cmd+" "+strings.Join(args, " "))
	return "", r.captureErr
}

func TestStartUsesCaptureOutputForQuietSuccess(t *testing.T) {
	runner := &landoTestRunner{}

	if err := Start(runner, "/tmp/demo"); err != nil {
		t.Fatalf("Start returned error: %v", err)
	}

	if len(runner.runCalls) != 0 {
		t.Fatalf("expected no Run calls, got %#v", runner.runCalls)
	}
	if len(runner.captureCalls) != 1 || runner.captureCalls[0] != "/tmp/demo|lando start" {
		t.Fatalf("unexpected CaptureOutput calls: %#v", runner.captureCalls)
	}
}

func TestStartWrapsCaptureOutputError(t *testing.T) {
	expectedErr := create.NewCodedError("INNER", "inner", nil)
	runner := &landoTestRunner{captureErr: expectedErr}

	err := Start(runner, "/tmp/demo")
	if err == nil {
		t.Fatal("expected Start to fail")
	}
	if !strings.Contains(err.Error(), "lando start failed") {
		t.Fatalf("unexpected error: %v", err)
	}
}
