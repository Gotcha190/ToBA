package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunCreateCleansUpFailedInstallWhenConfirmed(t *testing.T) {
	baseDir := t.TempDir()
	withWorkingDir(t, baseDir)

	input := strings.NewReader("y\n")
	output := &strings.Builder{}
	err := runCreateWithIO(
		CreateOptions{Name: "demo", SSHTarget: "user@192.168.0.1 -p 22", RemoteWordPressRoot: testRemoteWordPressRoot},
		&fakeRunner{runErrByCommand: map[string]error{"lando start": fmt.Errorf("lando failed")}},
		input,
		output,
	)
	if err == nil {
		t.Fatal("expected create to fail")
	}

	projectRoot := filepath.Join(baseDir, "demo")
	if _, statErr := os.Stat(projectRoot); !os.IsNotExist(statErr) {
		t.Fatalf("expected project root to be removed, got err=%v", statErr)
	}
	if !strings.Contains(output.String(), "Delete failed installation") {
		t.Fatalf("expected cleanup prompt, got: %s", output.String())
	}
	if strings.Contains(output.String(), "Total create time") {
		t.Fatalf("expected no total create time on failure, got: %s", output.String())
	}
}

func TestRunCreateKeepsProjectDirWhenLandoDestroyFails(t *testing.T) {
	baseDir := t.TempDir()
	withWorkingDir(t, baseDir)

	input := strings.NewReader("y\n")
	output := &strings.Builder{}
	err := runCreateWithIO(
		CreateOptions{Name: "demo", SSHTarget: "user@192.168.0.1 -p 22", RemoteWordPressRoot: testRemoteWordPressRoot},
		&fakeRunner{runErrByCommand: map[string]error{
			"lando start":      fmt.Errorf("lando failed"),
			"lando destroy -y": fmt.Errorf("destroy failed"),
		}},
		input,
		output,
	)
	if err == nil {
		t.Fatal("expected create to fail")
	}

	projectRoot := filepath.Join(baseDir, "demo")
	if _, statErr := os.Stat(projectRoot); statErr != nil {
		t.Fatalf("expected project root to remain after destroy failure, got err=%v", statErr)
	}
	if !strings.Contains(output.String(), "Failed to destroy Lando app in "+projectRoot) {
		t.Fatalf("expected destroy error, got: %s", output.String())
	}
	if !strings.Contains(output.String(), "Keeping project directory because removing it now could make manual Lando cleanup harder") {
		t.Fatalf("expected keep-directory error, got: %s", output.String())
	}
}
