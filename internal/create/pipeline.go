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
