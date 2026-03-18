package protocol

import (
	"strings"
	"time"

	jobrundomain "github.com/bobacgo/cron-job/internal/domain/jobrun"
)

const (
	VersionV1         = "v1"
	CurrentVersion    = VersionV1
	MethodExecutorRun = "/cronjob.v1.Executor/Run"
)

var SupportedVersions = []string{VersionV1}

type ErrorCode string

const (
	ErrorCodeNone           ErrorCode = "NONE"
	ErrorCodeInvalidVersion ErrorCode = "INVALID_VERSION"
	ErrorCodeInvalidRequest ErrorCode = "INVALID_REQUEST"
	ErrorCodeUnauthorized   ErrorCode = "UNAUTHORIZED"
	ErrorCodeTimeout        ErrorCode = "TIMEOUT"
	ErrorCodeInternal       ErrorCode = "INTERNAL"
)

type RunRequest struct {
	ProtocolVersion   string    `json:"protocol_version"`
	SupportedVersions []string  `json:"supported_versions"`
	JobID             string    `json:"job_id"`
	JobName           string    `json:"job_name"`
	RunID             string    `json:"run_id"`
	ScheduledAt       time.Time `json:"scheduled_at"`
	Attempt           int       `json:"attempt"`
	TriggerType       string    `json:"trigger_type"`
}

type RunResponse struct {
	ProtocolVersion string    `json:"protocol_version"`
	Status          string    `json:"status"`
	ErrorCode       ErrorCode `json:"error_code"`
	Message         string    `json:"message"`
	Output          string    `json:"output"`
}

func IsSupportedVersion(v string) bool {
	for _, item := range SupportedVersions {
		if item == strings.TrimSpace(v) {
			return true
		}
	}
	return false
}

func NegotiateVersion(preferred string, candidates []string) (string, bool) {
	preferred = strings.TrimSpace(preferred)
	if preferred != "" && IsSupportedVersion(preferred) {
		return preferred, true
	}
	for _, candidate := range candidates {
		if IsSupportedVersion(candidate) {
			return candidate, true
		}
	}
	return "", false
}

func NormalizeStatus(raw string) jobrundomain.Status {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "", "ok", "succeeded", "success":
		return jobrundomain.StatusSucceeded
	case "failed", "error":
		return jobrundomain.StatusFailed
	case "timeout", "timedout":
		return jobrundomain.StatusTimedOut
	case "canceled", "cancelled":
		return jobrundomain.StatusCanceled
	default:
		return jobrundomain.StatusSucceeded
	}
}

func StatusFromResponse(resp RunResponse) jobrundomain.Status {
	if resp.ErrorCode != "" && resp.ErrorCode != ErrorCodeNone {
		switch resp.ErrorCode {
		case ErrorCodeTimeout:
			return jobrundomain.StatusTimedOut
		case ErrorCodeUnauthorized, ErrorCodeInvalidRequest, ErrorCodeInvalidVersion, ErrorCodeInternal:
			return jobrundomain.StatusFailed
		default:
			return jobrundomain.StatusFailed
		}
	}
	return NormalizeStatus(resp.Status)
}
