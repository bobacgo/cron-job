package config

import (
	"os"

	"gopkg.in/yaml.v3"
)

type Config struct {
	HTTPAddr string `yaml:"http_addr"`
	LogDir   string `yaml:"log_dir"`
	DBDSN    string `yaml:"db_dsn"`
}

// Load reads configuration from a YAML file (default: config.yaml, override with CONFIG_FILE env var).
// Environment variables HTTP_ADDR, LOG_DIR, and DB_DSN take precedence over file values.
func Load() Config {
	cfg := Config{
		HTTPAddr: ":8080",
		LogDir:   "data/logs",
		DBDSN:    "root:root@tcp(127.0.0.1:3306)/cron_job?charset=utf8mb4&parseTime=true&loc=Local",
	}

	path := os.Getenv("CONFIG_FILE")
	if path == "" {
		path = "config.yaml"
	}
	if data, err := os.ReadFile(path); err == nil {
		_ = yaml.Unmarshal(data, &cfg)
	}

	// Environment variables override YAML file values.
	if v := os.Getenv("HTTP_ADDR"); v != "" {
		cfg.HTTPAddr = v
	}
	if v := os.Getenv("LOG_DIR"); v != "" {
		cfg.LogDir = v
	}
	if v := os.Getenv("DB_DSN"); v != "" {
		cfg.DBDSN = v
	} else if v := os.Getenv("DB_PATH"); v != "" {
		// Backward compatibility: allow DB_PATH as alias of DB_DSN.
		cfg.DBDSN = v
	}

	return cfg
}
