package create

import "errors"

type Step interface {
	Name() string
	Run(ctx *Context) error
}

type Pipeline struct {
	Steps []Step
}

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
