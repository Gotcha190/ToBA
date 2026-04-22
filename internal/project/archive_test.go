package project

import (
	"archive/zip"
	"bytes"
	"compress/gzip"
	"os"
	"path/filepath"
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
