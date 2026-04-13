package create

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
)

type CommandRunner interface {
	Run(dir string, cmd string, args ...string) error
	CaptureOutput(dir string, cmd string, args ...string) (string, error)
}

type ExecRunner struct{}

func (r ExecRunner) Run(dir string, cmd string, args ...string) error {
	command := exec.Command(cmd, args...)
	command.Dir = dir
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

func (r ExecRunner) CaptureOutput(dir string, cmd string, args ...string) (string, error) {
	command := exec.Command(cmd, args...)
	command.Dir = dir
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

type NoopRunner struct{}

func (r NoopRunner) Run(dir string, cmd string, args ...string) error {
	return nil
}

func (r NoopRunner) CaptureOutput(dir string, cmd string, args ...string) (string, error) {
	return "", nil
}
