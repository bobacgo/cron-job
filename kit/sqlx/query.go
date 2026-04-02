package sqlx

import (
	"context"
	"database/sql"
	"log/slog"
	"time"
)

// QueryRowContext 封装慢查询日志，超过阈值时通过 slog.WarnContext 打印 SQL。
func (d *DB) QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row {
	start := time.Now()
	row := d.DB.QueryRowContext(ctx, query, args...)
	d.logSlow(ctx, start, query, args)
	return row
}

// QueryContext 封装慢查询日志。
func (d *DB) QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error) {
	start := time.Now()
	rows, err := d.DB.QueryContext(ctx, query, args...)
	d.logSlow(ctx, start, query, args)
	return rows, err
}

// ExecContext 封装慢查询日志。
func (d *DB) ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error) {
	start := time.Now()
	result, err := d.DB.ExecContext(ctx, query, args...)
	d.logSlow(ctx, start, query, args)
	return result, err
}

// logSlow 当执行时间超过 slowThreshold 时打印慢 SQL，阈值为 0 则不记录。
func (d *DB) logSlow(ctx context.Context, start time.Time, query string, args []any) {
	if d.slowThreshold <= 0 {
		return
	}
	elapsed := time.Since(start)
	if elapsed >= d.slowThreshold {
		slog.WarnContext(ctx, "slow query",
			slog.Duration("elapsed", elapsed),
			slog.String("sql", query),
			slog.Any("args", args),
		)
	}
}
