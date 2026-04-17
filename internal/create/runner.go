package create

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type CommandRunner interface {
	Run(dir string, cmd string, args ...string) error
	CaptureOutput(dir string, cmd string, args ...string) (string, error)
}

type ExecRunner struct{}

// Run executes a shell command inside dir while streaming stdout and stderr to
// the current process.
//
// Parameters:
// - dir: working directory in which the command should run
// - cmd: program name to execute
// - args: command-line arguments passed to cmd
//
// Returns:
// - an error when the command exits unsuccessfully
//
// Side effects:
// - launches a child process
// - writes command output to the current process streams
func (r ExecRunner) Run(dir string, cmd string, args ...string) error {
	command := shellCommand(dir, cmd, args...)
	command.Env = withWorkingDirEnv(dir)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	command.Stdin = os.Stdin
	command.Stdout = io.MultiWriter(os.Stdout, &stdout)
	command.Stderr = io.MultiWriter(os.Stderr, &stderr)

	if err := command.Run(); err != nil {
		return formatCommandError(dir, cmd, args, err, stdout.String(), stderr.String())
	}

	return nil
}

// CaptureOutput executes a shell command inside dir and returns its output
// without streaming it to the console.
//
// Parameters:
// - dir: working directory in which the command should run
// - cmd: program name to execute
// - args: command-line arguments passed to cmd
//
// Returns:
// - the captured stdout output on success, or stderr/stdout content on failure
// - an error when the command exits unsuccessfully
func (r ExecRunner) CaptureOutput(dir string, cmd string, args ...string) (string, error) {
	command := shellCommand(dir, cmd, args...)
	command.Env = withWorkingDirEnv(dir)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	command.Stdout = &stdout
	command.Stderr = &stderr

	err := command.Run()
	if err != nil {
		if stderr.Len() > 0 {
			return stderr.String(), err
		}
		return stdout.String(), err
	}

	return stdout.String(), nil
}

// formatCommandError builds a command failure message that includes the
// working directory and captured command output when available.
//
// Parameters:
// - dir: working directory used for the command
// - cmd: program name
// - args: command-line arguments
// - err: original command execution error
// - stdout: captured standard output
// - stderr: captured standard error
//
// Returns:
// - a formatted error describing the failed command
func formatCommandError(dir string, cmd string, args []string, err error, stdout string, stderr string) error {
	commandLine := strings.Join(append([]string{cmd}, args...), " ")
	output := strings.TrimSpace(stderr)
	if output == "" {
		output = strings.TrimSpace(stdout)
	}

	if output == "" {
		return fmt.Errorf("command %q failed in %s: %w", commandLine, dir, err)
	}

	return fmt.Errorf("command %q failed in %s: %w\n%s", commandLine, dir, err, output)
}

// withWorkingDirEnv ensures the spawned process sees dir as its PWD value.
//
// Parameters:
// - dir: working directory to expose through the PWD environment variable
//
// Returns:
// - a copy of the current environment with PWD set to dir
func withWorkingDirEnv(dir string) []string {
	env := os.Environ()
	pwdPrefix := "PWD="

	for i, entry := range env {
		if strings.HasPrefix(entry, pwdPrefix) {
			env[i] = pwdPrefix + dir
			return env
		}
	}

	return append(env, pwdPrefix+dir)
}

// shellCommand builds a bash command that runs cmd with args inside dir.
//
// Parameters:
// - dir: working directory for the command
// - cmd: program name
// - args: program arguments
//
// Returns:
// - an exec.Cmd configured to run the generated shell script
func shellCommand(dir string, cmd string, args ...string) *exec.Cmd {
	script := buildShellScript(dir, cmd, args...)
	return exec.Command("bash", "-lc", script)
}

// buildShellScript assembles the shell script used by ExecRunner, including an
// initial directory change when dir is set.
//
// Parameters:
// - dir: optional working directory
// - cmd: program name
// - args: command arguments
//
// Returns:
// - a shell-safe script string executed by bash -lc
func buildShellScript(dir string, cmd string, args ...string) string {
	parts := make([]string, 0, len(args)+1)
	parts = append(parts, shellQuote(cmd))
	for _, arg := range args {
		parts = append(parts, shellQuote(arg))
	}

	if dir == "" {
		return strings.Join(parts, " ")
	}

	return "cd " + shellQuote(filepath.Clean(dir)) + " && " + strings.Join(parts, " ")
}

// shellQuote escapes a string for safe inclusion in a single-quoted shell
// script fragment.
//
// Parameters:
// - value: raw string that will be embedded in the shell script
//
// Returns:
// - a safely quoted shell fragment
func shellQuote(value string) string {
	return "'" + strings.ReplaceAll(value, "'", `'\''`) + "'"
}

type NoopRunner struct{}

// Run ignores the requested command and reports success.
//
// Parameters:
// - dir: ignored
// - cmd: ignored
// - args: ignored
//
// Returns:
// - always nil
func (r NoopRunner) Run(dir string, cmd string, args ...string) error {
	return nil
}

// CaptureOutput ignores the requested command and returns an empty result.
//
// Parameters:
// - dir: ignored
// - cmd: ignored
// - args: ignored
//
// Returns:
// - an empty string
// - always nil
func (r NoopRunner) CaptureOutput(dir string, cmd string, args ...string) (string, error) {
	return "", nil
}
