package create

import (
	"errors"
	"testing"
)

type recordingLogger struct {
	steps     []string
	successes []string
	errors    []string
}

func (l *recordingLogger) Step(msg string) {
	l.steps = append(l.steps, msg)
}

func (l *recordingLogger) Info(msg string) {}

func (l *recordingLogger) Warning(msg string) {}

func (l *recordingLogger) Success(msg string) {
	l.successes = append(l.successes, msg)
}

func (l *recordingLogger) Error(msg string) {
	l.errors = append(l.errors, msg)
}

type pipelineTestStep struct {
	name string
	err  error
	hit  *[]string
}

func (s pipelineTestStep) Name() string {
	return s.name
}

func (s pipelineTestStep) Run(ctx *Context) error {
	*s.hit = append(*s.hit, s.name)
	return s.err
}

func TestPipelineStopsOnFirstError(t *testing.T) {
	expectedErr := errors.New("boom")
	var executed []string
	logger := &recordingLogger{}

	pipeline := Pipeline{
		Steps: []Step{
			pipelineTestStep{name: "step-1", hit: &executed},
			pipelineTestStep{name: "step-2", err: expectedErr, hit: &executed},
			pipelineTestStep{name: "step-3", hit: &executed},
		},
	}

	err := pipeline.Run(&Context{Logger: logger})
	if !errors.Is(err, expectedErr) {
		t.Fatalf("expected %v, got %v", expectedErr, err)
	}

	if len(executed) != 2 {
		t.Fatalf("expected 2 executed steps, got %d", len(executed))
	}
	if executed[0] != "step-1" || executed[1] != "step-2" {
		t.Fatalf("unexpected executed steps: %v", executed)
	}

	if len(logger.successes) != 1 || logger.successes[0] != "step-1" {
		t.Fatalf("unexpected success logs: %v", logger.successes)
	}
	if len(logger.errors) != 1 {
		t.Fatalf("expected a single error log, got %v", logger.errors)
	}
}
