package steps

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/gotcha190/toba/internal/create"
)

const (
	remoteWordPressRoot = "www/toba.tamago-dev.pl"
)

var (
	sshTargetPattern = regexp.MustCompile(`^([^\s@]+@[^\s]+)\s+-p\s+([0-9]+)$`)
)

type sshTarget struct {
	UserHost string
	Port     string
}

// parseSSHTarget validates the configured SSH target string and splits it into
// user-host and port components.
//
// Parameters:
// - raw: raw TOBA_SSH_TARGET value
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

// runSSHCommand executes a remote script through ssh and wraps connection
// failures with starter-host context.
//
// Parameters:
// - ctx: shared create context providing the command runner
// - target: parsed SSH target
// - remoteDir: remote directory in which the script should run
// - script: shell fragment executed on the remote host
//
// Returns:
// - an error when the SSH command fails
//
// Side effects:
// - runs `ssh` against the configured starter host
func runSSHCommand(ctx *create.Context, target sshTarget, remoteDir string, script string) error {
	if err := ctx.Runner.Run("", "ssh", "-p", target.Port, target.UserHost, remoteScript(remoteDir, script)); err != nil {
		return fmt.Errorf("SSH command failed on %s:%s: %w", target.UserHost, target.Port, err)
	}

	return nil
}

// captureSSHCommand executes a remote script through ssh and returns its
// trimmed stdout output.
//
// Parameters:
// - ctx: shared create context providing the command runner
// - target: parsed SSH target
// - remoteDir: remote directory in which the script should run
// - script: shell fragment executed on the remote host
//
// Returns:
// - the trimmed stdout output
// - an error when the SSH command fails
func captureSSHCommand(ctx *create.Context, target sshTarget, remoteDir string, script string) (string, error) {
	output, err := ctx.Runner.CaptureOutput("", "ssh", "-p", target.Port, target.UserHost, remoteScript(remoteDir, script))
	if err != nil {
		return "", fmt.Errorf("failed to connect to SSH starter host %s:%s: %w", target.UserHost, target.Port, err)
	}

	return strings.TrimSpace(output), nil
}

// copyRemoteFile downloads one file from the starter host into localPath.
//
// Parameters:
// - ctx: shared create context providing the command runner
// - target: parsed SSH target
// - remotePath: file path on the remote host
// - localPath: destination path on the local machine
//
// Returns:
// - an error when the parent directory cannot be created or the download fails
//
// Side effects:
// - creates the local parent directory when needed
// - runs `scp` against the configured starter host
func copyRemoteFile(ctx *create.Context, target sshTarget, remotePath string, localPath string) error {
	if err := os.MkdirAll(filepath.Dir(localPath), 0755); err != nil {
		return err
	}

	if err := ctx.Runner.Run("", "scp", "-P", target.Port, target.UserHost+":"+remotePath, localPath); err != nil {
		return fmt.Errorf("failed to download starter file from %s:%s: %w", target.UserHost, target.Port, err)
	}

	return nil
}

// remoteScript builds the remote shell snippet that first changes into
// remoteDir before running script.
//
// Parameters:
// - remoteDir: remote working directory
// - script: remote shell fragment to execute
//
// Returns:
// - a combined remote shell command string
func remoteScript(remoteDir string, script string) string {
	return "cd " + shellQuote(remoteDir) + " && " + script
}

// shellQuote escapes a string for safe use inside a single-quoted remote shell
// command.
//
// Parameters:
// - value: raw string to quote
//
// Returns:
// - a safely quoted shell fragment
func shellQuote(value string) string {
	return "'" + strings.ReplaceAll(value, "'", `'\''`) + "'"
}
