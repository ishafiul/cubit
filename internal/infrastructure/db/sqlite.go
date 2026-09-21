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
	var hasSourceType, hasInlineCode, hasSubdomain, hasAutoDeploy bool
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
				if colName == "auto_deploy" {
					hasAutoDeploy = true
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
		if !hasAutoDeploy {
			_, _ = db.ExecContext(context.Background(), "ALTER TABLE applications ADD COLUMN auto_deploy INTEGER NOT NULL DEFAULT 1")
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

	// Ensure that only each application's active_deployment_id retains 'active' status.
	// Any older deployments previously marked 'active' are transitioned to 'superseded'.
	_, _ = db.ExecContext(context.Background(), `
		UPDATE deployments
		SET status = 'superseded'
		WHERE status = 'active'
		  AND id NOT IN (
			SELECT active_deployment_id
			FROM applications
			WHERE active_deployment_id IS NOT NULL AND active_deployment_id != ''
		  )
	`)

	// Dynamic migration for nodes.is_protected
	var hasIsProtected bool
	nodeRows, err := db.QueryContext(context.Background(), "PRAGMA table_info(nodes)")
	if err == nil {
		for nodeRows.Next() {
			var cid int
			var colName, ctype string
			var notnull, pk int
			var dfltValue sql.NullString
			if err := nodeRows.Scan(&cid, &colName, &ctype, &notnull, &dfltValue, &pk); err == nil {
				if colName == "is_protected" {
					hasIsProtected = true
				}
			}
		}
		nodeRows.Close()

		if !hasIsProtected {
			_, _ = db.ExecContext(context.Background(), "ALTER TABLE nodes ADD COLUMN is_protected INTEGER NOT NULL DEFAULT 0")
		}
	}

	// Always ensure the main cluster seed node is marked protected
	_, _ = db.ExecContext(context.Background(), "UPDATE nodes SET is_protected = 1 WHERE name = 'worker-node-01'")

	// Ensure r2_buckets table exists
	_, _ = db.ExecContext(context.Background(), `
		CREATE TABLE IF NOT EXISTS r2_buckets (
			name TEXT PRIMARY KEY,
			created_at TIMESTAMP NOT NULL
		);
	`)

	return db, nil
}
