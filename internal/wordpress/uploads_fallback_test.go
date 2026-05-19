package wordpress

import (
	"strings"
	"testing"
)

func TestUploadsFallbackBlockUsesSourceHostAndOrigin(t *testing.T) {
	block, err := UploadsFallbackBlock("https://starter.example.test/some/path")
	if err != nil {
		t.Fatalf("UploadsFallbackBlock returned error: %v", err)
	}

	for _, expected := range []string{
		"RewriteCond %{HTTP_HOST} !^starter\\.example\\.test$ [NC]",
		"RewriteRule ^wp-content/uploads/(.*)$ https://starter.example.test/wp-content/uploads/$1 [R=302,NE,L]",
	} {
		if !strings.Contains(block, expected) {
			t.Fatalf("expected block to contain %q, got:\n%s", expected, block)
		}
	}
}

func TestUpsertUploadsFallbackBlockReplacesExistingBlock(t *testing.T) {
	block, err := UploadsFallbackBlock("https://starter.example.test")
	if err != nil {
		t.Fatalf("UploadsFallbackBlock returned error: %v", err)
	}
	existing := "# BEGIN ToBA Uploads Fallback\nold\n# END ToBA Uploads Fallback\n\n# BEGIN WordPress\nwp\n# END WordPress\n"

	updated := UpsertUploadsFallbackBlock(existing, block)

	if strings.Contains(updated, "old") {
		t.Fatalf("expected old block to be removed, got:\n%s", updated)
	}
	if count := strings.Count(updated, "# BEGIN ToBA Uploads Fallback"); count != 1 {
		t.Fatalf("expected one fallback block, got %d in:\n%s", count, updated)
	}
}

func TestUpsertUploadsFallbackBlockInsertsBeforeWordPressMarker(t *testing.T) {
	block, err := UploadsFallbackBlock("https://starter.example.test")
	if err != nil {
		t.Fatalf("UploadsFallbackBlock returned error: %v", err)
	}
	existing := "# custom\n\n# BEGIN WordPress\nwp\n"

	updated := UpsertUploadsFallbackBlock(existing, block)

	fallbackIndex := strings.Index(updated, "# BEGIN ToBA Uploads Fallback")
	wordPressIndex := strings.Index(updated, "# BEGIN WordPress")
	if fallbackIndex < 0 || wordPressIndex < 0 || fallbackIndex > wordPressIndex {
		t.Fatalf("expected fallback before WordPress marker, got:\n%s", updated)
	}
	if !strings.HasPrefix(updated, "# custom\n\n") {
		t.Fatalf("expected existing prelude to be preserved, got:\n%s", updated)
	}
}

func TestUpsertUploadsFallbackBlockCreatesContentWhenHtaccessMissing(t *testing.T) {
	block, err := UploadsFallbackBlock("https://starter.example.test")
	if err != nil {
		t.Fatalf("UploadsFallbackBlock returned error: %v", err)
	}

	updated := UpsertUploadsFallbackBlock("", block)

	if updated != block+"\n" {
		t.Fatalf("unexpected content for missing htaccess:\n%s", updated)
	}
}
