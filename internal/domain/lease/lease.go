package lease

import "time"

type Lease struct {
	RunID       string
	HolderID    string
	ExpiresAt   time.Time
	HeartbeatAt time.Time
}

func (l Lease) Expired(now time.Time) bool {
	return !l.ExpiresAt.After(now)
}
