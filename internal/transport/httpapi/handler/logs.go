package handler

import (
	"encoding/json"
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
	if len(parts) == 2 && parts[1] == "logs" && r.Method == http.MethodGet {
		stream := strings.TrimSpace(r.URL.Query().Get("stream"))
		content, err := h.service.ReadRunLogStream(r.Context(), parts[0], stream)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"run_id": parts[0], "stream": stream, "content": content})
		return
	}
	if len(parts) == 2 && parts[1] == "cancel" && r.Method == http.MethodPost {
		run, err := h.service.CancelRun(r.Context(), parts[0])
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(run)
		return
	}
	if len(parts) == 2 && parts[1] == "retry" && r.Method == http.MethodPost {
		run, err := h.service.RetryRun(r.Context(), parts[0])
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(run)
		return
	}
	http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	return
}

func (h *RunLogHandler) Search(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
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
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{"items": items, "count": len(items)})
}
