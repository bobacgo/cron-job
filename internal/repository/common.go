package repository

import (
	"errors"
)

var ErrNotFound = errors.New("not found")

type scanFunc func(dest ...any) error

func formatTime(ts int64) int64 {
	return ts
}

func parseTime(raw int64) int64 {
	return raw
}

func boolToInt(v bool) int {
	if v {
		return 1
	}
	return 0
}
