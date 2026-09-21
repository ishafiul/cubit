package db

import (
	"context"
	"database/sql"
	_ "embed"
	"fmt"

	_ "modernc.org/sqlite"
)

//go:embed schema.sql
var schemaSQL string

// OpenSQLite opens or creates an embedded SQLite database configured for high performance in WAL mode.
func OpenSQLite(dsn string) (*sql.DB, error) {
	if dsn == "" {
		dsn = "file:cubit.db?_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)&_pragma=foreign_keys(ON)"
	}

	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open sqlite database: %w", err)
	}

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping sqlite database: %w", err)
	}

	// Apply schema migrations
	if _, err := db.ExecContext(context.Background(), schemaSQL); err != nil {
		return nil, fmt.Errorf("failed to execute schema migration: %w", err)
	}

	return db, nil
}
