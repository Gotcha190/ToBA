package create

import "os"

// NewContext constructs a create workflow context with default logger and
// runner implementations when nil values are provided.
//
// Parameters:
// - baseDir: base directory from which project paths should be derived
// - config: project configuration for the current run
// - logger: optional logger implementation
// - runner: optional command runner implementation
//
// Returns:
// - a fully initialized workflow context
//
// Side effects:
// - substitutes default logger and runner implementations when nil values are supplied
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
