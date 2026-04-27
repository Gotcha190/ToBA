package sourcedata

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/gotcha190/toba/internal/create"
)

var sshTargetPattern = regexp.MustCompile(`^([^\s@]+@[^\s]+)\s+-p\s+([0-9]+)$`)

type sshTarget struct {
	UserHost string
	Port     string
}

// parseSSHTarget validates the configured SSH target string and splits it into user-host and port components.
//
// Returns:
// - the parsed sshTarget structure
// - an error when the target does not match the expected `user@host -p port` format
func parseSSHTarget(raw string) (sshTarget, error) {
	trimmed := strings.TrimSpace(raw)
	match := sshTargetPattern.FindStringSubmatch(trimmed)
	if match == nil {
		return sshTarget{}, fmt.Errorf("invalid TOBA_SSH_TARGET %q; expected format: user@host -p port (example: user@192.168.0.1 -p 22)", raw)
	}

	if _, err := strconv.Atoi(match[2]); err != nil {
		return sshTarget{}, fmt.Errorf("invalid TOBA_SSH_TARGET %q; expected numeric port in format: user@host -p port", raw)
	}

	return sshTarget{
		UserHost: match[1],
		Port:     match[2],
	}, nil
}

// runSSHCommand executes a remote script through SSH.
//
// Returns:
// - an error when the SSH command fails
func runSSHCommand(ctx *create.Context, target sshTarget, remoteDir string, script string) error {
	if err := ctx.Runner.Run("", "ssh", "-p", target.Port, target.UserHost, remoteScript(remoteDir, script)); err != nil {
		return fmt.Errorf("SSH command failed on %s:%s: %w", target.UserHost, target.Port, err)
	}

	return nil
}

// captureSSHCommand executes a remote script through SSH and returns trimmed stdout.
//
// Returns:
// - the trimmed stdout output
// - an error when the SSH command fails
func captureSSHCommand(ctx *create.Context, target sshTarget, remoteDir string, script string) (string, error) {
	output, err := ctx.Runner.CaptureOutput("", "ssh", "-p", target.Port, target.UserHost, remoteScript(remoteDir, script))
	if err != nil {
		return "", formatSSHCommandError(target, err, output)
	}

	return strings.TrimSpace(output), nil
}

// formatSSHCommandError builds an SSH failure message with optional captured output.
//
// Returns:
// - the formatted SSH error
func formatSSHCommandError(target sshTarget, err error, output string) error {
	detail := strings.TrimSpace(output)
	if detail == "" {
		return fmt.Errorf("SSH command failed on %s:%s: %w", target.UserHost, target.Port, err)
	}

	return fmt.Errorf("SSH command failed on %s:%s: %w\n%s", target.UserHost, target.Port, err, detail)
}

// copyRemoteFile downloads one file from the starter host into localPath.
//
// Returns:
// - an error when the local directory cannot be created or the download fails
//
// Side effects:
// - creates the local parent directory when needed
// - removes the partial local file if scp fails
func copyRemoteFile(ctx *create.Context, target sshTarget, remotePath string, localPath string) error {
	if err := os.MkdirAll(filepath.Dir(localPath), 0755); err != nil {
		return err
	}

	if err := ctx.Runner.Run("", "scp", "-P", target.Port, target.UserHost+":"+remotePath, localPath); err != nil {
		_ = os.Remove(localPath)
		return fmt.Errorf("failed to download starter file from %s:%s: %w", target.UserHost, target.Port, err)
	}

	return nil
}

// remoteScript prepends an optional remote directory change before the given shell fragment.
//
// Returns:
// - the combined remote shell command string
func remoteScript(remoteDir string, script string) string {
	if strings.TrimSpace(remoteDir) == "" {
		return script
	}

	return "cd " + shellQuote(remoteDir) + " && " + script
}

// shellQuote escapes a string for safe use inside a single-quoted shell fragment.
//
// Returns:
// - the safely quoted shell string
func shellQuote(value string) string {
	return "'" + strings.ReplaceAll(value, "'", `'\''`) + "'"
}
