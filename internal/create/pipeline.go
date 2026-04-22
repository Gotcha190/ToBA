package create

import (
	"errors"
	"sync"
	"time"
)

type Step interface {
	Name() string
	Run(ctx *Context) error
}

type Stage struct {
	Name     string
	Steps    []Step
	Parallel bool
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

func (p *Pipeline) stages() []Stage {
	if len(p.Stages) > 0 {
		return p.Stages
	}

	if len(p.Steps) == 0 {
		return nil
	}

	stages := make([]Stage, 0, len(p.Steps))
	for _, step := range p.Steps {
		stages = append(stages, Stage{
			Name:  step.Name(),
			Steps: []Step{step},
		})
	}

	return stages
}

func (p *Pipeline) runSequentialStage(ctx *Context, stage Stage) error {
	for _, step := range stage.Steps {
		ctx.Logger.Step(step.Name())

		timing, err := runStep(ctx, stage.Name, step)
		p.recordTiming(timing)
		if err != nil {
			logStepError(ctx.Logger, step, err)
			return err
		}

		ctx.Logger.Success(step.Name())
	}

	return nil
}

func (p *Pipeline) runParallelStage(ctx *Context, stage Stage) error {
	type stepResult struct {
		timing StepTiming
		err    error
	}

	results := make([]stepResult, len(stage.Steps))
	var wg sync.WaitGroup

	for index, step := range stage.Steps {
		ctx.Logger.Step(step.Name())

		wg.Add(1)
		go func(i int, current Step) {
			defer wg.Done()
			timing, err := runStep(ctx, stage.Name, current)
			results[i] = stepResult{
				timing: timing,
				err:    err,
			}
		}(index, step)
	}

	wg.Wait()

	var firstErr error
	for index, step := range stage.Steps {
		p.recordTiming(results[index].timing)

		if results[index].err != nil {
			logStepError(ctx.Logger, step, results[index].err)
			if firstErr == nil {
				firstErr = results[index].err
			}
			continue
		}

		ctx.Logger.Success(step.Name())
	}

	return firstErr
}

func runStep(ctx *Context, stageName string, step Step) (StepTiming, error) {
	startedAt := time.Now()
	err := step.Run(ctx)
	finishedAt := time.Now()

	return StepTiming{
		Stage:      stageName,
		Name:       step.Name(),
		StartedAt:  startedAt,
		FinishedAt: finishedAt,
		Duration:   finishedAt.Sub(startedAt),
	}, err
}

func (p *Pipeline) recordTiming(timing StepTiming) {
	if p.Recorder == nil {
		return
	}

	p.Recorder.RecordStepTiming(timing)
}

func logStepError(logger Logger, step Step, err error) {
	var coded codeCarrier
	if errors.As(err, &coded) {
		logger.ErrorCode(coded.Code(), step.Name()+": "+err.Error())
		return
	}

	logger.Error(step.Name() + ": " + err.Error())
}

type synchronizedLogger struct {
	next Logger
	mu   sync.Mutex
}

func newSynchronizedLogger(next Logger) Logger {
	if next == nil {
		return nil
	}

	if _, ok := next.(*synchronizedLogger); ok {
		return next
	}

	return &synchronizedLogger{next: next}
}

func (l *synchronizedLogger) Step(msg string) {
	l.withLock(func() {
		l.next.Step(msg)
	})
}

func (l *synchronizedLogger) Info(msg string) {
	l.withLock(func() {
		l.next.Info(msg)
	})
}

func (l *synchronizedLogger) Prompt(msg string) {
	l.withLock(func() {
		l.next.Prompt(msg)
	})
}

func (l *synchronizedLogger) Warning(msg string) {
	l.withLock(func() {
		l.next.Warning(msg)
	})
}

func (l *synchronizedLogger) Success(msg string) {
	l.withLock(func() {
		l.next.Success(msg)
	})
}

func (l *synchronizedLogger) Error(msg string) {
	l.withLock(func() {
		l.next.Error(msg)
	})
}

func (l *synchronizedLogger) ErrorCode(code string, msg string) {
	l.withLock(func() {
		l.next.ErrorCode(code, msg)
	})
}

func (l *synchronizedLogger) withLock(fn func()) {
	l.mu.Lock()
	defer l.mu.Unlock()

	fn()
}
