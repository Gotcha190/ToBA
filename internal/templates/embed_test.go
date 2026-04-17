package templates

import (
	"testing"
)

func TestReadReturnsEmbeddedTemplates(t *testing.T) {
	for _, name := range []string{
		"config/php.ini",
		"lando/.lando.yml",
		"wp-cli.yml",
	} {
		content, err := Read(name)
		if err != nil {
			t.Fatalf("Read(%q) returned error: %v", name, err)
		}
		if len(content) == 0 {
			t.Fatalf("Read(%q) returned empty content", name)
		}
	}
}

func TestReadRejectsMissingTemplate(t *testing.T) {
	_, err := Read("missing.yml")
	if err != nil {
		return
	}
	t.Fatal("expected missing template error")
}
