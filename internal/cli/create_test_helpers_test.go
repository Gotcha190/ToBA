package cli

import (
	"archive/zip"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"testing"

	"github.com/gotcha190/toba/internal/create"
)

const testStarterRepo = "git@example.com:company/starter.git"
const testRemoteWordPressRoot = "www/example.com"

type recordedCommand struct {
	dir  string
	cmd  string
	args []string
}

type fakeRunner struct {
	mu                  sync.Mutex
	commands            []recordedCommand
	runErr              error
	runErrByCommand     map[string]error
	homeURL             string
	failCmd             string
	captureErr          error
	captureErrByCommand map[string]error
}

func (r *fakeRunner) Run(dir string, cmd string, args ...string) error {
	r.mu.Lock()
	r.commands = append(r.commands, recordedCommand{
		dir:  dir,
		cmd:  cmd,
		args: append([]string(nil), args...),
	})
	r.mu.Unlock()
	if cmd == "git" && len(args) == 3 && args[0] == "clone" {
		if err := os.MkdirAll(filepath.Join(dir, args[2]), 0755); err != nil {
			return err
		}
	}
	if cmd == "scp" && len(args) >= 3 {
		if err := materializeDownloadedFixture(args[len(args)-2], args[len(args)-1]); err != nil {
			return err
		}
	}
	if r.runErrByCommand != nil {
		if err, ok := r.runErrByCommand[cmd+" "+strings.Join(args, " ")]; ok {
			return err
		}
	}
	if r.runErr != nil && (r.failCmd == "" || r.failCmd == cmd) {
		return r.runErr
	}
	return nil
}

func (r *fakeRunner) CaptureOutput(dir string, cmd string, args ...string) (string, error) {
	r.mu.Lock()
	r.commands = append(r.commands, recordedCommand{
		dir:  dir,
		cmd:  cmd,
		args: append([]string(nil), args...),
	})
	r.mu.Unlock()
	if r.captureErrByCommand != nil {
		if err, ok := r.captureErrByCommand[cmd+" "+strings.Join(args, " ")]; ok {
			return "", err
		}
	}
	if r.captureErr != nil {
		return "", r.captureErr
	}
	if r.runErr != nil && (r.failCmd == "" || r.failCmd == cmd) {
		return "", r.runErr
	}
	if cmd == "ssh" && len(args) > 0 && strings.Contains(args[len(args)-1], "wp84 option get home") {
		if r.homeURL == "" {
			return "https://starter.example.test\n", nil
		}
		return r.homeURL + "\n", nil
	}
	if cmd == "lando" && len(args) == 3 && args[0] == "wp" && args[1] == "eval" {
		if strings.Contains(args[2], "get_option('stylesheet') ?: get_option('template')") {
			return "toet\n", nil
		}
	}
	if cmd == "lando" && len(args) == 5 && args[0] == "wp" && args[1] == "user" && args[2] == "get" && args[3] == "tamago" && args[4] == "--field=ID" {
		return "1\n", nil
	}
	return "", nil
}

func assertProjectSkeleton(t *testing.T, paths create.ProjectPaths) {
	t.Helper()

	for _, dir := range []string{paths.Root, paths.AppDir, paths.ConfigDir} {
		info, statErr := os.Stat(dir)
		if statErr != nil {
			t.Fatalf("expected %s to exist: %v", dir, statErr)
		}
		if !info.IsDir() {
			t.Fatalf("expected %s to be a directory", dir)
		}
	}
}

func assertStaticConfigs(t *testing.T, paths create.ProjectPaths) {
	t.Helper()

	landoConfigPath := filepath.Join(paths.Root, ".lando.yml")
	landoConfig, readErr := os.ReadFile(landoConfigPath)
	if readErr != nil {
		t.Fatalf("expected %s to exist: %v", landoConfigPath, readErr)
	}
	output := string(landoConfig)
	if !strings.Contains(output, "name: demo") {
		t.Fatalf("expected rendered .lando.yml to include project name, got:\n%s", output)
	}
	if !strings.Contains(output, `php: "8.4"`) {
		t.Fatalf("expected rendered .lando.yml to include php version, got:\n%s", output)
	}

	phpINIPath := filepath.Join(paths.ConfigDir, "php.ini")
	phpINI, readPHPINIError := os.ReadFile(phpINIPath)
	if readPHPINIError != nil {
		t.Fatalf("expected %s to exist: %v", phpINIPath, readPHPINIError)
	}
	if !strings.Contains(string(phpINI), "xdebug.mode = profile,debug,develop") {
		t.Fatalf("expected generated php.ini to contain xdebug mode, got:\n%s", string(phpINI))
	}

	wpCLIPath := filepath.Join(paths.Root, "wp-cli.yml")
	wpCLIConfig, readWPCLIError := os.ReadFile(wpCLIPath)
	if readWPCLIError != nil {
		t.Fatalf("expected %s to exist: %v", wpCLIPath, readWPCLIError)
	}
	if string(wpCLIConfig) != "path: app\nserver:\n  docroot: app\napache_modules:\n  - mod_rewrite\n" {
		t.Fatalf("unexpected wp-cli.yml contents:\n%s", string(wpCLIConfig))
	}
}

func assertHasCommand(t *testing.T, commands []recordedCommand, cmd string, args []string) {
	t.Helper()

	for _, command := range commands {
		if command.cmd != cmd {
			continue
		}
		if argsMatch(command.args, args) {
			return
		}
	}

	t.Fatalf("expected command %s %v, got %#v", cmd, args, commands)
}

func assertHasCommandInDir(t *testing.T, commands []recordedCommand, dir string, cmd string, args []string) {
	t.Helper()

	for _, command := range commands {
		if command.dir != dir || command.cmd != cmd {
			continue
		}
		if argsMatch(command.args, args) {
			return
		}
	}

	t.Fatalf("expected command in %s: %s %v, got %#v", dir, cmd, args, commands)
}

func assertNoCommand(t *testing.T, commands []recordedCommand, cmd string, args []string) {
	t.Helper()

	for _, command := range commands {
		if command.cmd != cmd {
			continue
		}
		if args == nil || argsMatch(command.args, args) {
			t.Fatalf("did not expect command %s %v, got %#v", cmd, args, commands)
		}
	}
}

func assertCommandCount(t *testing.T, commands []recordedCommand, cmd string, args []string, expected int) {
	t.Helper()

	count := 0
	for _, command := range commands {
		if command.cmd != cmd {
			continue
		}
		if argsMatch(command.args, args) {
			count++
		}
	}

	if count != expected {
		t.Fatalf("expected %d occurrences of %s %v, got %d in %#v", expected, cmd, args, count, commands)
	}
}

func argsMatch(actual []string, expected []string) bool {
	if len(actual) != len(expected) {
		return false
	}

	for i := range actual {
		if strings.HasPrefix(expected[i], "dynamic:") {
			if !matchesDynamicArg(actual[i], strings.TrimPrefix(expected[i], "dynamic:")) {
				return false
			}
			continue
		}
		if actual[i] != expected[i] {
			return false
		}
	}

	return true
}

func assertNodeDependsOn(t *testing.T, nodes []create.StepNode, id string, expected []string) {
	t.Helper()

	for _, node := range nodes {
		if node.ID != id {
			continue
		}
		if !reflect.DeepEqual(node.DependsOn, expected) {
			t.Fatalf("unexpected dependencies for %s:\nexpected: %#v\ngot: %#v", id, expected, node.DependsOn)
		}
		return
	}

	t.Fatalf("expected node %s in %#v", id, nodes)
}

func assertNoNode(t *testing.T, nodes []create.StepNode, id string) {
	t.Helper()

	for _, node := range nodes {
		if node.ID == id {
			t.Fatalf("did not expect node %s in %#v", id, nodes)
		}
	}
}

func matchesDynamicArg(actual string, pattern string) bool {
	switch pattern {
	case "home":
		return strings.Contains(actual, "wp84 option get home") && strings.Contains(actual, "'"+testRemoteWordPressRoot+"'")
	case "no-uploads-home":
		return strings.Contains(actual, "wp84 option get home") && strings.Contains(actual, "'"+testRemoteWordPressRoot+"'") && !strings.Contains(actual, "pid_uploads") && !strings.Contains(actual, "-i 'uploads/*'")
	case "remote-sql":
		return strings.HasPrefix(actual, "user@192.168.0.1:"+testRemoteWordPressRoot+"/") && strings.HasSuffix(actual, ".sql")
	case "local-sql":
		return strings.HasSuffix(actual, ".sql")
	case "remote-uploads":
		return strings.HasPrefix(actual, "user@192.168.0.1:"+testRemoteWordPressRoot+"/") && strings.HasSuffix(actual, "-uploads.zip")
	case "local-uploads":
		return strings.HasSuffix(actual, "-uploads.zip")
	default:
		return false
	}
}

func withWorkingDir(t *testing.T, dir string) {
	t.Helper()

	original, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get working directory: %v", err)
	}

	if err := os.Chdir(dir); err != nil {
		t.Fatalf("failed to chdir to temp dir: %v", err)
	}

	originalConfigHome, hadConfigHome := os.LookupEnv("XDG_CONFIG_HOME")
	configHome := filepath.Join(dir, ".config-home")
	if err := os.Setenv("XDG_CONFIG_HOME", configHome); err != nil {
		t.Fatalf("failed to set XDG_CONFIG_HOME: %v", err)
	}

	t.Cleanup(func() {
		if err := os.Chdir(original); err != nil {
			t.Fatalf("failed to restore working directory: %v", err)
		}
		if hadConfigHome {
			if err := os.Setenv("XDG_CONFIG_HOME", originalConfigHome); err != nil {
				t.Fatalf("failed to restore XDG_CONFIG_HOME: %v", err)
			}
			return
		}
		if err := os.Unsetenv("XDG_CONFIG_HOME"); err != nil {
			t.Fatalf("failed to unset XDG_CONFIG_HOME: %v", err)
		}
	})
}

func materializeDownloadedFixture(remoteSource string, localTarget string) error {
	switch {
	case strings.HasSuffix(remoteSource, ".sql"):
		return os.WriteFile(localTarget, []byte("# Home URL: https://starter.example.test\nSELECT 1;\n"), 0644)
	case strings.HasSuffix(remoteSource, "-plugins.zip"):
		return writeZipFixture(localTarget, map[string]string{
			"plugins/advanced-custom-fields-pro/acf.php": "<?php\n",
			"plugins/wp-optimize/wp-optimize.php":        "<?php\n",
		})
	case strings.HasSuffix(remoteSource, "-uploads.zip"):
		return writeZipFixture(localTarget, map[string]string{
			"uploads/2025/07/example.jpg": "image",
			"uploads/2026/readme.txt":     "uploaded",
		})
	default:
		return nil
	}
}

func writeCreateTestFile(t *testing.T, path string, content string) {
	t.Helper()

	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatalf("failed to create parent dir for %s: %v", path, err)
	}
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write %s: %v", path, err)
	}
}

func writeZipFixture(path string, files map[string]string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}

	output, err := os.Create(path)
	if err != nil {
		return err
	}
	defer func() {
		_ = output.Close()
	}()

	writer := zip.NewWriter(output)
	for fileName, content := range files {
		entry, err := writer.Create(fileName)
		if err != nil {
			return err
		}
		if _, err := entry.Write([]byte(content)); err != nil {
			return err
		}
	}

	return writer.Close()
}
