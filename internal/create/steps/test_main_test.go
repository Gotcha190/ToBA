package steps

import (
	"fmt"
	"os"
	"testing"
)

// TestMain isolates starter artifacts created by this test process.
//
// Parameters:
// - m: package test suite
//
// Returns:
// - null
//
// Side effects:
// - redirects temporary files to a private directory and removes it on exit
func TestMain(m *testing.M) {
	dir, err := os.MkdirTemp("", "toba-step-tests-")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	for _, key := range []string{"TMPDIR", "TMP", "TEMP"} {
		if err := os.Setenv(key, dir); err != nil {
			fmt.Fprintln(os.Stderr, err)
			_ = os.RemoveAll(dir)
			os.Exit(1)
		}
	}
	code := m.Run()
	if err := os.RemoveAll(dir); err != nil {
		fmt.Fprintln(os.Stderr, err)
		code = 1
	}
	os.Exit(code)
}
