package steps

import (
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
		ctx.Logger.Info("Created: " + dir)
	}

	return nil
}
