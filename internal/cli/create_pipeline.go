package cli

import (
	"os"
	"strings"

	"github.com/gotcha190/toba/internal/create"
	"github.com/gotcha190/toba/internal/create/steps"
)

// buildCreatePipeline constructs the create workflow dependency graph.
//
// Parameters:
// - baseDir: directory in which the project will be created
// - config: project configuration used to decide safe parallelization
// - sequential: whether to disable parallel scheduling for ready nodes
//
// Returns:
// - the configured create pipeline
func buildCreatePipeline(baseDir string, config create.ProjectConfig, sequential bool) create.Pipeline {
	remoteBootstrapParallel := canParallelizeRemoteBootstrap(baseDir, config)

	prepareStarterDataStep := create.Step(steps.NewPrepareStarterDataStep())
	prepareStarterDataDeps := []string{"collect-config"}
	projectDirDeps := []string{"prepare-starter-data"}
	cloneThemeDeps := []string{"project-dir", "prepare-starter-data"}
	importPluginsDeps := []string{"project-dir", "prepare-starter-data"}
	importUploadsDeps := []string{"project-dir", "prepare-starter-data"}
	importOthersDeps := []string{"project-dir", "prepare-starter-data"}
	importDatabaseDeps := []string{"install-wordpress", "prepare-starter-data", "clone-theme", "import-plugins", "import-others"}
	refreshThemeCachesDeps := []string{"generate-acorn-key", "clear-imported-caches", "flush-rewrite-rules", "import-others"}

	if remoteBootstrapParallel {
		prepareStarterDataStep = forcedRemotePrepareStarterDataStep{base: steps.NewPrepareStarterDataStep()}
		projectDirDeps = []string{"collect-config"}
		cloneThemeDeps = []string{"project-dir"}
	}

	nodes := []create.StepNode{
		{ID: "collect-config", Step: steps.NewCollectConfigStep()},
		{ID: "prepare-starter-data", Step: prepareStarterDataStep, DependsOn: prepareStarterDataDeps},
		{ID: "project-dir", Step: steps.NewProjectDirStep(), DependsOn: projectDirDeps},
		{ID: "generate-lando-config", Step: steps.NewGenerateLandoConfigStep(), DependsOn: []string{"project-dir"}},
		{ID: "start-lando", Step: steps.NewStartLandoStep(), DependsOn: []string{"generate-lando-config"}},
		{ID: "clone-theme", Step: steps.NewCloneStarterThemeStep(), DependsOn: cloneThemeDeps},
		{ID: "setup-theme-git", Step: steps.NewSetupThemeGitStep(), DependsOn: []string{"clone-theme"}},
		{ID: "import-plugins", Step: steps.NewImportPluginsStep(), DependsOn: importPluginsDeps},
	}
	if !config.NoUploads {
		nodes = append(nodes, create.StepNode{ID: "import-uploads", Step: steps.NewImportUploadsStep(), DependsOn: importUploadsDeps})
	}
	nodes = append(nodes,
		create.StepNode{ID: "import-others", Step: steps.NewImportOthersStep(), DependsOn: importOthersDeps},
		create.StepNode{ID: "install-wordpress", Step: steps.NewInstallWordPressStep(), DependsOn: []string{"start-lando"}},
		create.StepNode{ID: "build-theme", Step: steps.NewBuildThemeStep(), DependsOn: []string{"start-lando", "clone-theme", "setup-theme-git", "import-plugins"}},
		create.StepNode{ID: "import-database", Step: steps.NewImportDatabaseStep(), DependsOn: importDatabaseDeps},
		create.StepNode{ID: "reset-admin-password", Step: steps.NewResetAdminPasswordStep(), DependsOn: []string{"import-database"}},
		create.StepNode{ID: "activate-theme", Step: steps.NewActivateThemeStep(), DependsOn: []string{"import-database", "clone-theme", "build-theme"}},
		create.StepNode{ID: "generate-acorn-key", Step: steps.NewGenerateAcornKeyStep(), DependsOn: []string{"activate-theme"}},
		create.StepNode{ID: "clear-imported-caches", Step: steps.NewClearImportedCachesStep(), DependsOn: []string{"import-database"}},
		create.StepNode{ID: "flush-rewrite-rules", Step: steps.NewFlushRewriteRulesStep(), DependsOn: []string{"activate-theme"}},
	)
	if config.NoUploads {
		nodes = append(nodes, create.StepNode{ID: "configure-uploads-fallback", Step: steps.NewConfigureUploadsFallbackStep(), DependsOn: []string{"flush-rewrite-rules"}})
		refreshThemeCachesDeps = []string{"generate-acorn-key", "clear-imported-caches", "configure-uploads-fallback", "import-others"}
	}
	nodes = append(nodes, create.StepNode{ID: "refresh-theme-caches", Step: steps.NewRefreshThemeCachesStep(), DependsOn: refreshThemeCachesDeps})

	return create.Pipeline{
		Sequential: sequential,
		Nodes:      nodes,
	}
}

// canParallelizeRemoteBootstrap reports whether remote starter preparation can
// overlap with local project directory creation.
//
// Parameters:
// - baseDir: directory in which the project will be created
// - config: project configuration to inspect
//
// Returns:
// - true when remote starter data can be prepared in parallel with project setup
func canParallelizeRemoteBootstrap(baseDir string, config create.ProjectConfig) bool {
	normalized := config
	if err := normalized.Normalize(); err != nil {
		return false
	}

	if strings.TrimSpace(normalized.SSHTarget) == "" || strings.TrimSpace(normalized.RemoteWordPressRoot) == "" {
		return false
	}

	paths := create.NewProjectPaths(baseDir, normalized.Name)
	info, err := os.Stat(paths.Root)
	if err == nil {
		return !info.IsDir()
	}

	return os.IsNotExist(err)
}

type forcedRemotePrepareStarterDataStep struct {
	base create.Step
}

// Name returns the wrapped prepare starter data step name.
//
// Returns:
// - the step name
func (s forcedRemotePrepareStarterDataStep) Name() string {
	return s.base.Name()
}

// Run forces remote starter data mode before running the wrapped step.
//
// Parameters:
// - ctx: shared create workflow context
//
// Returns:
// - an error when the wrapped step fails
//
// Side effects:
// - updates ctx.StarterData.Mode before delegating execution
func (s forcedRemotePrepareStarterDataStep) Run(ctx *create.Context) error {
	ctx.StarterData.Mode = "remote"
	return s.base.Run(ctx)
}
