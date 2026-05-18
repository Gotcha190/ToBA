package create

import (
	"errors"
	"sort"
)

type graphNodeState struct {
	def        StepNode
	index      int
	remaining  int
	dependents []string
}

func (p *Pipeline) buildDependencyGraphStates() (map[string]*graphNodeState, error) {
	states := make(map[string]*graphNodeState, len(p.Nodes))
	for index, node := range p.Nodes {
		if node.Step == nil {
			return nil, errors.New("pipeline node step is nil")
		}

		nodeID := node.ID
		if nodeID == "" {
			nodeID = node.Step.Name()
		}
		if _, exists := states[nodeID]; exists {
			return nil, errors.New("duplicate pipeline node id: " + nodeID)
		}

		states[nodeID] = &graphNodeState{
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
				return nil, errors.New("unknown pipeline dependency: " + dependency)
			}
			parent.dependents = append(parent.dependents, state.def.ID)
		}
	}

	return states, nil
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
	type nodeResult struct {
		id     string
		index  int
		timing StepTiming
		err    error
	}

	states, err := p.buildDependencyGraphStates()
	if err != nil {
		return err
	}

	ready := make([]*graphNodeState, 0, len(states))
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

	startNode := func(state *graphNodeState) {
		running++

		ctx.Logger.Step(state.def.Step.Name())
		go func(current *graphNodeState) {
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

// runDependencyGraphSequential executes pipeline nodes one at a time in their
// declared order, while still validating dependency correctness.
//
// Parameters:
// - ctx: shared workflow context passed to every pipeline step
//
// Returns:
// - the first step error, or an error when the dependency graph is invalid
//
// Side effects:
// - runs nodes sequentially
// - writes progress, success, and error messages through the logger
// - records timings when a recorder is configured
func (p *Pipeline) runDependencyGraphSequential(ctx *Context) error {
	states, err := p.buildDependencyGraphStates()
	if err != nil {
		return err
	}

	completed := make(map[string]struct{}, len(states))
	for _, node := range p.Nodes {
		nodeID := node.ID
		if nodeID == "" {
			nodeID = node.Step.Name()
		}

		state := states[nodeID]
		for _, dependency := range state.def.DependsOn {
			if _, ok := completed[dependency]; ok {
				continue
			}
			return errors.New("pipeline node " + nodeID + " appears before dependency " + dependency + " in sequential mode")
		}

		ctx.Logger.Step(state.def.Step.Name())
		timing, stepErr := runStep(ctx, state.def.ID, state.def.Step)
		p.recordTiming(timing)
		if stepErr != nil {
			logStepError(ctx.Logger, state.def.Step, stepErr)
			return stepErr
		}

		ctx.Logger.Success(state.def.Step.Name())
		completed[state.def.ID] = struct{}{}
	}

	return nil
}
