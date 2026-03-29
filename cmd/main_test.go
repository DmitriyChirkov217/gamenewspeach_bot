package main

import "testing"

func TestLocalhostFallbackDSN(t *testing.T) {
	t.Run("replaces docker host and default port", func(t *testing.T) {
		got, ok := localhostFallbackDSN("postgres://user:pass@db:5432/app?sslmode=disable")
		if !ok {
			t.Fatal("expected fallback dsn")
		}

		want := "postgres://user:pass@localhost:5433/app?sslmode=disable"
		if got != want {
			t.Fatalf("unexpected fallback dsn: got %q want %q", got, want)
		}
	})

	t.Run("keeps custom port", func(t *testing.T) {
		got, ok := localhostFallbackDSN("postgres://user:pass@db:6543/app?sslmode=disable")
		if !ok {
			t.Fatal("expected fallback dsn")
		}

		want := "postgres://user:pass@localhost:6543/app?sslmode=disable"
		if got != want {
			t.Fatalf("unexpected fallback dsn: got %q want %q", got, want)
		}
	})

	t.Run("returns false for non docker host", func(t *testing.T) {
		if _, ok := localhostFallbackDSN("postgres://user:pass@localhost:5432/app?sslmode=disable"); ok {
			t.Fatal("expected no fallback for localhost")
		}
	})
}
