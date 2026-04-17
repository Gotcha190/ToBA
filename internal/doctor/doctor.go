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

// FullWorkflowChecks returns the external binaries required by the complete
// ToBA create workflow.
//
// Parameters:
// - none
//
// Returns:
// - the ordered list of dependency checks used by `toba doctor`
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

// RunChecks executes each binary check and returns one result per requested
// dependency.
//
// Parameters:
// - checks: dependency definitions that should be verified
//
// Returns:
// - one Result value per requested dependency
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

// checkBinary reports whether name is discoverable in PATH.
//
// Parameters:
// - name: binary name to resolve
//
// Returns:
// - nil when the binary is present
// - an error when the binary cannot be found
func checkBinary(name string) error {
	_, err := exec.LookPath(name)
	if err != nil {
		return errors.New(name + " is not installed or not in PATH")
	}
	return nil
}
