package runlog

import "time"

type Event struct {
	RunID      string
	Type       string
	Message    string
	OccurredAt time.Time
}

type LogRecord struct {
	RunID      string
	Stream     string
	Content    string
	OccurredAt time.Time
}
