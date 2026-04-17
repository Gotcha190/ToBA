package doctor

import (
	"errors"
	"os/exec"
)

type Check struct {
	Name   string
	Binary string
}

type Result struct {
	Check Check
	Err   error
}

func FullWorkflowChecks() []Check {
	return []Check{
		{Name: "Git", Binary: "git"},
		{Name: "Node", Binary: "node"},
		{Name: "NPM", Binary: "npm"},
		{Name: "Lando", Binary: "lando"},
		{Name: "Docker", Binary: "docker"},
		{Name: "SSH", Binary: "ssh"},
		{Name: "SCP", Binary: "scp"},
		{Name: "Zip", Binary: "zip"},
	}
}

func RunChecks(checks []Check) []Result {
	results := make([]Result, 0, len(checks))
	for _, check := range checks {
		results = append(results, Result{
			Check: check,
			Err:   checkBinary(check.Binary),
		})
	}
	return results
}

func checkBinary(name string) error {
	_, err := exec.LookPath(name)
	if err != nil {
		return errors.New(name + " is not installed or not in PATH")
	}
	return nil
}
