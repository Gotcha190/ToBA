package doctor

import (
	"errors"
	"os/exec"
)

type Check struct {
	Name   string
	Binary string
}

func FullWorkflowChecks() []Check {
	return []Check{
		{Name: "Git", Binary: "git"},
		{Name: "Node", Binary: "node"},
		{Name: "NPM", Binary: "npm"},
		{Name: "Composer", Binary: "composer"},
		{Name: "Lando", Binary: "lando"},
		{Name: "Docker", Binary: "docker"},
		{Name: "SSH", Binary: "ssh"},
		{Name: "SCP", Binary: "scp"},
		{Name: "Zip", Binary: "zip"},
	}
}

func RunCheck(check Check) error {
	return checkBinary(check.Binary)
}

func CheckGit() error {
	return checkBinary("git")
}

func CheckNode() error {
	return checkBinary("node")
}

func CheckNpm() error {
	return checkBinary("npm")
}

func CheckComposer() error {
	return checkBinary("composer")
}

func CheckLando() error {
	return checkBinary("lando")
}

func CheckDocker() error {
	return checkBinary("docker")
}

// checkBinary sprawdza czy dany program jest w PATH
func checkBinary(name string) error {
	_, err := exec.LookPath(name)
	if err != nil {
		return errors.New(name + " is not installed or not in PATH")
	}
	return nil
}
