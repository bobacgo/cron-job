package config

import (
	"os"

	"github.com/bobacgo/cron-job/kit/database"
	"github.com/bobacgo/cron-job/kit/types"
	"gopkg.in/yaml.v3"
)

// Config 是整个应用的配置结构体
// 和 config.yaml 的结构一一对应
type Config struct {
	HTTPAddr string                           `yaml:"http_addr"`
	LogDir   string                           `yaml:"log_dir"`
	Database types.ConfigMap[database.Config] `yaml:"database"`
}

func (c *Config) Validate() error {
	if err := c.Database.Validate(); err != nil {
		return err
	}

	// validate other fields
	return nil
}

func Load() (*Config, error) {
	cfg := &Config{}
	path := "config.yaml"
	if data, err := os.ReadFile(path); err == nil {
		_ = yaml.Unmarshal(data, cfg)
	}
	return cfg, cfg.Validate()
}
