package create

import "time"

type Step interface {
	Name() string
	Run(ctx *Context) error
}

type Stage struct {
	Name     string
	Steps    []Step
	Parallel bool
}

type StepNode struct {
	ID        string
	Step      Step
	DependsOn []string
}

type StepTiming struct {
	Stage      string
	Name       string
	StartedAt  time.Time
	FinishedAt time.Time
	Duration   time.Duration
}

type StepTimingRecorder interface {
	RecordStepTiming(timing StepTiming)
}

type Pipeline struct {
	Steps    []Step
	Stages   []Stage
	Nodes    []StepNode
	Recorder StepTimingRecorder
}

// Run executes each pipeline step in order and logs coded errors with their
// explicit error codes when available.
//
// Parameters:
// - ctx: shared workflow context passed to every pipeline step
//
// Returns:
// - an error as soon as the first step fails
//
// Side effects:
// - writes step progress and errors through the configured logger
// - invokes each step sequentially
func (p *Pipeline) Run(ctx *Context) error {
	if ctx == nil {
		return nil
	}

	originalLogger := ctx.Logger
	ctx.Logger = newSynchronizedLogger(ctx.Logger)
	defer func() {
		ctx.Logger = originalLogger
	}()

	if len(p.Nodes) > 0 {
		return p.runDependencyGraph(ctx)
	}

	for _, stage := range p.stages() {
		var err error
		if stage.Parallel {
			err = p.runParallelStage(ctx, stage)
		} else {
			err = p.runSequentialStage(ctx, stage)
		}
		if err != nil {
			return err
		}
	}

	return nil
}
