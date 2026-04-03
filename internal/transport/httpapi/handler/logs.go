package handler

import (
	"net/http"
	"strconv"
	"strings"

	jobapp "github.com/bobacgo/cron-job/internal/app/job"
	"github.com/bobacgo/cron-job/internal/repository"
)

type RunLogHandler struct {
	service *jobapp.Service
}

func NewRunLogHandler(service *jobapp.Service) *RunLogHandler {
	return &RunLogHandler{service: service}
}

func (h *RunLogHandler) Handle(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/v1/job-runs/")
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) != 2 {
		writeError(w, http.StatusMethodNotAllowed, http.ErrNotSupported)
		return
	}

	switch {
	case parts[1] == "logs" && r.Method == http.MethodGet:
		h.Logs(w, r, parts[0])
		return
	case parts[1] == "cancel" && r.Method == http.MethodPost:
		h.Cancel(w, r, parts[0])
		return
	case parts[1] == "retry" && r.Method == http.MethodPost:
		h.Retry(w, r, parts[0])
		return
	default:
		writeError(w, http.StatusMethodNotAllowed, http.ErrNotSupported)
		return
	}
}

func (h *RunLogHandler) Logs(w http.ResponseWriter, r *http.Request, runID string) {
	stream := strings.TrimSpace(r.URL.Query().Get("stream"))
	content, err := h.service.ReadRunLogStream(r.Context(), runID, stream)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSONStatus(w, http.StatusOK, map[string]any{"run_id": runID, "stream": stream, "content": content})
}

func (h *RunLogHandler) Cancel(w http.ResponseWriter, r *http.Request, runID string) {
	run, err := h.service.CancelRun(r.Context(), runID)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	writeJSONStatus(w, http.StatusOK, run)
}

func (h *RunLogHandler) Retry(w http.ResponseWriter, r *http.Request, runID string) {
	run, err := h.service.RetryRun(r.Context(), runID)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	writeJSONStatus(w, http.StatusCreated, run)
}

func (h *RunLogHandler) Search(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, http.ErrNotSupported)
		return
	}
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	items, err := h.service.SearchRunLogs(r.Context(), repository.LogQuery{
		RunID:    strings.TrimSpace(r.URL.Query().Get("run_id")),
		Stream:   strings.TrimSpace(r.URL.Query().Get("stream")),
		Contains: strings.TrimSpace(r.URL.Query().Get("q")),
		Limit:    limit,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSONStatus(w, http.StatusOK, map[string]any{"items": items, "count": len(items)})
}
