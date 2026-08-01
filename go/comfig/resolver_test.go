package comfig

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestEnvResolver(t *testing.T) {
	t.Run("finds environment variable", func(t *testing.T) {
		var (
			key   = "test"
			value = "hello"
		)

		t.Setenv(key, value)
		got, err := NewEnvResolver().Resolve(context.Background(), key)
		if err != nil {
			t.Fatal(err)
		}
		if got != value {
			t.Fatalf("got = %q", got)
		}
	})

	t.Run("errors on missing environment variable", func(t *testing.T) {
		if _, err := NewEnvResolver().Resolve(context.Background(), "missing_environment_variable"); err == nil {
			t.Fatal("expected error for missing environment variable")
		}
	})

	t.Run("uses overridden prefix", func(t *testing.T) {
		resolver := NewEnvResolver(WithPrefixOverride("secret"))

		if got := resolver.Prefix(); got != "secret" {
			t.Fatalf("got prefix %q, want %q", got, "secret")
		}
	})
}

func TestFileResolver(t *testing.T) {
	t.Run("reads secret", func(t *testing.T) {
		value := "hello from file"

		dir := t.TempDir()
		file := filepath.Join(dir, "secret")
		if err := os.WriteFile(file, []byte(value), 0o600); err != nil {
			t.Fatal(err)
		}
		got, err := NewFileResolver().Resolve(context.Background(), file)
		if err != nil {
			t.Fatal(err)
		}
		if got != value {
			t.Fatalf("got = %q", got)
		}
	})

	t.Run("errors on missing file/directory", func(t *testing.T) {
		dir := t.TempDir()
		got, err := NewFileResolver().Resolve(context.Background(), filepath.Join(dir, "missing_file"))
		if err == nil {
			t.Fatalf("expected error for missing file")
		}
		if got != "" {
			t.Fatalf("expected \"\" got = %q", got)
		}
	})

	t.Run("uses overridden prefix", func(t *testing.T) {
		resolver := NewFileResolver(WithPrefixOverride("secret-file"))

		if got := resolver.Prefix(); got != "secret-file" {
			t.Fatalf("got prefix %q, want %q", got, "secret-file")
		}
	})
}
