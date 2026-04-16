package steps

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/gotcha190/ToBA/internal/create"
	"github.com/gotcha190/ToBA/internal/project"
)

type ProjectDirStep struct{}

func NewProjectDirStep() *ProjectDirStep {
	return &ProjectDirStep{}
}

func (s *ProjectDirStep) Name() string {
	return "Project directory setup"
}

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
