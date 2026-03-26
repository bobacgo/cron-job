package database

import (
	"database/sql"
	"fmt"
	"log/slog"
	"maps"
	"slices"

	"github.com/bobacgo/cron-job/kit/types"
)

type DB struct {
	*sql.DB
}

func (d DB) Validate() error {
	if err := d.Ping(); err != nil {
		_ = d.DB.Close()
		return fmt.Errorf("ping db err: %w", err)
	}
	return nil
}

// 多数据源管理
func NewDBManager(cfgKV types.ConfigMap[Config]) (types.ConfigMap[DB], error) {
	dbs := make(types.ConfigMap[DB], len(cfgKV))
	for k, cfg := range cfgKV {
		var err error
		if dbs[k], err = NewDB(cfg); err != nil {
			return nil, fmt.Errorf("k = %s , init err: %v", k, err)
		}
	}
	keys := slices.Collect(maps.Keys(dbs))
	slices.Sort(keys)
	slog.Info("database instances object", "drivers", keys)
	return dbs, nil
}
