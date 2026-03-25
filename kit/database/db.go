package database

import (
	"database/sql"
	"fmt"

	_ "github.com/go-sql-driver/mysql"
)

const MySQLDriverName = "mysql"

type MySQLOptions struct {
	MaxOpenConn int
	MaxIdleConn int
}

// OpenMySQL 打开 MySQL 数据库连接，并校验连通性。
func OpenMySQL(dsn string, opts MySQLOptions) (*sql.DB, error) {
	db, err := sql.Open(MySQLDriverName, dsn)
	if err != nil {
		return nil, fmt.Errorf("open mysql db err: %w", err)
	}

	if opts.MaxOpenConn > 0 {
		db.SetMaxOpenConns(opts.MaxOpenConn)
	}
	if opts.MaxIdleConn > 0 {
		db.SetMaxIdleConns(opts.MaxIdleConn)
	}

	if err := db.Ping(); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("ping mysql db err: %w", err)
	}

	return db, nil
}

// NewDB 新建数据库连接
// 内部已自动 ping
func NewDB(driver, source string, conf Config) (*sql.DB, error) {
	db, err := sql.Open(driver, source)
	if err != nil {
		return nil, fmt.Errorf("open db err: %w", err)
	}

	if err := db.Ping(); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("ping db err: %w", err)
	}

	// 影响最大并发数。
	// 过大可能导致数据库负载过高，过小会限制并发性能。
	//一般设置在 100~500，具体根据数据库负载情况调整。
	db.SetMaxOpenConns(conf.MaxOpenConn) // 设置最大连接数
	// 控制保持在池中的空闲连接数。
	// 过大会浪费资源，过小可能导致频繁创建连接，增加延迟。
	// 典型范围是 10~50。
	db.SetMaxIdleConns(conf.MaxIdleConn) // 设置闲置连接数
	// 控制连接存活的最大时间，避免连接长时间占用资源导致 MySQL 关闭连接。
	// 建议设置 30min ~ 1h，防止连接泄露。
	db.SetConnMaxLifetime(conf.MaxLifeTime.TimeDuration()) // 连接的最大可复用时间
	// 控制空闲连接的最长时间，防止长期空闲的连接占用资源。
	// 典型值 10min，根据业务需求调整。
	db.SetConnMaxIdleTime(conf.MaxIdleTime.TimeDuration()) // 空闲连接的最大生存时间
	return db, nil
}
