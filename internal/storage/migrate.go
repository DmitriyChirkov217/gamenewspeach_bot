package storage

import (
	"context"
	"embed"
	"fmt"
	"path"
	"sort"
	"strings"

	"github.com/jmoiron/sqlx"
)

//go:embed migrations/*.sql
var migrationFiles embed.FS

func Migrate(ctx context.Context, db *sqlx.DB) error {
	conn, err := db.Connx(ctx)
	if err != nil {
		return err
	}
	defer conn.Close()

	if _, err := conn.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version VARCHAR(255) PRIMARY KEY,
			applied_at TIMESTAMP NOT NULL DEFAULT NOW()
		);
	`); err != nil {
		return fmt.Errorf("create schema_migrations: %w", err)
	}

	entries, err := migrationFiles.ReadDir("migrations")
	if err != nil {
		return fmt.Errorf("read migrations: %w", err)
	}

	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".sql") {
			names = append(names, entry.Name())
		}
	}
	sort.Strings(names)

	for _, name := range names {
		var applied bool
		if err := conn.GetContext(
			ctx,
			&applied,
			`SELECT EXISTS(SELECT 1 FROM schema_migrations WHERE version = $1)`,
			name,
		); err != nil {
			return fmt.Errorf("check migration %s: %w", name, err)
		}
		if applied {
			continue
		}

		raw, err := migrationFiles.ReadFile(path.Join("migrations", name))
		if err != nil {
			return fmt.Errorf("read migration %s: %w", name, err)
		}

		upSQL, err := extractGooseUp(string(raw))
		if err != nil {
			return fmt.Errorf("parse migration %s: %w", name, err)
		}

		tx, err := conn.BeginTxx(ctx, nil)
		if err != nil {
			return fmt.Errorf("begin migration %s: %w", name, err)
		}

		if _, err := tx.ExecContext(ctx, upSQL); err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("apply migration %s: %w", name, err)
		}

		if _, err := tx.ExecContext(
			ctx,
			`INSERT INTO schema_migrations(version) VALUES ($1)`,
			name,
		); err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("save migration %s: %w", name, err)
		}

		if err := tx.Commit(); err != nil {
			return fmt.Errorf("commit migration %s: %w", name, err)
		}
	}

	return nil
}

func extractGooseUp(content string) (string, error) {
	lines := strings.Split(content, "\n")
	var (
		inUp             bool
		inStatementBlock bool
		builder          strings.Builder
	)

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		switch trimmed {
		case "-- +goose Up":
			inUp = true
			continue
		case "-- +goose Down":
			inUp = false
			break
		case "-- +goose StatementBegin":
			if inUp {
				inStatementBlock = true
			}
			continue
		case "-- +goose StatementEnd":
			if inUp && inStatementBlock {
				inStatementBlock = false
				builder.WriteString("\n")
			}
			continue
		}

		if inUp && (inStatementBlock || !strings.HasPrefix(trimmed, "-- +goose")) {
			builder.WriteString(line)
			builder.WriteString("\n")
		}
	}

	upSQL := strings.TrimSpace(builder.String())
	if upSQL == "" {
		return "", fmt.Errorf("empty up migration")
	}

	return upSQL, nil
}
