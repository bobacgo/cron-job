package handler

import (
	"encoding/json"
	"net/http"
	"strings"

	jobapp "github.com/bobacgo/cron-job/internal/app/job"
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
	if len(parts) != 2 || parts[1] != "logs" || r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	content, err := h.service.ReadRunLog(r.Context(), parts[0])
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{"run_id": parts[0], "content": content})
}
