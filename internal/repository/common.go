package repository

import (
	"errors"
	"time"
)

var ErrNotFound = errors.New("not found")

type scanFunc func(dest ...any) error

func formatTime(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.UTC().Format(time.RFC3339Nano)
}

func parseTime(raw string) time.Time {
	if raw == "" {
		return time.Time{}
	}
	parsed, err := time.Parse(time.RFC3339Nano, raw)
	if err != nil {
		return time.Time{}
	}
	return parsed.UTC()
}

func boolToInt(v bool) int {
	if v {
		return 1
	}
	return 0
}
