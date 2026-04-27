package create

import (
	"errors"
	"sync"
	"time"
)

// stages resolves the pipeline's stage list from explicit stages or legacy
// sequential steps.
//
// Returns:
// - configured stages, generated single-step stages, or nil when the pipeline is empty
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

// runSequentialStage executes each step in stage one after another.
//
// Parameters:
// - ctx: shared workflow context passed to every stage step
// - stage: stage definition to execute
//
// Returns:
// - the first step error, or nil when every step succeeds
//
// Side effects:
// - writes progress and success messages through the logger
// - records timings when a recorder is configured
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

// runParallelStage executes all steps in stage concurrently and waits for all
// of them to finish.
//
// Parameters:
// - ctx: shared workflow context passed to every stage step
// - stage: stage definition to execute
//
// Returns:
// - the first step error in stage order, or nil when every step succeeds
//
// Side effects:
// - writes progress and success messages through the logger
// - records timings when a recorder is configured
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

// runStep executes step and captures its timing metadata.
//
// Parameters:
// - ctx: shared workflow context passed to the step
// - stageName: logical stage or node identifier for timing output
// - step: pipeline step to execute
//
// Returns:
// - timing metadata for the executed step
// - an error when the step fails
//
// Side effects:
// - invokes the step's Run method
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

// recordTiming forwards timing metadata to the configured recorder.
//
// Parameters:
// - timing: completed step timing metadata
//
// Returns:
// - null
//
// Side effects:
// - calls the configured timing recorder when present
func (p *Pipeline) recordTiming(timing StepTiming) {
	if p.Recorder == nil {
		return
	}

	p.Recorder.RecordStepTiming(timing)
}

// logStepError writes a coded or generic step error through logger.
//
// Parameters:
// - logger: logger used for the error message
// - step: step that failed
// - err: error returned by the step
//
// Returns:
// - null
//
// Side effects:
// - writes one error line through logger
func logStepError(logger Logger, step Step, err error) {
	var coded codeCarrier
	if errors.As(err, &coded) {
		logger.ErrorCode(coded.Code(), step.Name()+": "+err.Error())
		return
	}

	logger.Error(step.Name() + ": " + err.Error())
}
