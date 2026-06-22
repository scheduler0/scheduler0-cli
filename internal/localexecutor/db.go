package localexecutor

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	_ "modernc.org/sqlite"
)

const dbFilename = "local-executor.db"

// openDB opens (or creates) the SQLite database at ~/.scheduler0/local-executor.db.
func openDB() (*sql.DB, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get home dir: %w", err)
	}

	dbDir := filepath.Join(homeDir, ".scheduler0")
	if err := os.MkdirAll(dbDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create ~/.scheduler0 dir: %w", err)
	}

	dbPath := filepath.Join(dbDir, dbFilename)
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open sqlite db: %w", err)
	}

	// Single writer, many readers is fine for this use-case.
	db.SetMaxOpenConns(1)

	if err := migrate(db); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("failed to migrate local executor db: %w", err)
	}

	return db, nil
}

func migrate(db *sql.DB) error {
	_, err := db.Exec(`
	CREATE TABLE IF NOT EXISTS jobs_cache (
		id             INTEGER PRIMARY KEY,
		executor_id    INTEGER NOT NULL,
		project_id     INTEGER,
		account_id     INTEGER,
		spec           TEXT NOT NULL,
		data           TEXT,
		timezone       TEXT,
		timezone_offset INTEGER,
		start_date     TEXT,
		end_date       TEXT,
		retry_max      INTEGER DEFAULT 0,
		status         TEXT,
		updated_at     DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS executions_uncommitted (
		id                 INTEGER PRIMARY KEY AUTOINCREMENT,
		job_id             INTEGER NOT NULL,
		unique_id          TEXT NOT NULL,
		state              INTEGER NOT NULL,   -- 0=scheduled, 1=success, 2=failed
		last_execution_time TEXT,
		next_execution_time TEXT,
		execution_version  INTEGER NOT NULL DEFAULT 0,
		job_queue_version  INTEGER NOT NULL DEFAULT 0,
		created_at         DATETIME DEFAULT CURRENT_TIMESTAMP,
		reported           INTEGER NOT NULL DEFAULT 0  -- 1 once successfully reported to server
	);

	CREATE INDEX IF NOT EXISTS idx_executions_unreported ON executions_uncommitted(reported, created_at);
	`)
	return err
}
