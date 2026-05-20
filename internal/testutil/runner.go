package testutil

import (
	"strings"
	"sync"
)

type RecordedCommand struct {
	Dir  string
	Cmd  string
	Args []string
}

type RecordingRunner struct {
	mu                  sync.Mutex
	Commands            []RecordedCommand
	Err                 error
	RunErrByCommand     map[string]error
	CaptureErrByCommand map[string]error
	Outputs             map[string]string
	AfterRun            func(dir string, cmd string, args []string) error
}

func (r *RecordingRunner) Run(dir string, cmd string, args ...string) error {
	r.mu.Lock()
	r.Commands = append(r.Commands, RecordedCommand{
		Dir:  dir,
		Cmd:  cmd,
		Args: append([]string(nil), args...),
	})
	r.mu.Unlock()

	key := commandKey(cmd, args)
	if r.RunErrByCommand != nil {
		if err, ok := r.RunErrByCommand[key]; ok {
			return err
		}
	}
	if r.Err != nil {
		return r.Err
	}
	if r.AfterRun != nil {
		return r.AfterRun(dir, cmd, append([]string(nil), args...))
	}
	return nil
}

func (r *RecordingRunner) CaptureOutput(dir string, cmd string, args ...string) (string, error) {
	r.mu.Lock()
	r.Commands = append(r.Commands, RecordedCommand{
		Dir:  dir,
		Cmd:  cmd,
		Args: append([]string(nil), args...),
	})
	r.mu.Unlock()

	key := commandKey(cmd, args)
	if r.CaptureErrByCommand != nil {
		if err, ok := r.CaptureErrByCommand[key]; ok {
			return r.Outputs[key], err
		}
	}
	if r.Err != nil {
		return "", r.Err
	}
	return r.Outputs[key], nil
}

func CommandKey(cmd string, args ...string) string {
	return commandKey(cmd, args)
}

func commandKey(cmd string, args []string) string {
	return cmd + " " + strings.Join(args, " ")
}
