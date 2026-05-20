package cli

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/gotcha190/toba/internal/create"
)

func TestBuildCreatePipelineOverlapsRemoteBootstrapWhenProjectDirDoesNotExist(t *testing.T) {
	baseDir := t.TempDir()

	pipeline := buildCreatePipeline(baseDir, create.ProjectConfig{
		Name:                "demo",
		PHPVersion:          "8.4",
		StarterRepo:         testStarterRepo,
		SSHTarget:           "user@192.168.0.1 -p 22",
		RemoteWordPressRoot: testRemoteWordPressRoot,
	}, false)

	assertNodeDependsOn(t, pipeline.Nodes, "project-dir", []string{"collect-config"})
	assertNodeDependsOn(t, pipeline.Nodes, "clone-theme", []string{"project-dir"})
	assertNodeDependsOn(t, pipeline.Nodes, "setup-theme-git", []string{"clone-theme"})
	assertNodeDependsOn(t, pipeline.Nodes, "import-plugins", []string{"project-dir", "prepare-starter-data"})
	assertNodeDependsOn(t, pipeline.Nodes, "build-theme", []string{"start-lando", "clone-theme", "import-plugins"})
	assertNodeDependsOn(t, pipeline.Nodes, "import-database", []string{"install-wordpress", "prepare-starter-data", "clone-theme", "import-plugins", "import-others"})
}

func TestBuildCreatePipelineKeepsPrepareStarterDataAheadOfProjectDirForExistingProject(t *testing.T) {
	baseDir := t.TempDir()
	if err := os.Mkdir(filepath.Join(baseDir, "demo"), 0755); err != nil {
		t.Fatalf("failed to prepare existing project dir: %v", err)
	}

	pipeline := buildCreatePipeline(baseDir, create.ProjectConfig{
		Name:                "demo",
		PHPVersion:          "8.4",
		StarterRepo:         testStarterRepo,
		SSHTarget:           "user@192.168.0.1 -p 22",
		RemoteWordPressRoot: testRemoteWordPressRoot,
	}, false)

	assertNodeDependsOn(t, pipeline.Nodes, "project-dir", []string{"prepare-starter-data"})
	assertNodeDependsOn(t, pipeline.Nodes, "clone-theme", []string{"project-dir", "prepare-starter-data"})
	assertNodeDependsOn(t, pipeline.Nodes, "setup-theme-git", []string{"clone-theme"})
	assertNodeDependsOn(t, pipeline.Nodes, "build-theme", []string{"start-lando", "clone-theme", "import-plugins"})
}

func TestBuildCreatePipelineSetsSequentialModeWhenRequested(t *testing.T) {
	baseDir := t.TempDir()

	pipeline := buildCreatePipeline(baseDir, create.ProjectConfig{
		Name:                "demo",
		PHPVersion:          "8.4",
		StarterRepo:         testStarterRepo,
		SSHTarget:           "user@192.168.0.1 -p 22",
		RemoteWordPressRoot: testRemoteWordPressRoot,
	}, true)

	if !pipeline.Sequential {
		t.Fatal("expected sequential pipeline mode to be enabled")
	}
}

func TestBuildCreatePipelineLeavesSequentialDisabledByDefault(t *testing.T) {
	baseDir := t.TempDir()

	pipeline := buildCreatePipeline(baseDir, create.ProjectConfig{
		Name:                "demo",
		PHPVersion:          "8.4",
		StarterRepo:         testStarterRepo,
		SSHTarget:           "user@192.168.0.1 -p 22",
		RemoteWordPressRoot: testRemoteWordPressRoot,
	}, false)

	if pipeline.Sequential {
		t.Fatal("expected sequential pipeline mode to be disabled")
	}
}

func TestBuildCreatePipelineNoUploadsSkipsImportAndAddsFallback(t *testing.T) {
	baseDir := t.TempDir()

	pipeline := buildCreatePipeline(baseDir, create.ProjectConfig{
		Name:                "demo",
		PHPVersion:          "8.4",
		StarterRepo:         testStarterRepo,
		SSHTarget:           "user@192.168.0.1 -p 22",
		RemoteWordPressRoot: testRemoteWordPressRoot,
		NoUploads:           true,
	}, false)

	assertNoNode(t, pipeline.Nodes, "import-uploads")
	assertNodeDependsOn(t, pipeline.Nodes, "configure-uploads-fallback", []string{"flush-rewrite-rules"})
	assertNodeDependsOn(t, pipeline.Nodes, "refresh-theme-caches", []string{"generate-acorn-key", "clear-imported-caches", "configure-uploads-fallback", "import-others"})
	assertNodeDependsOn(t, pipeline.Nodes, "import-database", []string{"install-wordpress", "prepare-starter-data", "clone-theme", "import-plugins", "import-others"})
}
