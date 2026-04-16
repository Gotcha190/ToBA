package cmd

import (
	"strings"
	"testing"
)

func TestRunUpdateRejectsLinkFlag(t *testing.T) {
	err := runUpdate([]string{"--link=/tmp/override.zip"})
	if err == nil {
		t.Fatal("expected runUpdate to reject --link")
	}
	if !strings.Contains(err.Error(), "flag provided but not defined") {
		t.Fatalf("unexpected error: %v", err)
	}
}
