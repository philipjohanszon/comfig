package comfig

import "testing"

func TestEnvironment(t *testing.T) {
	t.Run("uses lowercase environment variable when set", func(t *testing.T) {
		t.Setenv("env", "staging")
		t.Setenv("ENV", "production")

		if got := Environment(); got != "staging" {
			t.Fatalf("got environment %q, want %q", got, "staging")
		}
	})

	t.Run("uses uppercase environment variable when lowercase is empty", func(t *testing.T) {
		t.Setenv("env", "")
		t.Setenv("ENV", "production")

		if got := Environment(); got != "production" {
			t.Fatalf("got environment %q, want %q", got, "production")
		}
	})

	t.Run("defaults to local when environment variables are empty", func(t *testing.T) {
		t.Setenv("env", "")
		t.Setenv("ENV", "")

		if got := Environment(); got != "local" {
			t.Fatalf("got environment %q, want %q", got, "local")
		}
	})
}
