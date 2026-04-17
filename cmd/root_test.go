package cmd

import (
	"io"
	"os"
	"strings"
	"testing"
)

func TestPrintUsageDoesNotMentionUpdate(t *testing.T) {
	stdout := os.Stdout
	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatalf("failed to create pipe: %v", err)
	}
	defer reader.Close()

	os.Stdout = writer
	printUsage()
	writer.Close()
	os.Stdout = stdout

	output, readErr := io.ReadAll(reader)
	if readErr != nil {
		t.Fatalf("failed to read usage output: %v", readErr)
	}

	if strings.Contains(string(output), "update") {
		t.Fatalf("usage should not mention update, got %q", string(output))
	}
}
