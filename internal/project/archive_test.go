package project

import (
	"archive/zip"
	"bytes"
	"compress/gzip"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestExtractZipPreservesStructure(t *testing.T) {
	data := zipBytes(t, map[string]string{
		"plugins/example/plugin.php": "<?php echo 'ok';",
		"plugins/example/readme.txt": "hello",
	})

	dest := t.TempDir()
	if err := ExtractZip(data, dest); err != nil {
		t.Fatalf("ExtractZip returned error: %v", err)
	}

	for _, expected := range []string{
		filepath.Join(dest, "plugins", "example", "plugin.php"),
		filepath.Join(dest, "plugins", "example", "readme.txt"),
	} {
		if _, err := os.Stat(expected); err != nil {
			t.Fatalf("expected %s to exist: %v", expected, err)
		}
	}
}

func TestExtractZipRejectsZipSlip(t *testing.T) {
	data := zipBytes(t, map[string]string{
		"../../evil.txt": "boom",
	})

	dest := t.TempDir()
	if err := ExtractZip(data, dest); err == nil {
		t.Fatal("expected zip-slip error")
	}
}

func TestExtractZipFileRejectsSymlink(t *testing.T) {
	path := filepath.Join(t.TempDir(), "symlink.zip")
	output, err := os.Create(path)
	if err != nil {
		t.Fatalf("failed to create %s: %v", path, err)
	}

	writer := zip.NewWriter(output)
	header := &zip.FileHeader{Name: "plugins/example-link"}
	header.SetMode(os.ModeSymlink | 0777)

	entry, err := writer.CreateHeader(header)
	if err != nil {
		t.Fatalf("failed to create symlink entry: %v", err)
	}
	if _, err := entry.Write([]byte("plugins/example")); err != nil {
		t.Fatalf("failed to write symlink payload: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("failed to close zip writer: %v", err)
	}
	if err := output.Close(); err != nil {
		t.Fatalf("failed to close %s: %v", path, err)
	}

	err = ExtractZipFile(path, t.TempDir())
	if err == nil {
		t.Fatal("expected symlink archive to be rejected")
	}
	if !strings.Contains(err.Error(), "symlink") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestExtractZipFilesFallsBackToSequentialWhenArchivesOverlap(t *testing.T) {
	dir := t.TempDir()
	first := zipFile(t, dir, "plugins-a.zip", map[string]string{
		"plugins/example/plugin.php": "<?php echo 'first';",
	})
	second := zipFile(t, dir, "plugins-b.zip", map[string]string{
		"plugins/example/plugin.php": "<?php echo 'second';",
	})

	dest := t.TempDir()
	if err := ExtractZipFiles([]string{first, second}, dest); err != nil {
		t.Fatalf("ExtractZipFiles returned error: %v", err)
	}

	content, err := os.ReadFile(filepath.Join(dest, "plugins", "example", "plugin.php"))
	if err != nil {
		t.Fatalf("failed to read extracted file: %v", err)
	}
	if string(content) != "<?php echo 'second';" {
		t.Fatalf("expected last archive to win for overlapping file, got %q", string(content))
	}
}

func TestWriteGzipFile(t *testing.T) {
	var source bytes.Buffer
	gzipWriter := gzip.NewWriter(&source)
	if _, err := gzipWriter.Write([]byte("hello")); err != nil {
		t.Fatalf("failed to write gzip content: %v", err)
	}
	if err := gzipWriter.Close(); err != nil {
		t.Fatalf("failed to close gzip writer: %v", err)
	}

	dest := filepath.Join(t.TempDir(), "database.sql")
	if err := WriteGzipFile(source.Bytes(), dest); err != nil {
		t.Fatalf("WriteGzipFile returned error: %v", err)
	}

	content, err := os.ReadFile(dest)
	if err != nil {
		t.Fatalf("failed to read %s: %v", dest, err)
	}
	if string(content) != "hello" {
		t.Fatalf("unexpected content: %q", string(content))
	}
}

func zipBytes(t *testing.T, files map[string]string) []byte {
	t.Helper()

	var buffer bytes.Buffer
	writer := zip.NewWriter(&buffer)

	for name, content := range files {
		entry, err := writer.Create(name)
		if err != nil {
			t.Fatalf("failed to create zip entry %s: %v", name, err)
		}

		if _, err := entry.Write([]byte(content)); err != nil {
			t.Fatalf("failed to write zip entry %s: %v", name, err)
		}
	}

	if err := writer.Close(); err != nil {
		t.Fatalf("failed to close zip writer: %v", err)
	}

	return buffer.Bytes()
}

func zipFile(t *testing.T, dir string, name string, files map[string]string) string {
	t.Helper()

	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, zipBytes(t, files), 0644); err != nil {
		t.Fatalf("failed to write %s: %v", path, err)
	}

	return path
}
