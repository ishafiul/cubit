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

	// Dynamic column migrations for existing SQLite databases
	var hasSourceType, hasInlineCode, hasSubdomain bool
	rows, err := db.QueryContext(context.Background(), "PRAGMA table_info(applications)")
	if err == nil {
		for rows.Next() {
			var cid int
			var colName, ctype string
			var notnull, pk int
			var dfltValue sql.NullString
			if err := rows.Scan(&cid, &colName, &ctype, &notnull, &dfltValue, &pk); err == nil {
				if colName == "source_type" {
					hasSourceType = true
				}
				if colName == "inline_code" {
					hasInlineCode = true
				}
				if colName == "subdomain" {
					hasSubdomain = true
				}
			}
		}
		rows.Close()

		if !hasSourceType {
			_, _ = db.ExecContext(context.Background(), "ALTER TABLE applications ADD COLUMN source_type TEXT NOT NULL DEFAULT 'git'")
		}
		if !hasInlineCode {
			_, _ = db.ExecContext(context.Background(), "ALTER TABLE applications ADD COLUMN inline_code TEXT")
		}
		if !hasSubdomain {
			_, _ = db.ExecContext(context.Background(), "ALTER TABLE applications ADD COLUMN subdomain TEXT NOT NULL DEFAULT ''")
		}
	}

	var hasBuildVersion bool
	depRows, err := db.QueryContext(context.Background(), "PRAGMA table_info(deployments)")
	if err == nil {
		for depRows.Next() {
			var cid int
			var colName, ctype string
			var notnull, pk int
			var dfltValue sql.NullString
			if err := depRows.Scan(&cid, &colName, &ctype, &notnull, &dfltValue, &pk); err == nil {
				if colName == "build_version" {
					hasBuildVersion = true
				}
			}
		}
		depRows.Close()

		if !hasBuildVersion {
			_, _ = db.ExecContext(context.Background(), "ALTER TABLE deployments ADD COLUMN build_version INTEGER NOT NULL DEFAULT 1")
		}
	}

	return db, nil
}
