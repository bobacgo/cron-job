package database

import (
	"fmt"

	"github.com/bobacgo/cron-job/kit/types"
)

type Config struct {
	Driver string `yaml:"driver"` // 驱动名称
	// Data Source Name
	Dsn           string         `yaml:"dsn"`           // root:****@tcp(127.0.0.1:3306)/test
	SlowThreshold types.Duration `yaml:"slowThreshold"` // 慢日志阈值
	MaxOpenConn   int            `yaml:"maxOpenConn"`   // 最大连接数 (高并发 500，低并发 100)
	MaxIdleConn   int            `yaml:"maxIdleConn"`   // 最大空闲连接数 (高并发 50，低并发 10)
	MaxLifeTime   types.Duration `yaml:"maxLifeTime"`   // 最大连接时间 (高并发 1h，低并发 30m)
	MaxIdleTime   types.Duration `yaml:"maxIdleTime"`   // 最大空闲时间 (高并发 15m，低并发 10m)
}

func (c Config) Validate() error {
	if c.Driver == "" {
		return fmt.Errorf("driver is required")
	}
	if c.Dsn == "" {
		return fmt.Errorf("dsn is required")
	}
	if err := c.SlowThreshold.Check(); err != nil {
		return fmt.Errorf("slowThreshold invalid: %w", err)
	}
	if err := c.MaxLifeTime.Check(); err != nil {
		return fmt.Errorf("maxLifeTime invalid: %w", err)
	}
	if err := c.MaxIdleTime.Check(); err != nil {
		return fmt.Errorf("maxIdleTime invalid: %w", err)
	}
	return nil
}
