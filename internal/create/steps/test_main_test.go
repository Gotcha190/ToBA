package steps

import (
	"os"
	"path/filepath"
	"testing"
)

func TestMain(m *testing.M) {
	cleanupTobaStarterTemp()
	code := m.Run()
	cleanupTobaStarterTemp()
	os.Exit(code)
}

func cleanupTobaStarterTemp() {
	matches, err := filepath.Glob(filepath.Join(os.TempDir(), "toba-starter-*"))
	if err != nil {
		return
	}

	for _, match := range matches {
		_ = os.RemoveAll(match)
	}
}
