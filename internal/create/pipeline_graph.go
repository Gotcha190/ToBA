package create

import (
	"errors"
	"sort"
)

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
