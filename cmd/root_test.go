package cmd

import (
	"bytes"
	"flag"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/gotcha190/toba/internal/create"
)

func TestPrintUsageDoesNotMentionUpdate(t *testing.T) {
	stdout := os.Stdout
	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatalf("failed to create pipe: %v", err)
	}
	defer func() {
		_ = reader.Close()
	}()

	os.Stdout = writer
	printUsage()
	_ = writer.Close()
	os.Stdout = stdout

	output, readErr := io.ReadAll(reader)
	if readErr != nil {
		t.Fatalf("failed to read usage output: %v", readErr)
	}

	if strings.Contains(string(output), "update") {
		t.Fatalf("usage should not mention update, got %q", string(output))
	}
}

func TestPrintUsageStartsWithBanner(t *testing.T) {
	stdout := os.Stdout
	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatalf("failed to create pipe: %v", err)
	}
	defer func() {
		_ = reader.Close()
	}()

	os.Stdout = writer
	printUsage()
	_ = writer.Close()
	os.Stdout = stdout

	output, readErr := io.ReadAll(reader)
	if readErr != nil {
		t.Fatalf("failed to read usage output: %v", readErr)
	}

	rendered := string(output)
	if !strings.HasPrefix(rendered, usageBanner) {
		t.Fatalf("usage should start with banner, got %q", rendered)
	}

	usageIndex := strings.Index(rendered, "Usage: toba <command>")
	if usageIndex == -1 {
		t.Fatalf("usage header missing, got %q", rendered)
	}

	if usageIndex <= len(usageBanner)-1 {
		t.Fatalf("usage header should be printed after banner, got %q", rendered)
	}
}

func TestRunConfigWithoutInitCreatesGlobalTemplate(t *testing.T) {
	workDir := t.TempDir()
	original, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get working directory: %v", err)
	}
	if err := os.Chdir(workDir); err != nil {
		t.Fatalf("failed to chdir: %v", err)
	}
	t.Cleanup(func() {
		if err := os.Chdir(original); err != nil {
			t.Fatalf("failed to restore working directory: %v", err)
		}
	})

	configHome := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", configHome)

	if err := runConfig(nil); err != nil {
		t.Fatalf("runConfig returned error: %v", err)
	}

	globalEnvPath, err := create.GlobalEnvPath()
	if err != nil {
		t.Fatalf("GlobalEnvPath returned error: %v", err)
	}
	if _, err := os.Stat(globalEnvPath); err != nil {
		t.Fatalf("expected %s to exist: %v", globalEnvPath, err)
	}
}

func TestRunConfigRejectsLegacyInitArgument(t *testing.T) {
	err := runConfig([]string{"init"})
	if err == nil {
		t.Fatal("expected usage error")
	}
	if err.Error() != "usage: toba config" {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestParseCreateOptionsSetsSequentialFlag(t *testing.T) {
	opts, err := parseCreateOptions([]string{"demo", "--sequential"})
	if err != nil {
		t.Fatalf("parseCreateOptions returned error: %v", err)
	}
	if opts.Name != "demo" {
		t.Fatalf("expected project name demo, got %q", opts.Name)
	}
	if !opts.Sequential {
		t.Fatal("expected sequential flag to be enabled")
	}
}

func TestParseCreateOptionsSetsNoUploadsFlag(t *testing.T) {
	opts, err := parseCreateOptions([]string{"demo", "--no-uploads"})
	if err != nil {
		t.Fatalf("parseCreateOptions returned error: %v", err)
	}
	if opts.Name != "demo" {
		t.Fatalf("expected project name demo, got %q", opts.Name)
	}
	if !opts.NoUploads {
		t.Fatal("expected no-uploads flag to be enabled")
	}
}

func TestCreateUsageMentionsNoUploadsFlag(t *testing.T) {
	stderr := os.Stderr
	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatalf("failed to create pipe: %v", err)
	}
	defer func() {
		_ = reader.Close()
	}()

	os.Stderr = writer
	_, parseErr := parseCreateOptions([]string{"--help"})
	_ = writer.Close()
	os.Stderr = stderr

	if parseErr != flag.ErrHelp {
		t.Fatalf("expected flag.ErrHelp, got %v", parseErr)
	}

	var output bytes.Buffer
	if _, err := io.Copy(&output, reader); err != nil {
		t.Fatalf("failed to read usage output: %v", err)
	}
	if !strings.Contains(output.String(), "--no-uploads") {
		t.Fatalf("expected create usage to mention --no-uploads, got %q", output.String())
	}
}
