package sourcedata

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestRemotePreparationScript(t *testing.T) {
	if _, err := exec.LookPath("sh"); err != nil {
		t.Skip("POSIX shell is required")
	}
	for _, tc := range []struct {
		name     string
		absolute bool
		fail     bool
		signal   bool
		uploads  bool
	}{
		{name: "relative-success", uploads: true},
		{name: "absolute-success", absolute: true, uploads: true},
		{name: "no-uploads"},
		{name: "relative-failure", fail: true, uploads: true},
		{name: "absolute-failure", absolute: true, fail: true, uploads: true},
		{name: "signal-cleanup", fail: true, signal: true, uploads: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			base := t.TempDir()
			root := filepath.Join(base, "www", "site with spaces")
			bin := filepath.Join(base, "bin")
			for _, dir := range []string{filepath.Join(root, "wp-content"), bin} {
				if err := os.MkdirAll(dir, 0755); err != nil {
					t.Fatal(err)
				}
			}
			// Failed export waits until both archives start. Their TERM traps write
			// late data, which must be removed only after the children have exited.
			wp := "#!/bin/sh\nif [ \"$1\" = option ]; then echo https://example.test; exit; fi\necho dump > \"$3\"\n"
			zip := "#!/bin/sh\nfor arg do case \"$arg\" in ../*) target=$arg;; esac; done\necho archive > \"$target\"\n"
			if tc.fail {
				wp += "while [ ! -f plugins.zip ] || [ ! -f uploads.zip ]; do sleep 0.01; done\nexit 7\n"
				if tc.signal {
					wp = strings.TrimSuffix(wp, "exit 7\n") + "kill -TERM \"$PPID\"\nexit 0\n"
				}
				zip += "trap 'echo late > \"$target\"; exit 0' TERM\nwhile :; do sleep 0.01; done\n"
			}
			for name, script := range map[string]string{"wp84": wp, "zip": zip} {
				if err := os.WriteFile(filepath.Join(bin, name), []byte(script), 0755); err != nil {
					t.Fatal(err)
				}
			}
			remoteRoot := "www/site with spaces"
			if tc.absolute {
				remoteRoot = root
			}
			script := remotePreparationScript(remoteRoot, remoteRoot+"/dump.sql", remoteRoot+"/plugins.zip", remoteRoot+"/uploads.zip", remoteRoot+"/home.txt", tc.uploads)
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			cmd := exec.CommandContext(ctx, "sh", "-c", script)
			cmd.Dir = base
			cmd.Env = append(os.Environ(), "PATH="+bin+string(os.PathListSeparator)+os.Getenv("PATH"))
			output, err := cmd.CombinedOutput()
			if tc.fail {
				expectedCode := 7
				if tc.signal {
					expectedCode = 130
				}
				if err == nil || cmd.ProcessState.ExitCode() != expectedCode {
					t.Fatalf("expected export failure: %v, %s", err, output)
				}
			} else if err != nil || strings.TrimSpace(string(output)) != "https://example.test" {
				t.Fatalf("unexpected preparation result: %v, %s", err, output)
			}
			for _, name := range []string{"dump.sql", "plugins.zip", "uploads.zip", "home.txt"} {
				_, err := os.Stat(filepath.Join(root, name))
				wantMissing := tc.fail || (name == "uploads.zip" && !tc.uploads)
				if wantMissing && !os.IsNotExist(err) {
					t.Errorf("artifact %s should not remain: %v", name, err)
				}
				if !wantMissing && err != nil {
					t.Errorf("missing artifact %s: %v", name, err)
				}
			}
		})
	}
}
