package create

import "errors"

type Step interface {
	Name() string
	Run(ctx *Context) error
}

type Pipeline struct {
	Steps []Step
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
	for _, step := range p.Steps {
		ctx.Logger.Step(step.Name())

		if err := step.Run(ctx); err != nil {
			var coded codeCarrier
			if errors.As(err, &coded) {
				ctx.Logger.ErrorCode(coded.Code(), step.Name()+": "+err.Error())
			} else {
				ctx.Logger.Error(step.Name() + ": " + err.Error())
			}
			return err
		}

		ctx.Logger.Success(step.Name())
	}

	return nil
}
