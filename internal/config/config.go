package config

import "os"

type Config struct {
	HTTPAddr string
	LogDir   string
	DBDSN    string
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
	dbDSN := os.Getenv("DB_DSN")
	if dbDSN == "" {
		// Backward compatibility: allow DB_PATH as alias of DSN.
		dbDSN = os.Getenv("DB_PATH")
	}
	if dbDSN == "" {
		dbDSN = "root:root@tcp(127.0.0.1:3306)/cron_job?charset=utf8mb4&parseTime=true&loc=Local"
	}

	return Config{HTTPAddr: addr, LogDir: logDir, DBDSN: dbDSN}
}
