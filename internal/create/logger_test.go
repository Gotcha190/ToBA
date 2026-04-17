package create

import (
	"strings"
	"testing"
)

func TestConsoleLoggerPromptAndErrorCodeFormat(t *testing.T) {
	var output strings.Builder
	logger := NewConsoleLogger(&output)

	logger.Prompt("Delete project? [y/N]: ")
	logger.ErrorCode("FAIL_CODE", "something failed")

	got := output.String()
	if !strings.Contains(got, "[PROMPT] Delete project? [y/N]: ") {
		t.Fatalf("unexpected prompt output: %q", got)
	}
	if !strings.Contains(got, "[ERROR][FAIL_CODE] something failed\n") {
		t.Fatalf("unexpected coded error output: %q", got)
	}
}
