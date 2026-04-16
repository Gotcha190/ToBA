package create

type Context struct {
	Config ProjectConfig
	DryRun bool
	Logger Logger
	Runner CommandRunner
	Paths  ProjectPaths

	StarterData StarterData

	ProjectCreated bool
}

type StarterData struct {
	Mode         string
	TempDir      string
	DatabasePath string
	PluginsPaths []string
	UploadsPaths []string
	OthersPaths  []string
	ThemePaths   []string
	SourceURL    string
}
