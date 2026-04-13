package wordpress

import (
	"errors"
	"reflect"
	"testing"

	"github.com/gotcha190/ToBA/internal/create"
)

type recordedCommand struct {
	dir  string
	cmd  string
	args []string
}

type fakeRunner struct {
	commands []recordedCommand
	err      error
}

func (r *fakeRunner) Run(dir string, cmd string, args ...string) error {
	r.commands = append(r.commands, recordedCommand{
		dir:  dir,
		cmd:  cmd,
		args: append([]string(nil), args...),
	})
	return r.err
}

func (r *fakeRunner) CaptureOutput(dir string, cmd string, args ...string) (string, error) {
	return "", r.err
}

func TestInstallRunsExpectedCommands(t *testing.T) {
	runner := &fakeRunner{}
	config := create.ProjectConfig{
		Name:   "my-project",
		Domain: "my-project.lndo.site",
	}

	err := Install(runner, "/tmp/demo", config)
	if err != nil {
		t.Fatalf("Install returned error: %v", err)
	}

	expected := []recordedCommand{
		{
			dir:  "/tmp/demo",
			cmd:  "lando",
			args: []string{"wp", "core", "download", "--locale=pl_PL"},
		},
		{
			dir:  "/tmp/demo",
			cmd:  "lando",
			args: []string{"wp", "config", "create", "--dbname=wordpress", "--dbuser=wordpress", "--dbpass=wordpress", "--dbhost=database"},
		},
		{
			dir:  "/tmp/demo",
			cmd:  "lando",
			args: []string{"wp", "core", "install", "--url=my-project.lndo.site", "--title=My Project", "--admin_user=tamago", "--admin_email=email@email.pl", "--admin_password=tamago"},
		},
	}

	if !reflect.DeepEqual(runner.commands, expected) {
		t.Fatalf("unexpected command sequence:\nexpected: %#v\ngot: %#v", expected, runner.commands)
	}
}

func TestInstallReturnsCommandError(t *testing.T) {
	expectedErr := errors.New("boom")
	runner := &fakeRunner{err: expectedErr}

	err := Install(runner, "/tmp/demo", create.ProjectConfig{Name: "demo", Domain: "demo.lndo.site"})
	if !errors.Is(err, expectedErr) {
		t.Fatalf("expected %v, got %v", expectedErr, err)
	}
}

func TestProjectTitle(t *testing.T) {
	if got := ProjectTitle("my-project_name"); got != "My Project Name" {
		t.Fatalf("unexpected title: %q", got)
	}
}
