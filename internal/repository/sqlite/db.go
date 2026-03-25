package sqlite

import (
	"database/sql"
	"fmt"

	"github.com/bobacgo/cron-job/kit/database"
)

func Open(dsn string) (*sql.DB, error) {
	db, err := database.OpenMySQL(dsn, database.MySQLOptions{})
	if err != nil {
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
			id VARCHAR(64) PRIMARY KEY,
			name VARCHAR(255) NOT NULL,
			description TEXT NOT NULL,
			enabled TINYINT(1) NOT NULL,
			schedule_cron VARCHAR(255) NOT NULL,
			schedule_interval_seconds BIGINT NOT NULL,
			schedule_time_zone VARCHAR(128) NOT NULL,
			schedule_starting_deadline_seconds INT NOT NULL,
			executor_kind VARCHAR(32) NOT NULL,
			sdk_protocol VARCHAR(32) NOT NULL,
			sdk_url TEXT NOT NULL,
			sdk_method VARCHAR(32) NOT NULL,
			sdk_timeout_seconds BIGINT NOT NULL,
			binary_command TEXT NOT NULL,
			binary_args_json LONGTEXT NOT NULL,
			binary_timeout_seconds BIGINT NOT NULL,
			shell_script LONGTEXT NOT NULL,
			shell_shell VARCHAR(255) NOT NULL DEFAULT '',
			shell_timeout_seconds BIGINT NOT NULL DEFAULT 0,
			retry_max_retries INT NOT NULL,
			retry_initial_backoff_seconds BIGINT NOT NULL,
			retry_max_backoff_seconds BIGINT NOT NULL,
			retry_backoff_multiple REAL NOT NULL,
			concurrency_policy VARCHAR(32) NOT NULL,
			next_run_at VARCHAR(64) NOT NULL,
			last_run_at VARCHAR(64) NOT NULL,
			last_success_at VARCHAR(64) NOT NULL,
			created_at VARCHAR(64) NOT NULL,
			updated_at VARCHAR(64) NOT NULL
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;`,
		`CREATE TABLE IF NOT EXISTS job_runs (
			id VARCHAR(64) PRIMARY KEY,
			job_id VARCHAR(64) NOT NULL,
			scheduled_at VARCHAR(64) NOT NULL,
			started_at VARCHAR(64) NOT NULL,
			finished_at VARCHAR(64) NOT NULL,
			status VARCHAR(32) NOT NULL,
			attempt INT NOT NULL,
			trigger_type VARCHAR(32) NOT NULL,
			message TEXT NOT NULL,
			created_at VARCHAR(64) NOT NULL,
			updated_at VARCHAR(64) NOT NULL,
			dedup_key VARCHAR(255) NOT NULL UNIQUE,
			CONSTRAINT fk_job_runs_job FOREIGN KEY (job_id) REFERENCES jobs(id)
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;`,
		`CREATE TABLE IF NOT EXISTS dependencies (
			job_id VARCHAR(64) NOT NULL,
			depends_on_job_id VARCHAR(64) NOT NULL,
			PRIMARY KEY (job_id, depends_on_job_id),
			CONSTRAINT fk_dependencies_job FOREIGN KEY (job_id) REFERENCES jobs(id),
			CONSTRAINT fk_dependencies_dep FOREIGN KEY (depends_on_job_id) REFERENCES jobs(id)
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;`,
	}
	for _, stmt := range statements {
		if _, err := db.Exec(stmt); err != nil {
			return err
		}
	}
	if err := ensureJobShellColumns(db); err != nil {
		return err
	}
	if err := ensureIndex(db, "job_runs", "idx_job_runs_job_id", "job_id"); err != nil {
		return err
	}
	if err := ensureIndex(db, "job_runs", "idx_job_runs_status", "status"); err != nil {
		return err
	}

	return nil
}

func ensureIndex(db *sql.DB, table, indexName, column string) error {
	var cnt int
	if err := db.QueryRow(`
SELECT COUNT(1)
FROM information_schema.statistics
WHERE table_schema = DATABASE() AND table_name = ? AND index_name = ?
`, table, indexName).Scan(&cnt); err != nil {
		return err
	}
	if cnt > 0 {
		return nil
	}
	_, err := db.Exec(fmt.Sprintf("CREATE INDEX %s ON %s(%s)", indexName, table, column))
	return err
}

func ensureJobShellColumns(db *sql.DB) error {
	if err := ensureColumn(db, "jobs", "shell_script", `ALTER TABLE jobs ADD COLUMN shell_script LONGTEXT NULL`); err != nil {
		return err
	}
	if err := ensureColumn(db, "jobs", "shell_shell", `ALTER TABLE jobs ADD COLUMN shell_shell VARCHAR(255) NOT NULL DEFAULT ''`); err != nil {
		return err
	}
	if err := ensureColumn(db, "jobs", "shell_timeout_seconds", `ALTER TABLE jobs ADD COLUMN shell_timeout_seconds BIGINT NOT NULL DEFAULT 0`); err != nil {
		return err
	}
	if _, err := db.Exec(`UPDATE jobs SET shell_script = '' WHERE shell_script IS NULL`); err != nil {
		return err
	}
	_, err := db.Exec(`ALTER TABLE jobs MODIFY COLUMN shell_script LONGTEXT NOT NULL`)
	return err
}

func ensureColumn(db *sql.DB, table, column, ddl string) error {
	var cnt int
	if err := db.QueryRow(`
SELECT COUNT(1)
FROM information_schema.columns
WHERE table_schema = DATABASE() AND table_name = ? AND column_name = ?
`, table, column).Scan(&cnt); err != nil {
		return err
	}
	if cnt > 0 {
		return nil
	}
	_, err := db.Exec(ddl)
	return err
}
