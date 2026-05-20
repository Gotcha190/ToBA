package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gotcha190/toba/internal/create"
)

func TestRunCreateUsesEnvConfig(t *testing.T) {
	baseDir := t.TempDir()
	withWorkingDir(t, baseDir)
	runner := &fakeRunner{}

	globalEnvPath, err := create.GlobalEnvPath()
	if err != nil {
		t.Fatalf("GlobalEnvPath returned error: %v", err)
	}

	env := "" +
		"TOBA_PHP_VERSION=8.4\n" +
		"TOBA_DOMAIN=stale-from-env.lndo.site\n" +
		"TOBA_STARTER_REPO=" + testStarterRepo + "\n" +
		"TOBA_SSH_TARGET=user@192.168.0.1 -p 22\n" +
		"TOBA_REMOTE_WORDPRESS_ROOT=" + testRemoteWordPressRoot + "\n"
	if err := os.MkdirAll(filepath.Dir(globalEnvPath), 0755); err != nil {
		t.Fatalf("failed to create config dir: %v", err)
	}
	if err := os.WriteFile(globalEnvPath, []byte(env), 0644); err != nil {
		t.Fatalf("failed to write global env file: %v", err)
	}

	if err := runCreateWithRunner(CreateOptions{Name: "demo"}, runner); err != nil {
		t.Fatalf("runCreateWithRunner returned error: %v", err)
	}

	if len(runner.commands) == 0 {
		t.Fatal("expected commands to be executed from env config")
	}
	var installURL string
	for _, command := range runner.commands {
		if command.cmd != "lando" || len(command.args) < 4 {
			continue
		}
		if command.args[0] == "wp" && command.args[1] == "core" && command.args[2] == "install" {
			installURL = command.args[3]
			break
		}
	}
	if installURL != "--url=demo.lndo.site" {
		t.Fatalf("expected derived domain --url=demo.lndo.site, got %q", installURL)
	}
}

func TestRunCreateWritesConfigSourceToProvidedOutput(t *testing.T) {
	baseDir := t.TempDir()
	withWorkingDir(t, baseDir)
	runner := &fakeRunner{}
	output := &strings.Builder{}

	globalEnvPath, err := create.GlobalEnvPath()
	if err != nil {
		t.Fatalf("GlobalEnvPath returned error: %v", err)
	}

	env := "" +
		"TOBA_PHP_VERSION=8.4\n" +
		"TOBA_STARTER_REPO=" + testStarterRepo + "\n" +
		"TOBA_SSH_TARGET=user@192.168.0.1 -p 22\n" +
		"TOBA_REMOTE_WORDPRESS_ROOT=" + testRemoteWordPressRoot + "\n"
	if err := os.MkdirAll(filepath.Dir(globalEnvPath), 0755); err != nil {
		t.Fatalf("failed to create config dir: %v", err)
	}
	if err := os.WriteFile(globalEnvPath, []byte(env), 0644); err != nil {
		t.Fatalf("failed to write global env file: %v", err)
	}

	if err := runCreateWithIO(CreateOptions{Name: "demo"}, runner, strings.NewReader("n\n"), output); err != nil {
		t.Fatalf("runCreateWithIO returned error: %v", err)
	}

	if !strings.Contains(output.String(), "Using config from "+globalEnvPath) {
		t.Fatalf("expected config source log, got %q", output.String())
	}
}

func TestRunCreateRemoteWordPressRootFlagOverridesEnv(t *testing.T) {
	baseDir := t.TempDir()
	withWorkingDir(t, baseDir)
	runner := &fakeRunner{}

	globalEnvPath, err := create.GlobalEnvPath()
	if err != nil {
		t.Fatalf("GlobalEnvPath returned error: %v", err)
	}

	env := "" +
		"TOBA_STARTER_REPO=" + testStarterRepo + "\n" +
		"TOBA_SSH_TARGET=user@192.168.0.1 -p 22\n" +
		"TOBA_REMOTE_WORDPRESS_ROOT=www/stale-from-env.example\n"
	if err := os.MkdirAll(filepath.Dir(globalEnvPath), 0755); err != nil {
		t.Fatalf("failed to create config dir: %v", err)
	}
	if err := os.WriteFile(globalEnvPath, []byte(env), 0644); err != nil {
		t.Fatalf("failed to write global env file: %v", err)
	}

	if err := runCreateWithRunner(CreateOptions{Name: "demo", RemoteWordPressRoot: testRemoteWordPressRoot}, runner); err != nil {
		t.Fatalf("runCreateWithRunner returned error: %v", err)
	}

	assertHasCommand(t, runner.commands, "ssh", []string{"-p", "22", "user@192.168.0.1", "dynamic:home"})
}

func TestRunCreateFailsWithoutGlobalConfig(t *testing.T) {
	baseDir := t.TempDir()
	withWorkingDir(t, baseDir)

	err := runCreateWithRunner(CreateOptions{}, &fakeRunner{})
	if err == nil {
		t.Fatal("expected missing global config error")
	}
	if !strings.Contains(err.Error(), "toba create <project-name>") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRunCreateFailsWhenStarterRepoIsMissing(t *testing.T) {
	baseDir := t.TempDir()
	withWorkingDir(t, baseDir)

	err := runCreateWithRunner(CreateOptions{Name: "demo", SSHTarget: "user@192.168.0.1 -p 22", RemoteWordPressRoot: testRemoteWordPressRoot}, &fakeRunner{})
	if err == nil {
		t.Fatal("expected missing starter repo error")
	}
	if !strings.Contains(err.Error(), "TOBA_STARTER_REPO") || !strings.Contains(err.Error(), "--starter-repo") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRunCreateFailsWhenSSHTargetIsMissing(t *testing.T) {
	baseDir := t.TempDir()
	withWorkingDir(t, baseDir)

	globalEnvPath, err := create.GlobalEnvPath()
	if err != nil {
		t.Fatalf("GlobalEnvPath returned error: %v", err)
	}
	if err := os.MkdirAll(filepath.Dir(globalEnvPath), 0755); err != nil {
		t.Fatalf("failed to create config dir: %v", err)
	}
	if err := os.WriteFile(globalEnvPath, []byte("TOBA_STARTER_REPO="+testStarterRepo+"\n"), 0644); err != nil {
		t.Fatalf("failed to write global env file: %v", err)
	}

	err = runCreateWithRunner(CreateOptions{Name: "demo"}, &fakeRunner{})
	if err == nil {
		t.Fatal("expected missing SSH target error")
	}
	if !strings.Contains(err.Error(), "TOBA_SSH_TARGET") || !strings.Contains(err.Error(), "--ssh-target") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRunCreateFailsWhenRemoteWordPressRootIsMissing(t *testing.T) {
	baseDir := t.TempDir()
	withWorkingDir(t, baseDir)

	globalEnvPath, err := create.GlobalEnvPath()
	if err != nil {
		t.Fatalf("GlobalEnvPath returned error: %v", err)
	}
	if err := os.MkdirAll(filepath.Dir(globalEnvPath), 0755); err != nil {
		t.Fatalf("failed to create config dir: %v", err)
	}
	if err := os.WriteFile(globalEnvPath, []byte("TOBA_STARTER_REPO="+testStarterRepo+"\nTOBA_SSH_TARGET=user@192.168.0.1 -p 22\n"), 0644); err != nil {
		t.Fatalf("failed to write global env file: %v", err)
	}

	err = runCreateWithRunner(CreateOptions{Name: "demo"}, &fakeRunner{})
	if err == nil {
		t.Fatal("expected missing remote WordPress root error")
	}
	if !strings.Contains(err.Error(), "TOBA_REMOTE_WORDPRESS_ROOT") || !strings.Contains(err.Error(), "--remote-wordpress-root") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRunCreateFailsForProjectNameWithSpaces(t *testing.T) {
	baseDir := t.TempDir()
	withWorkingDir(t, baseDir)

	err := runCreateWithRunner(CreateOptions{Name: "demo project", StarterRepo: testStarterRepo, SSHTarget: "user@192.168.0.1 -p 22", RemoteWordPressRoot: testRemoteWordPressRoot}, &fakeRunner{})
	if err == nil {
		t.Fatal("expected project name validation error")
	}
	if !strings.Contains(err.Error(), "project name cannot contain spaces") {
		t.Fatalf("unexpected error: %v", err)
	}
}
