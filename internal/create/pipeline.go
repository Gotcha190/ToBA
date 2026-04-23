package create

import (
	"errors"
	"sort"
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

// runDependencyGraph executes pipeline nodes as soon as their dependencies
// complete.
//
// Parameters:
// - ctx: shared workflow context passed to every pipeline step
//
// Returns:
// - the first step error, or an error when the dependency graph is invalid
//
// Side effects:
// - runs ready steps concurrently
// - writes progress, success, and error messages through the logger
// - records timings when a recorder is configured
func (p *Pipeline) runDependencyGraph(ctx *Context) error {
	type nodeState struct {
		def        StepNode
		index      int
		remaining  int
		dependents []string
	}

	type nodeResult struct {
		id     string
		index  int
		timing StepTiming
		err    error
	}

	states := make(map[string]*nodeState, len(p.Nodes))
	for index, node := range p.Nodes {
		if node.Step == nil {
			return errors.New("pipeline node step is nil")
		}

		nodeID := node.ID
		if nodeID == "" {
			nodeID = node.Step.Name()
		}
		if _, exists := states[nodeID]; exists {
			return errors.New("duplicate pipeline node id: " + nodeID)
		}

		states[nodeID] = &nodeState{
			def: StepNode{
				ID:        nodeID,
				Step:      node.Step,
				DependsOn: append([]string(nil), node.DependsOn...),
			},
			index:     index,
			remaining: len(node.DependsOn),
		}
	}

	for _, state := range states {
		for _, dependency := range state.def.DependsOn {
			parent, exists := states[dependency]
			if !exists {
				return errors.New("unknown pipeline dependency: " + dependency)
			}
			parent.dependents = append(parent.dependents, state.def.ID)
		}
	}

	ready := make([]*nodeState, 0, len(states))
	for _, state := range states {
		if state.remaining == 0 {
			ready = append(ready, state)
		}
	}

	sort.Slice(ready, func(i, j int) bool {
		return ready[i].index < ready[j].index
	})

	results := make(chan nodeResult, len(states))
	running := 0
	completed := 0
	stopScheduling := false
	var firstErr error
	firstErrIndex := len(p.Nodes)

	startNode := func(state *nodeState) {
		running++

		ctx.Logger.Step(state.def.Step.Name())
		go func(current *nodeState) {
			timing, err := runStep(ctx, current.def.ID, current.def.Step)
			results <- nodeResult{
				id:     current.def.ID,
				index:  current.index,
				timing: timing,
				err:    err,
			}
		}(state)
	}

	for {
		for !stopScheduling && len(ready) > 0 {
			current := ready[0]
			ready = ready[1:]
			startNode(current)
		}

		if running == 0 {
			break
		}

		result := <-results
		running--
		completed++

		state := states[result.id]

		p.recordTiming(result.timing)
		if result.err != nil {
			logStepError(ctx.Logger, state.def.Step, result.err)
			stopScheduling = true
			if firstErr == nil || result.index < firstErrIndex {
				firstErr = result.err
				firstErrIndex = result.index
			}
			continue
		}

		ctx.Logger.Success(state.def.Step.Name())

		if stopScheduling {
			continue
		}

		for _, dependentID := range state.dependents {
			dependent := states[dependentID]
			dependent.remaining--
			if dependent.remaining == 0 {
				ready = append(ready, dependent)
			}
		}
		sort.Slice(ready, func(i, j int) bool {
			return ready[i].index < ready[j].index
		})
	}

	if firstErr != nil {
		return firstErr
	}

	if completed != len(states) {
		return errors.New("pipeline dependency graph did not complete; check for dependency cycles")
	}

	return nil
}

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

type synchronizedLogger struct {
	next Logger
	mu   sync.Mutex
}

// newSynchronizedLogger wraps next with a mutex-protected logger.
//
// Parameters:
// - next: logger to wrap
//
// Returns:
// - a synchronized logger, nil, or next when it is already synchronized
func newSynchronizedLogger(next Logger) Logger {
	if next == nil {
		return nil
	}

	if _, ok := next.(*synchronizedLogger); ok {
		return next
	}

	return &synchronizedLogger{next: next}
}

// Step writes a synchronized step message.
//
// Parameters:
// - msg: step label to display
//
// Returns:
// - null
//
// Side effects:
// - forwards the message to the wrapped logger
func (l *synchronizedLogger) Step(msg string) {
	l.withLock(func() {
		l.next.Step(msg)
	})
}

// Info writes a synchronized informational message.
//
// Parameters:
// - msg: informational message to display
//
// Returns:
// - null
//
// Side effects:
// - forwards the message to the wrapped logger
func (l *synchronizedLogger) Info(msg string) {
	l.withLock(func() {
		l.next.Info(msg)
	})
}

// Prompt writes a synchronized prompt message.
//
// Parameters:
// - msg: prompt text to display
//
// Returns:
// - null
//
// Side effects:
// - forwards the message to the wrapped logger
func (l *synchronizedLogger) Prompt(msg string) {
	l.withLock(func() {
		l.next.Prompt(msg)
	})
}

// Warning writes a synchronized warning message.
//
// Parameters:
// - msg: warning message to display
//
// Returns:
// - null
//
// Side effects:
// - forwards the message to the wrapped logger
func (l *synchronizedLogger) Warning(msg string) {
	l.withLock(func() {
		l.next.Warning(msg)
	})
}

// Success writes a synchronized success message.
//
// Parameters:
// - msg: success message to display
//
// Returns:
// - null
//
// Side effects:
// - forwards the message to the wrapped logger
func (l *synchronizedLogger) Success(msg string) {
	l.withLock(func() {
		l.next.Success(msg)
	})
}

// Error writes a synchronized error message.
//
// Parameters:
// - msg: error message to display
//
// Returns:
// - null
//
// Side effects:
// - forwards the message to the wrapped logger
func (l *synchronizedLogger) Error(msg string) {
	l.withLock(func() {
		l.next.Error(msg)
	})
}

// ErrorCode writes a synchronized coded error message.
//
// Parameters:
// - code: symbolic error code
// - msg: human-readable error message
//
// Returns:
// - null
//
// Side effects:
// - forwards the message to the wrapped logger
func (l *synchronizedLogger) ErrorCode(code string, msg string) {
	l.withLock(func() {
		l.next.ErrorCode(code, msg)
	})
}

// withLock runs fn while holding the logger mutex.
//
// Parameters:
// - fn: callback to execute while locked
//
// Returns:
// - null
//
// Side effects:
// - serializes access to the wrapped logger
func (l *synchronizedLogger) withLock(fn func()) {
	l.mu.Lock()
	defer l.mu.Unlock()

	fn()
}
