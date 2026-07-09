package comfig

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestFileSystemSource(t *testing.T) {
	t.Run("reads file", func(t *testing.T) {
		dir := t.TempDir()
		contents := `{"project":"test-project"}`
		if err := os.WriteFile(filepath.Join(dir, "test.json"), []byte(contents), 0o600); err != nil {
			t.Fatal(err)
		}
		got, err := NewFileSystemSourceByDirectory(dir).Configuration(context.Background(), "test", "json")
		if err != nil {
			t.Fatal(err)
		}
		if string(got) != contents {
			t.Fatalf("got = %q", got)
		}
	})

	t.Run("errors on missing file", func(t *testing.T) {
		dir := t.TempDir()
		contents, err := NewFileSystemSourceByDirectory(dir).Configuration(context.Background(), "test", "json")
		if err == nil {
			t.Fatal("expected error for missing file")
		}
		if contents != nil {
			t.Fatal("expected nil contents for missing file")
		}
	})

	t.Run("errors on missing directory", func(t *testing.T) {
		dir := filepath.Join(t.TempDir(), "unknown")
		contents, err := NewFileSystemSourceByDirectory(dir).Configuration(context.Background(), "test", "json")
		if err == nil {
			t.Fatal("expected error for missing directory")
		}
		if contents != nil {
			t.Fatal("expected nil contents for missing directory")
		}
	})
}
