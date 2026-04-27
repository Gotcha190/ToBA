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
//
// Returns:
// - the configured create pipeline
func buildCreatePipeline(baseDir string, config create.ProjectConfig) create.Pipeline {
	remoteBootstrapParallel := canParallelizeRemoteBootstrap(baseDir, config)

	prepareStarterDataStep := create.Step(steps.NewPrepareStarterDataStep())
	prepareStarterDataDeps := []string{"collect-config"}
	projectDirDeps := []string{"prepare-starter-data"}
	installThemeDeps := []string{"project-dir", "prepare-starter-data"}
	importPluginsDeps := []string{"project-dir", "prepare-starter-data"}
	importUploadsDeps := []string{"project-dir", "prepare-starter-data"}
	importOthersDeps := []string{"project-dir", "prepare-starter-data"}
	importDatabaseDeps := []string{"install-wordpress", "prepare-starter-data", "install-theme", "import-plugins", "import-others"}

	if remoteBootstrapParallel {
		prepareStarterDataStep = forcedRemotePrepareStarterDataStep{base: steps.NewPrepareStarterDataStep()}
		projectDirDeps = []string{"collect-config"}
		installThemeDeps = []string{"project-dir"}
	}

	return create.Pipeline{
		Nodes: []create.StepNode{
			{ID: "collect-config", Step: steps.NewCollectConfigStep()},
			{ID: "prepare-starter-data", Step: prepareStarterDataStep, DependsOn: prepareStarterDataDeps},
			{ID: "project-dir", Step: steps.NewProjectDirStep(), DependsOn: projectDirDeps},
			{ID: "generate-lando-config", Step: steps.NewGenerateLandoConfigStep(), DependsOn: []string{"project-dir"}},
			{ID: "start-lando", Step: steps.NewStartLandoStep(), DependsOn: []string{"generate-lando-config"}},
			{ID: "install-theme", Step: steps.NewInstallThemeStep(), DependsOn: installThemeDeps},
			{ID: "import-plugins", Step: steps.NewImportPluginsStep(), DependsOn: importPluginsDeps},
			{ID: "import-uploads", Step: steps.NewImportUploadsStep(), DependsOn: importUploadsDeps},
			{ID: "import-others", Step: steps.NewImportOthersStep(), DependsOn: importOthersDeps},
			{ID: "install-wordpress", Step: steps.NewInstallWordPressStep(), DependsOn: []string{"start-lando"}},
			{ID: "build-theme", Step: steps.NewBuildThemeStep(), DependsOn: []string{"start-lando", "install-theme", "import-plugins"}},
			{ID: "import-database", Step: steps.NewImportDatabaseStep(), DependsOn: importDatabaseDeps},
			{ID: "reset-admin-password", Step: steps.NewResetAdminPasswordStep(), DependsOn: []string{"import-database"}},
			{ID: "activate-theme", Step: steps.NewActivateThemeStep(), DependsOn: []string{"import-database", "install-theme", "build-theme"}},
			{ID: "generate-acorn-key", Step: steps.NewGenerateAcornKeyStep(), DependsOn: []string{"activate-theme"}},
			{ID: "clear-imported-caches", Step: steps.NewClearImportedCachesStep(), DependsOn: []string{"import-database"}},
			{ID: "flush-rewrite-rules", Step: steps.NewFlushRewriteRulesStep(), DependsOn: []string{"activate-theme"}},
			{ID: "refresh-theme-caches", Step: steps.NewRefreshThemeCachesStep(), DependsOn: []string{"generate-acorn-key", "clear-imported-caches", "flush-rewrite-rules", "import-others"}},
		},
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
