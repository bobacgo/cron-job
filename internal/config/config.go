package config

import "os"

type Config struct {
	HTTPAddr string
	LogDir   string
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

	return Config{HTTPAddr: addr, LogDir: logDir}
}
