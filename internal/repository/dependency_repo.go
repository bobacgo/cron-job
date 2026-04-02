package repository

import (
	"context"
	"database/sql"
	"fmt"

	dependencydomain "github.com/bobacgo/cron-job/internal/domain/dependency"
	"github.com/bobacgo/cron-job/kit/database"
)

type dependencyRepo struct{ db *sql.DB }

func (r *dependencyRepo) Replace(ctx context.Context, jobID string, edges []dependencydomain.Edge) error {
	tx := database.Tx(func(ctx context.Context, tx *sql.Tx) error {
		if _, err := tx.ExecContext(ctx, `DELETE FROM dependencies WHERE job_id = ?`, jobID); err != nil {
			return fmt.Errorf("delete dependencies: %w", err)
		}
		for _, edge := range edges {
			if _, err := tx.ExecContext(ctx, `
			INSERT INTO dependencies (
				job_id,
				depends_on_job_id
			) VALUES (?, ?)
			`, edge.JobID, edge.DependsOnJobID); err != nil {
				return fmt.Errorf("insert dependency (job_id: %s, depends_on_job_id: %s): %w", edge.JobID, edge.DependsOnJobID, err)
			}
		}
		return nil
	})

	return tx.Exec(ctx, r.db)
}

// ListByJob 会返回 jobID 相关的所有依赖关系，即 edges 中的 JobID 都是 jobID
func (r *dependencyRepo) ListByJob(ctx context.Context, jobID string) ([]dependencydomain.Edge, error) {
	rows, err := r.db.QueryContext(ctx, `
	SELECT
		job_id,
		depends_on_job_id
	FROM dependencies
	WHERE job_id = ?
	ORDER BY depends_on_job_id ASC
	`, jobID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return r.scanEdges(rows)
}

func (r *dependencyRepo) ListAll(ctx context.Context) ([]dependencydomain.Edge, error) {
	rows, err := r.db.QueryContext(ctx, `
	SELECT
		job_id,
		depends_on_job_id
	FROM dependencies
	ORDER BY job_id ASC, depends_on_job_id ASC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return r.scanEdges(rows)
}

func (r *dependencyRepo) scanEdges(rows *sql.Rows) ([]dependencydomain.Edge, error) {
	items := make([]dependencydomain.Edge, 0)
	for rows.Next() {
		var edge dependencydomain.Edge
		if err := rows.Scan(&edge.JobID, &edge.DependsOnJobID); err != nil {
			return nil, err
		}
		items = append(items, edge)
	}
	return items, rows.Err()
}
