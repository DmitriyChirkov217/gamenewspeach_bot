package storage

import (
	"strings"
	"testing"
)

func TestExtractGooseUp(t *testing.T) {
	content := strings.Join([]string{
		"-- +goose Up",
		"CREATE TABLE test (id INT);",
		"-- +goose Down",
		"DROP TABLE test;",
	}, "\n")

	got, err := extractGooseUp(content)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "CREATE TABLE test (id INT);" {
		t.Fatalf("unexpected up sql: %q", got)
	}
}

func TestExtractGooseUpStatementBlock(t *testing.T) {
	content := strings.Join([]string{
		"-- +goose Up",
		"-- +goose StatementBegin",
		"CREATE FUNCTION test() RETURNS void AS $$",
		"BEGIN",
		"  NULL;",
		"END;",
		"$$ LANGUAGE plpgsql;",
		"-- +goose StatementEnd",
	}, "\n")

	got, err := extractGooseUp(content)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(got, "CREATE FUNCTION test()") {
		t.Fatalf("unexpected up sql: %q", got)
	}
}
