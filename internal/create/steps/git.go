package steps

import (
	"path/filepath"

	"github.com/gotcha190/toba/internal/create"
	"github.com/gotcha190/toba/internal/git"
)

type SetupThemeGitStep struct{}

// NewSetupThemeGitStep creates the pipeline step that disconnects the cloned
// starter theme from its remote, creates local branches, and best-effort pushes
// them to the derived project repository.
//
// Returns:
// - a configured SetupThemeGitStep instance
func NewSetupThemeGitStep() *SetupThemeGitStep {
	return &SetupThemeGitStep{}
}

// Name returns the human-readable pipeline label for this step.
//
// Returns:
// - the display name used by pipeline logging
func (s *SetupThemeGitStep) Name() string {
	return "Configure starter theme git"
}

// Run removes the starter theme remote, creates develop/starter branches, and
// optionally pushes them to the project repository.
//
// Parameters:
// - ctx: shared create context containing starter data, paths, and runner access
//
// Returns:
// - nil; git setup and push are intentionally non-critical
//
// Side effects:
// - may run git remote, branch, ls-remote, and push commands inside the cloned theme repository
// - logs planned actions instead of mutating the project in dry-run mode
func (s *SetupThemeGitStep) Run(ctx *create.Context) error {
	if len(ctx.StarterData.ThemePaths) > 0 {
		return nil
	}

	themeDir := filepath.Join(ctx.Paths.Themes, ctx.Config.Name)
	if ctx.DryRun {
		ctx.Logger.Info("Would run in " + themeDir + ": git remote remove origin")
		ctx.Logger.Info("Would run in " + themeDir + ": git branch -M develop")
		ctx.Logger.Info("Would run in " + themeDir + ": git branch -f starter develop")
		repoURL, err := git.ProjectRepoURLFromStarter(ctx.Config.StarterRepo, ctx.Config.Name)
		if err != nil {
			ctx.Logger.Warning("Could not derive project git repo from starter repo; skipping branch push")
			return nil
		}
		ctx.Logger.Info("Would check project repo derived from starter repo: " + repoURL)
		ctx.Logger.Info("Would add origin and push develop/starter if project repo exists")
		return nil
	}

	result := git.TrySetupProjectBranches(ctx.Runner, themeDir, ctx.Config.StarterRepo, ctx.Config.Name)
	for _, warning := range result.Warnings {
		ctx.Logger.Warning(warning)
	}
	if result.Pushed {
		ctx.Logger.Info("Pushed develop and starter branches to " + result.RepoURL)
	}

	return nil
}
