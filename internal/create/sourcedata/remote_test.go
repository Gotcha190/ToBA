package sourcedata

import (
	"io"
	"os"
	"runtime"
	"testing"

	"github.com/gotcha190/toba/internal/create"
)

type remoteTestRunner struct{ create.NoopRunner }

// CaptureOutput supplies the remote URL without contacting an SSH host.
func (remoteTestRunner) CaptureOutput(string, string, ...string) (string, error) {
	return "https://example.test", nil
}

func TestRemotePreparationAllowsConcurrentThemeReads(t *testing.T) {
	for _, dryRun := range []bool{false, true} {
		ctx := create.NewContext(t.TempDir(), create.ProjectConfig{
			Name: "demo", SSHTarget: "user@example.test -p 22",
			RemoteWordPressRoot: "www/example.test", DryRun: dryRun,
		}, create.NewConsoleLogger(io.Discard), remoteTestRunner{})
		ctx.StarterData.Mode = ModeRemote
		started, stop, done := make(chan struct{}), make(chan struct{}), make(chan struct{})
		go func() {
			defer close(done)
			close(started)
			for {
				// Clone and git setup inspect this field while SSH prepares data.
				if len(ctx.StarterData.ThemePaths) != 0 {
					t.Error("remote mode acquired local themes")
				}
				select {
				case <-stop:
					return
				default:
					runtime.Gosched()
				}
			}
		}()
		<-started
		err := Prepare(ctx)
		close(stop)
		<-done
		if ctx.StarterData.TempDir != "" {
			if cleanupErr := os.RemoveAll(ctx.StarterData.TempDir); cleanupErr != nil {
				t.Fatal(cleanupErr)
			}
		}
		if err != nil {
			t.Fatal(err)
		}
		if dryRun && ctx.StarterData.TempDir != "" {
			t.Fatal("dry-run must not claim ownership of a temp directory")
		}
	}
}
