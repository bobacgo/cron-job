package database

import (
	"database/sql"
	"fmt"
	"log/slog"
	"maps"
	"slices"
)

const ComponentName = "database"

func withPrefix(prefix, format string, msgs ...any) string {
	format = "[" + prefix + "] " + format
	return fmt.Sprintf(format, msgs...)
}

const defaultInstanceKey = "default"

// 多数据源管理
type DBManager map[string]*sql.DB

func NewDBManager(cfgKV map[string]DialectorConfig) (DBManager, error) {
	if _, ok := cfgKV[defaultInstanceKey]; !ok {
		return nil, fmt.Errorf("not found default instance, must be has default")
	}
	dbs := make(DBManager, len(cfgKV))
	for k, cfg := range cfgKV {
		var err error
		if dbs[k], err = NewDB(cfg.Driver, cfg.Source, cfg.Config); err != nil {
			return nil, fmt.Errorf("k = %s , init err: %v", k, err)
		}
	}
	keys := slices.Collect(maps.Keys(dbs))
	slices.Sort(keys)
	slog.Info(withPrefix(ComponentName, "instances object %+q", keys))
	return dbs, nil
}

func (m DBManager) Default() *sql.DB {
	return m[defaultInstanceKey]
}

func (m DBManager) Get(k string) *sql.DB {
	return m[k]
}

type DialectorConfig struct {
	Driver string
	Source string
	Config Config
}

// DriverOpenFunc 驱动打开函数
// 输入 dsn，输出驱动名
type DriverOpenFunc func(dsn string) string

func DialectorMap(drivers []DriverOpenFunc, cfgMap map[string]Config) map[string]DialectorConfig {
	dialectorMap := make(map[string]DialectorConfig, len(drivers))

	driverMap := map[string]DriverOpenFunc{MySQLDriverName: func(_ string) string { return MySQLDriverName }} // 默认提供 mysql 驱动
	for _, d := range drivers {
		name := d("") // 驱动名
		driverMap[name] = d
	}

	driverKeys := slices.Collect(maps.Keys(driverMap))
	slices.Sort(driverKeys)
	slog.Info(withPrefix(ComponentName, "support driver %+q", driverKeys))

	for k, c := range cfgMap {
		openFunc, ok := driverMap[c.Driver]
		if !ok {
			slog.Warn(withPrefix(ComponentName, "driver not found, Please check the configuration file"), "driver", c.Driver)
			continue
		}
		dialectorMap[k] = DialectorConfig{Driver: openFunc(c.Source), Source: c.Source, Config: c}
	}
	return dialectorMap
}
