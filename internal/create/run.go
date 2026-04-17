package create

import "os"

func NewContext(baseDir string, config ProjectConfig, logger Logger, runner CommandRunner) *Context {
	if logger == nil {
		logger = NewConsoleLogger(os.Stdout)
	}
	if runner == nil {
		runner = ExecRunner{}
	}

	return &Context{
		Config: config,
		DryRun: config.DryRun,
		Logger: logger,
		Runner: runner,
		Paths:  NewProjectPaths(baseDir, config.Name),
	}
}
