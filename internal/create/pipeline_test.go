package create

import (
	"errors"
	"sync"
	"testing"
	"time"
)

type recordingLogger struct {
	steps     []string
	successes []string
	errors    []string
	prompts   []string
	coded     []string
}

func (l *recordingLogger) Step(msg string) {
	l.steps = append(l.steps, msg)
}

func (l *recordingLogger) Info(msg string) {}

func (l *recordingLogger) Prompt(msg string) {
	l.prompts = append(l.prompts, msg)
}

func (l *recordingLogger) Warning(msg string) {}

func (l *recordingLogger) Success(msg string) {
	l.successes = append(l.successes, msg)
}

func (l *recordingLogger) Error(msg string) {
	l.errors = append(l.errors, msg)
}

func (l *recordingLogger) ErrorCode(code string, msg string) {
	l.coded = append(l.coded, code+": "+msg)
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

type recordingTimingRecorder struct {
	timings []StepTiming
}

func (r *recordingTimingRecorder) RecordStepTiming(timing StepTiming) {
	r.timings = append(r.timings, timing)
}

type gatedStep struct {
	name    string
	started chan<- string
	release <-chan struct{}
	err     error
}

func (s gatedStep) Name() string {
	return s.name
}

func (s gatedStep) Run(ctx *Context) error {
	if s.started != nil {
		s.started <- s.name
	}
	if s.release != nil {
		<-s.release
	}
	return s.err
}

func TestPipelineSequentialStepsRemainBackwardCompatible(t *testing.T) {
	expectedErr := errors.New("boom")
	var executed []string
	logger := &recordingLogger{}
	recorder := &recordingTimingRecorder{}

	pipeline := Pipeline{
		Recorder: recorder,
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
	if len(recorder.timings) != 2 {
		t.Fatalf("expected timings for executed steps, got %d", len(recorder.timings))
	}
}

func TestPipelineLogsCodedErrors(t *testing.T) {
	expectedErr := errors.New("boom")
	logger := &recordingLogger{}

	pipeline := Pipeline{
		Steps: []Step{
			pipelineTestStep{name: "step-1", err: NewCodedError("FAIL_CODE", "failed to do thing", expectedErr), hit: &[]string{}},
		},
	}

	err := pipeline.Run(&Context{Logger: logger})
	if !errors.Is(err, expectedErr) {
		t.Fatalf("expected wrapped error, got %v", err)
	}
	if len(logger.coded) != 1 {
		t.Fatalf("expected a single coded error log, got %v", logger.coded)
	}
	if len(logger.errors) != 0 {
		t.Fatalf("expected no plain error logs, got %v", logger.errors)
	}
}

func TestPipelineParallelStageWaitsForAllStepsAndReturnsFirstStageError(t *testing.T) {
	logger := &recordingLogger{}
	recorder := &recordingTimingRecorder{}
	started := make(chan string, 3)
	release := make(chan struct{})

	stageErr := errors.New("parallel boom")
	laterErr := errors.New("later boom")

	pipeline := Pipeline{
		Recorder: recorder,
		Stages: []Stage{
			{
				Name:     "parallel",
				Parallel: true,
				Steps: []Step{
					gatedStep{name: "step-1", started: started, release: release, err: stageErr},
					gatedStep{name: "step-2", started: started, release: release},
					gatedStep{name: "step-3", started: started, release: release, err: laterErr},
				},
			},
		},
	}

	errCh := make(chan error, 1)
	go func() {
		errCh <- pipeline.Run(&Context{Logger: logger})
	}()

	var seen []string
	timeout := time.After(2 * time.Second)
	for len(seen) < 3 {
		select {
		case name := <-started:
			seen = append(seen, name)
		case <-timeout:
			t.Fatalf("timed out waiting for parallel steps, saw %v", seen)
		}
	}

	select {
	case err := <-errCh:
		t.Fatalf("pipeline returned before all parallel steps completed: %v", err)
	default:
	}

	close(release)

	select {
	case err := <-errCh:
		if !errors.Is(err, stageErr) {
			t.Fatalf("expected first stage error %v, got %v", stageErr, err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for pipeline completion")
	}

	if len(logger.successes) != 1 || logger.successes[0] != "step-2" {
		t.Fatalf("unexpected success logs: %v", logger.successes)
	}
	if len(logger.errors) != 2 {
		t.Fatalf("expected two error logs, got %v", logger.errors)
	}
	if len(recorder.timings) != 3 {
		t.Fatalf("expected timings for all parallel steps, got %d", len(recorder.timings))
	}
}

func TestPipelineDependencyGraphStartsReadyDependentsWithoutWaitingForWholeLayer(t *testing.T) {
	logger := &recordingLogger{}

	startLandoDone := make(chan struct{})
	fileRestoreRelease := make(chan struct{})
	started := make(chan string, 4)

	pipeline := Pipeline{
		Nodes: []StepNode{
			{ID: "project-dir", Step: gatedStep{name: "project-dir", started: started}},
			{ID: "start-lando", Step: gatedStep{name: "start-lando", started: started, release: startLandoDone}, DependsOn: []string{"project-dir"}},
			{ID: "import-uploads", Step: gatedStep{name: "import-uploads", started: started, release: fileRestoreRelease}, DependsOn: []string{"project-dir"}},
			{ID: "install-wordpress", Step: gatedStep{name: "install-wordpress", started: started}, DependsOn: []string{"start-lando"}},
		},
	}

	errCh := make(chan error, 1)
	go func() {
		errCh <- pipeline.Run(&Context{Logger: logger})
	}()

	for range 3 {
		select {
		case <-started:
		case <-time.After(2 * time.Second):
			t.Fatal("timed out waiting for initial nodes to start")
		}
	}

	close(startLandoDone)

	select {
	case name := <-started:
		if name != "install-wordpress" {
			t.Fatalf("expected install-wordpress to start next, got %s", name)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for dependency to start")
	}

	select {
	case err := <-errCh:
		t.Fatalf("pipeline finished too early: %v", err)
	default:
	}

	close(fileRestoreRelease)

	select {
	case err := <-errCh:
		if err != nil {
			t.Fatalf("unexpected pipeline error: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for pipeline completion")
	}
}

func TestPipelineDependencyGraphWaitsForRunningNodesAfterError(t *testing.T) {
	logger := &recordingLogger{}
	started := make(chan string, 2)
	release := make(chan struct{})
	expectedErr := errors.New("boom")

	pipeline := Pipeline{
		Nodes: []StepNode{
			{ID: "fast-fail", Step: gatedStep{name: "fast-fail", started: started, err: expectedErr}},
			{ID: "slow-runner", Step: gatedStep{name: "slow-runner", started: started, release: release}},
			{ID: "never-start", Step: gatedStep{name: "never-start", started: started}, DependsOn: []string{"slow-runner"}},
		},
	}

	errCh := make(chan error, 1)
	go func() {
		errCh <- pipeline.Run(&Context{Logger: logger})
	}()

	for range 2 {
		select {
		case <-started:
		case <-time.After(2 * time.Second):
			t.Fatal("timed out waiting for running nodes")
		}
	}

	select {
	case name := <-started:
		t.Fatalf("unexpected extra started node: %s", name)
	default:
	}

	select {
	case err := <-errCh:
		t.Fatalf("pipeline returned before slow running node finished: %v", err)
	default:
	}

	close(release)

	select {
	case err := <-errCh:
		if !errors.Is(err, expectedErr) {
			t.Fatalf("expected %v, got %v", expectedErr, err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for pipeline completion")
	}
}

func TestSynchronizedLoggerSerializesConcurrentWrites(t *testing.T) {
	base := &concurrencyProbeLogger{}
	logger := newSynchronizedLogger(base)
	var wg sync.WaitGroup

	for range 8 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			logger.Info("hello")
		}()
	}

	wg.Wait()

	if base.maxConcurrent != 1 {
		t.Fatalf("expected serialized logger writes, got max concurrent=%d", base.maxConcurrent)
	}
}

type concurrencyProbeLogger struct {
	active        int
	maxConcurrent int
	mu            sync.Mutex
}

func (l *concurrencyProbeLogger) Step(msg string)                   { l.probe() }
func (l *concurrencyProbeLogger) Info(msg string)                   { l.probe() }
func (l *concurrencyProbeLogger) Prompt(msg string)                 { l.probe() }
func (l *concurrencyProbeLogger) Warning(msg string)                { l.probe() }
func (l *concurrencyProbeLogger) Success(msg string)                { l.probe() }
func (l *concurrencyProbeLogger) Error(msg string)                  { l.probe() }
func (l *concurrencyProbeLogger) ErrorCode(code string, msg string) { l.probe() }

func (l *concurrencyProbeLogger) probe() {
	l.mu.Lock()
	l.active++
	if l.active > l.maxConcurrent {
		l.maxConcurrent = l.active
	}
	l.mu.Unlock()

	time.Sleep(5 * time.Millisecond)

	l.mu.Lock()
	l.active--
	l.mu.Unlock()
}
