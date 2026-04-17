package steps

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/gotcha190/toba/internal/create"
	"github.com/gotcha190/toba/internal/project"
)

type ProjectDirStep struct{}

// NewProjectDirStep creates the pipeline step that creates or reuses the
// project directory layout.
//
// Parameters:
// - none
//
// Returns:
// - a configured ProjectDirStep instance
func NewProjectDirStep() *ProjectDirStep {
	return &ProjectDirStep{}
}

// Name returns the human-readable pipeline label for this step.
//
// Parameters:
// - none
//
// Returns:
// - the display name used by pipeline logging
func (s *ProjectDirStep) Name() string {
	return "Project directory setup"
}

// Run creates the project directory tree or prepares an existing local backup
// folder for use as the project root.
//
// Parameters:
// - ctx: shared create context containing project paths and directory-mode flags
//
// Returns:
// - an error when directories already exist in an incompatible state or cannot be created
//
// Side effects:
// - creates project, app, and config directories
// - mutates ctx.ProjectCreated for freshly created roots
func (s *ProjectDirStep) Run(ctx *create.Context) error {
	if ctx.UseExistingProjectDir {
		return prepareExistingProjectDir(ctx)
	}

	exists, err := project.DirExists(ctx.Paths.Root)
	if err != nil {
		return err
	}
	if exists {
		return project.ErrDirExists{Path: ctx.Paths.Root}
	}

	dirs := []string{
		ctx.Paths.Root,
		ctx.Paths.AppDir,
		ctx.Paths.ConfigDir,
	}

	for _, dir := range dirs {
		if ctx.DryRun {
			ctx.Logger.Info("Would create: " + dir)
			continue
		}

		if err := project.CreateDir(dir); err != nil {
			return err
		}
		if dir == ctx.Paths.Root {
			ctx.ProjectCreated = true
		}
		ctx.Logger.Info("Created: " + dir)
	}

	return nil
}

// prepareExistingProjectDir validates an existing local backup folder and adds
// only the generated subdirectories required by the local project.
//
// Parameters:
// - ctx: shared create context containing the reused project root
//
// Returns:
// - an error when generated project markers already exist or the required directories cannot be created
//
// Side effects:
// - creates the `app` and `config` directories inside the reused project root
func prepareExistingProjectDir(ctx *create.Context) error {
	markers, err := existingProjectMarkers(ctx.Paths.Root)
	if err != nil {
		return err
	}
	if len(markers) > 0 {
		return fmt.Errorf("project directory already contains generated project markers: %s", strings.Join(markers, ", "))
	}

	dirs := []string{
		ctx.Paths.AppDir,
		ctx.Paths.ConfigDir,
	}

	for _, dir := range dirs {
		if ctx.DryRun {
			ctx.Logger.Info("Would create: " + dir)
			continue
		}

		if err := project.CreateDir(dir); err != nil {
			return err
		}
		ctx.Logger.Info("Created: " + dir)
	}

	return nil
}

// existingProjectMarkers reports generated project files or directories that
// would make reusing root unsafe.
//
// Parameters:
// - root: project root candidate to inspect
//
// Returns:
// - a list of conflicting generated markers
// - an error when marker inspection fails
func existingProjectMarkers(root string) ([]string, error) {
	checks := []string{
		".lando.yml",
		"app",
		"config",
	}

	var markers []string
	for _, name := range checks {
		target := filepath.Join(root, name)
		if _, err := os.Stat(target); err == nil {
			markers = append(markers, name)
			continue
		} else if !os.IsNotExist(err) {
			return nil, err
		}
	}

	return markers, nil
}
