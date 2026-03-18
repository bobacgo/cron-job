package config

import "os"

type Config struct {
	HTTPAddr string
	LogDir   string
	DBPath   string
}

func Load() Config {
	addr := os.Getenv("HTTP_ADDR")
	if addr == "" {
		addr = ":8080"
	}
	logDir := os.Getenv("LOG_DIR")
	if logDir == "" {
		logDir = "data/logs"
	}
	dbPath := os.Getenv("DB_PATH")
	if dbPath == "" {
		dbPath = "data/cron-job.db"
	}

	return Config{HTTPAddr: addr, LogDir: logDir, DBPath: dbPath}
}
