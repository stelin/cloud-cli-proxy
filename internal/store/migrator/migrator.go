package migrator

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
)

func RunMigrations(ctx context.Context, db *pgxpool.Pool, dir string) error {
	if _, err := db.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			filename TEXT PRIMARY KEY,
			applied_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)
	`); err != nil {
		return fmt.Errorf("ensure schema_migrations: %w", err)
	}

	entries, err := filepath.Glob(filepath.Join(dir, "*.sql"))
	if err != nil {
		return fmt.Errorf("discover migrations: %w", err)
	}

	sort.Strings(entries)

	for _, path := range entries {
		filename := filepath.Base(path)
		var applied bool
		if err := db.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM schema_migrations WHERE filename = $1)`, filename).Scan(&applied); err != nil {
			return fmt.Errorf("check migration %s: %w", filename, err)
		}
		if applied {
			continue
		}

		contents, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("read migration %s: %w", filename, err)
		}

		tx, err := db.Begin(ctx)
		if err != nil {
			return fmt.Errorf("begin migration %s: %w", filename, err)
		}

		statements := strings.TrimSpace(string(contents))
		if statements != "" {
			if _, err := tx.Exec(ctx, statements); err != nil {
				_ = tx.Rollback(ctx)
				return fmt.Errorf("apply migration %s: %w", filename, err)
			}
		}

		if _, err := tx.Exec(ctx, `INSERT INTO schema_migrations (filename) VALUES ($1)`, filename); err != nil {
			_ = tx.Rollback(ctx)
			return fmt.Errorf("record migration %s: %w", filename, err)
		}

		if err := tx.Commit(ctx); err != nil {
			return fmt.Errorf("commit migration %s: %w", filename, err)
		}
	}

	return nil
}
