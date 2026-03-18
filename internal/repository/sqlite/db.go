package sqlite

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	_ "modernc.org/sqlite"
)

func Open(path string) (*sql.DB, error) {
	if path == "" {
		return nil, fmt.Errorf("sqlite path is required")
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, err
	}
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}
	if _, err := db.Exec(`PRAGMA foreign_keys = ON;`); err != nil {
		_ = db.Close()
		return nil, err
	}
	if err := migrate(db); err != nil {
		_ = db.Close()
		return nil, err
	}
	return db, nil
}

func migrate(db *sql.DB) error {
	statements := []string{
		`CREATE TABLE IF NOT EXISTS jobs (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			description TEXT NOT NULL,
			enabled INTEGER NOT NULL,
			schedule_cron TEXT NOT NULL,
			schedule_interval_seconds INTEGER NOT NULL,
			schedule_time_zone TEXT NOT NULL,
			schedule_starting_deadline_seconds INTEGER NOT NULL,
			executor_kind TEXT NOT NULL,
			sdk_protocol TEXT NOT NULL,
			sdk_url TEXT NOT NULL,
			sdk_method TEXT NOT NULL,
			sdk_timeout_seconds INTEGER NOT NULL,
			binary_command TEXT NOT NULL,
			binary_args_json TEXT NOT NULL,
			binary_timeout_seconds INTEGER NOT NULL,
			retry_max_retries INTEGER NOT NULL,
			retry_initial_backoff_seconds INTEGER NOT NULL,
			retry_max_backoff_seconds INTEGER NOT NULL,
			retry_backoff_multiple REAL NOT NULL,
			concurrency_policy TEXT NOT NULL,
			next_run_at TEXT NOT NULL,
			last_run_at TEXT NOT NULL,
			last_success_at TEXT NOT NULL,
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL
		);`,
		`CREATE TABLE IF NOT EXISTS job_runs (
			id TEXT PRIMARY KEY,
			job_id TEXT NOT NULL,
			scheduled_at TEXT NOT NULL,
			started_at TEXT NOT NULL,
			finished_at TEXT NOT NULL,
			status TEXT NOT NULL,
			attempt INTEGER NOT NULL,
			trigger_type TEXT NOT NULL,
			message TEXT NOT NULL,
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL,
			dedup_key TEXT NOT NULL UNIQUE,
			FOREIGN KEY (job_id) REFERENCES jobs(id)
		);`,
		`CREATE INDEX IF NOT EXISTS idx_job_runs_job_id ON job_runs(job_id);`,
		`CREATE INDEX IF NOT EXISTS idx_job_runs_status ON job_runs(status);`,
		`CREATE TABLE IF NOT EXISTS dependencies (
			job_id TEXT NOT NULL,
			depends_on_job_id TEXT NOT NULL,
			PRIMARY KEY (job_id, depends_on_job_id),
			FOREIGN KEY (job_id) REFERENCES jobs(id),
			FOREIGN KEY (depends_on_job_id) REFERENCES jobs(id)
		);`,
	}
	for _, stmt := range statements {
		if _, err := db.Exec(stmt); err != nil {
			return err
		}
	}
	return nil
}
