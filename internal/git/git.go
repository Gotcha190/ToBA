package git

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/gotcha190/toba/internal/create"
)

type MissingStarterRepoError struct{}

type ProjectBranchSetupResult struct {
	RepoURL  string
	Pushed   bool
	Warnings []string
}

// Error explains how to provide the missing starter repository setting.
//
// Returns:
// - the human-readable error string
func (e MissingStarterRepoError) Error() string {
	return "starter repo is not configured; add TOBA_STARTER_REPO to ~/.config/toba/.env via 'toba config' or pass --starter-repo and try again"
}

// Clone clones repo into parentDir/name.
//
// Parameters:
// - runner: command runner used to launch git
// - parentDir: directory where the clone target should be created
// - repo: git repository to clone
// - name: destination directory name
//
// Returns:
// - the clone target directory
// - an error when the repository is missing, target exists, or clone fails
func Clone(runner create.CommandRunner, parentDir string, repo string, name string) (string, error) {
	if strings.TrimSpace(repo) == "" {
		return "", MissingStarterRepoError{}
	}

	if err := os.MkdirAll(parentDir, 0755); err != nil {
		return "", err
	}

	targetDir := filepath.Join(parentDir, name)
	if _, err := os.Stat(targetDir); err == nil {
		return "", create.NewCodedError("THEME_DIR_EXISTS", "theme directory already exists: "+targetDir, nil)
	} else if !os.IsNotExist(err) {
		return "", err
	}

	if err := runner.Run(parentDir, "git", "clone", repo, name); err != nil {
		return "", create.NewCodedError("THEME_CLONE_FAILED", "starter theme clone failed", err)
	}

	return targetDir, nil
}

// DetachRemoteAndCreateBranches removes the cloned remote and prepares local
// develop and starter branches at the same commit.
//
// Parameters:
// - runner: command runner used to launch git
// - repoDir: cloned repository directory
//
// Returns:
// - an error when any git setup command fails
func DetachRemoteAndCreateBranches(runner create.CommandRunner, repoDir string) error {
	commands := [][]string{
		{"remote", "remove", "origin"},
		{"branch", "-M", "develop"},
		{"branch", "-f", "starter", "develop"},
	}

	for _, args := range commands {
		if err := runner.Run(repoDir, "git", args...); err != nil {
			return create.NewCodedError("THEME_GIT_SETUP_FAILED", "starter theme git setup failed", err)
		}
	}

	return nil
}

// ProjectRepoURLFromStarter derives the project repository URL from the
// configured starter repository URL and project name.
//
// Parameters:
// - starterRepo: starter repository URL in supported GitHub SSH or HTTPS format
// - projectName: destination project repository name
//
// Returns:
// - the derived project repository URL
// - an error when starterRepo or projectName cannot be used
func ProjectRepoURLFromStarter(starterRepo string, projectName string) (string, error) {
	starterRepo = strings.TrimSpace(starterRepo)
	projectName = strings.TrimSpace(projectName)
	if starterRepo == "" {
		return "", fmt.Errorf("starter repo is empty")
	}
	if projectName == "" {
		return "", fmt.Errorf("project name is empty")
	}

	sshPattern := regexp.MustCompile(`^(git@[^:]+:[^/]+)/[^/]+\.git$`)
	if matches := sshPattern.FindStringSubmatch(starterRepo); len(matches) == 2 {
		return matches[1] + "/" + projectName + ".git", nil
	}

	httpsPattern := regexp.MustCompile(`^https://github\.com/([^/]+)/[^/]+\.git$`)
	if matches := httpsPattern.FindStringSubmatch(starterRepo); len(matches) == 2 {
		return "https://github.com/" + matches[1] + "/" + projectName + ".git", nil
	}

	return "", fmt.Errorf("unsupported starter repo URL format: %s", starterRepo)
}

// TrySetupProjectBranches prepares local develop/starter branches and, when
// available, pushes them to the project repository derived from starterRepo.
//
// Parameters:
// - runner: command runner used to launch git
// - repoDir: cloned repository directory
// - starterRepo: starter repository URL used to derive the project repository
// - projectName: destination project repository name
//
// Returns:
// - setup result containing the derived repository URL, push state, and warnings
func TrySetupProjectBranches(runner create.CommandRunner, repoDir string, starterRepo string, projectName string) ProjectBranchSetupResult {
	result := ProjectBranchSetupResult{}

	if err := runner.Run(repoDir, "git", "remote", "remove", "origin"); err != nil {
		result.Warnings = append(result.Warnings, "Could not remove starter git remote: "+err.Error())
	}
	if err := runner.Run(repoDir, "git", "branch", "-M", "develop"); err != nil {
		result.Warnings = append(result.Warnings, "Could not create develop branch: "+err.Error())
	}
	if err := runner.Run(repoDir, "git", "branch", "-f", "starter", "develop"); err != nil {
		result.Warnings = append(result.Warnings, "Could not create starter branch: "+err.Error())
	}

	repoURL, err := ProjectRepoURLFromStarter(starterRepo, projectName)
	if err != nil {
		result.Warnings = append(result.Warnings, "Could not derive project git repo from starter repo; skipping branch push: "+err.Error())
		return result
	}
	result.RepoURL = repoURL

	if err := runner.Run(repoDir, "git", "ls-remote", repoURL); err != nil {
		result.Warnings = append(result.Warnings, "Project git repo is not available yet; skipping branch push: "+repoURL)
		return result
	}
	if err := runner.Run(repoDir, "git", "remote", "add", "origin", repoURL); err != nil {
		result.Warnings = append(result.Warnings, "Could not add project git remote: "+err.Error())
		return result
	}
	if err := runner.Run(repoDir, "git", "push", "-u", "origin", "develop", "starter"); err != nil {
		result.Warnings = append(result.Warnings, "Could not push develop and starter branches to "+repoURL+": "+err.Error())
		return result
	}

	result.Pushed = true
	return result
}
