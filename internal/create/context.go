package create

type Context struct {
	Config ProjectConfig
	DryRun bool
	Logger Logger
	Runner CommandRunner
	Paths  ProjectPaths

	ProjectCreated bool
}
