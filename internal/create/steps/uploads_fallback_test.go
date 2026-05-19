package steps

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gotcha190/toba/internal/create"
)

func TestConfigureUploadsFallbackStepWritesHtaccess(t *testing.T) {
	ctx := create.NewContext(t.TempDir(), create.ProjectConfig{Name: "demo", NoUploads: true}, &starterTestLogger{}, &starterTestRunner{})
	ctx.StarterData.SourceURL = "https://starter.example.test"

	if err := os.MkdirAll(ctx.Paths.AppDir, 0755); err != nil {
		t.Fatalf("failed to create app dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(ctx.Paths.AppDir, ".htaccess"), []byte("# BEGIN WordPress\nwp\n"), 0644); err != nil {
		t.Fatalf("failed to write htaccess: %v", err)
	}

	if err := NewConfigureUploadsFallbackStep().Run(ctx); err != nil {
		t.Fatalf("ConfigureUploadsFallbackStep returned error: %v", err)
	}

	content, err := os.ReadFile(filepath.Join(ctx.Paths.AppDir, ".htaccess"))
	if err != nil {
		t.Fatalf("failed to read htaccess: %v", err)
	}
	if !strings.Contains(string(content), "https://starter.example.test/wp-content/uploads/$1") {
		t.Fatalf("expected fallback target in htaccess, got:\n%s", string(content))
	}
	if strings.Index(string(content), "# BEGIN ToBA Uploads Fallback") > strings.Index(string(content), "# BEGIN WordPress") {
		t.Fatalf("expected fallback before WordPress block, got:\n%s", string(content))
	}
}

func TestConfigureUploadsFallbackStepDryRunDoesNotWrite(t *testing.T) {
	ctx := create.NewContext(t.TempDir(), create.ProjectConfig{Name: "demo", DryRun: true, NoUploads: true}, &starterTestLogger{}, &starterTestRunner{})
	ctx.StarterData.SourceURL = "https://starter.example.test"

	if err := NewConfigureUploadsFallbackStep().Run(ctx); err != nil {
		t.Fatalf("ConfigureUploadsFallbackStep returned error: %v", err)
	}
	if _, err := os.Stat(filepath.Join(ctx.Paths.AppDir, ".htaccess")); !os.IsNotExist(err) {
		t.Fatalf("expected dry-run not to write htaccess, stat error=%v", err)
	}
}
