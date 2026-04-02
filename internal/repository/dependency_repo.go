package repository

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	dependencydomain "github.com/bobacgo/cron-job/internal/domain/dependency"
	"github.com/bobacgo/cron-job/kit/database"
)

type dependencyRepo struct{ db *sql.DB }

var dependencyFields = []string{
	"job_id",
	"depends_on_job_id",
}

func (r *dependencyRepo) Replace(ctx context.Context, jobID string, edges []dependencydomain.Edge) error {
	tx := database.Tx(func(ctx context.Context, tx *sql.Tx) error {
		if _, err := tx.ExecContext(ctx, `DELETE FROM dependencies WHERE job_id = ?`, jobID); err != nil {
			return fmt.Errorf("delete dependencies: %w", err)
		}

		cols := strings.Join(dependencyFields, ", ")
		placeholders := strings.Trim(strings.Repeat("?, ", len(dependencyFields)), ", ")

		for _, edge := range edges {
			if _, err := tx.ExecContext(ctx,
				fmt.Sprintf(`INSERT INTO dependencies (%s) VALUES (%s)`, cols, placeholders),
				edge.JobID,
				edge.DependsOnJobID); err != nil {
				return fmt.Errorf("insert dependency (job_id: %s, depends_on_job_id: %s): %w", edge.JobID, edge.DependsOnJobID, err)
			}
		}
		return nil
	})

	return tx.Exec(ctx, r.db)
}

// ListByJob 会返回 jobID 相关的所有依赖关系，即 edges 中的 JobID 都是 jobID
func (r *dependencyRepo) ListByJob(ctx context.Context, jobID string) ([]dependencydomain.Edge, error) {
	rows, err := r.db.QueryContext(ctx, fmt.Sprintf(`
	SELECT
		%s
	FROM dependencies
	WHERE job_id = ?
	ORDER BY depends_on_job_id ASC
	`, strings.Join(dependencyFields, ", ")), jobID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return r.scanEdges(rows)
}

func (r *dependencyRepo) ListAll(ctx context.Context) ([]dependencydomain.Edge, error) {
	rows, err := r.db.QueryContext(ctx, fmt.Sprintf(`
	SELECT
		%s
	FROM dependencies
	`, strings.Join(dependencyFields, ", ")))
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
