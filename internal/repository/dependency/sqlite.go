package dependency

import (
	"context"
	"database/sql"

	dependencydomain "github.com/bobacgo/cron-job/internal/domain/dependency"
)

type SQLiteRepository struct {
	db *sql.DB
}

func NewSQLiteRepository(db *sql.DB) *SQLiteRepository {
	return &SQLiteRepository{db: db}
}

func (r *SQLiteRepository) Replace(ctx context.Context, jobID string, edges []dependencydomain.Edge) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	if _, err := tx.ExecContext(ctx, `DELETE FROM dependencies WHERE job_id = ?`, jobID); err != nil {
		return err
	}
	for _, edge := range edges {
		if _, err := tx.ExecContext(ctx, `INSERT INTO dependencies (job_id, depends_on_job_id) VALUES (?, ?)`, edge.JobID, edge.DependsOnJobID); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (r *SQLiteRepository) ListByJob(ctx context.Context, jobID string) ([]dependencydomain.Edge, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT job_id, depends_on_job_id FROM dependencies WHERE job_id = ? ORDER BY depends_on_job_id ASC`, jobID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]dependencydomain.Edge, 0)
	for rows.Next() {
		var edge dependencydomain.Edge
		if err := rows.Scan(&edge.JobID, &edge.DependsOnJobID); err != nil {
			return nil, err
		}
		items = append(items, edge)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

func (r *SQLiteRepository) ListAll(ctx context.Context) ([]dependencydomain.Edge, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT job_id, depends_on_job_id FROM dependencies ORDER BY job_id ASC, depends_on_job_id ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]dependencydomain.Edge, 0)
	for rows.Next() {
		var edge dependencydomain.Edge
		if err := rows.Scan(&edge.JobID, &edge.DependsOnJobID); err != nil {
			return nil, err
		}
		items = append(items, edge)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}
