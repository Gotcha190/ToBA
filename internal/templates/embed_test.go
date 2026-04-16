package templates

import (
	"strings"
	"testing"
)

func TestReadUsesEmbeddedPrefix(t *testing.T) {
	content, err := Read("embedded:wp-cli.yml")
	if err != nil {
		t.Fatalf("Read returned error: %v", err)
	}
	if !strings.Contains(string(content), "path: app") {
		t.Fatalf("unexpected wp-cli.yml content: %q", string(content))
	}
}

func TestListReturnsEmptySliceForMissingEmbeddedDir(t *testing.T) {
	entries, err := List("wordpress/does-not-exist")
	if err != nil {
		t.Fatalf("List returned error: %v", err)
	}
	if len(entries) != 0 {
		t.Fatalf("expected no entries, got %v", entries)
	}
}
