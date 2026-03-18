package handler

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	jobapp "github.com/bobacgo/cron-job/internal/app/job"
	jobdomain "github.com/bobacgo/cron-job/internal/domain/job"
)

type JobHandler struct {
	service *jobapp.Service
}

func NewJobHandler(service *jobapp.Service) *JobHandler {
	return &JobHandler{service: service}
}

func (h *JobHandler) Service() *jobapp.Service {
	return h.service
}

func (h *JobHandler) List(w http.ResponseWriter, r *http.Request) {
	items, err := h.service.List(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(items)
}

func (h *JobHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req createJobRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	job := jobdomain.Job{
		Name:        req.Name,
		Description: req.Description,
		Enabled:     req.Enabled,
		Schedule: jobdomain.Schedule{
			Interval: time.Duration(req.IntervalSeconds) * time.Second,
			Cron:     req.Cron,
			TimeZone: req.TimeZone,
		},
		Executor:          req.toExecutorSpec(),
		ConcurrencyPolicy: jobdomain.ConcurrencyPolicy(req.ConcurrencyPolicy),
		RetryPolicy: jobdomain.RetryPolicy{
			MaxRetries:      req.MaxRetries,
			InitialBackoff:  time.Duration(req.InitialBackoffSeconds) * time.Second,
			MaxBackoff:      time.Duration(req.MaxBackoffSeconds) * time.Second,
			BackoffMultiple: req.BackoffMultiple,
		},
	}

	created, err := h.service.Create(r.Context(), job, req.DependencyIDs)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(created)
}

func (h *JobHandler) Get(w http.ResponseWriter, r *http.Request, id string) {
	detail, err := h.service.GetDetail(r.Context(), id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(detail)
}

func (h *JobHandler) Trigger(w http.ResponseWriter, r *http.Request, id string) {
	run, err := h.service.Trigger(r.Context(), id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(run)
}

func (h *JobHandler) Pause(w http.ResponseWriter, r *http.Request, id string) {
	job, err := h.service.Pause(r.Context(), id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(job)
}

func (h *JobHandler) Resume(w http.ResponseWriter, r *http.Request, id string) {
	job, err := h.service.Resume(r.Context(), id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(job)
}

func (h *JobHandler) HandleByID(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/v1/jobs/")
	if path == "" {
		http.NotFound(w, r)
		return
	}
	parts := strings.Split(strings.Trim(path, "/"), "/")
	id := parts[0]
	if len(parts) == 2 && parts[1] == "trigger" {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		h.Trigger(w, r, id)
		return
	}
	if len(parts) == 2 && parts[1] == "pause" {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		h.Pause(w, r, id)
		return
	}
	if len(parts) == 2 && parts[1] == "resume" {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		h.Resume(w, r, id)
		return
	}
	if len(parts) == 1 && r.Method == http.MethodGet {
		h.Get(w, r, id)
		return
	}
	http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
}

type createJobRequest struct {
	Name                  string   `json:"name"`
	Description           string   `json:"description"`
	Enabled               bool     `json:"enabled"`
	Cron                  string   `json:"cron"`
	TimeZone              string   `json:"time_zone"`
	IntervalSeconds       int      `json:"interval_seconds"`
	ExecutorType          string   `json:"executor_type"`
	ConcurrencyPolicy     string   `json:"concurrency_policy"`
	MaxRetries            int      `json:"max_retries"`
	InitialBackoffSeconds int      `json:"initial_backoff_seconds"`
	MaxBackoffSeconds     int      `json:"max_backoff_seconds"`
	BackoffMultiple       float64  `json:"backoff_multiple"`
	SDKProtocol           string   `json:"sdk_protocol"`
	SDKURL                string   `json:"sdk_url"`
	SDKMethod             string   `json:"sdk_method"`
	SDKTimeoutSeconds     int      `json:"sdk_timeout_seconds"`
	BinaryCommand         string   `json:"binary_command"`
	BinaryArgs            []string `json:"binary_args"`
	BinaryTimeoutSeconds  int      `json:"binary_timeout_seconds"`
	DependencyIDs         []string `json:"dependency_ids"`
	ShellScript           string   `json:"shell_script"`
	ShellShell            string   `json:"shell_shell"`
	ShellTimeoutSeconds   int      `json:"shell_timeout_seconds"`
}

func (r createJobRequest) toExecutorSpec() jobdomain.ExecutorSpec {
	if r.ExecutorType == string(jobdomain.ExecutorKindBinary) {
		return jobdomain.ExecutorSpec{
			Kind: jobdomain.ExecutorKindBinary,
			Binary: &jobdomain.BinaryTarget{
				Command: r.BinaryCommand,
				Args:    r.BinaryArgs,
				Timeout: time.Duration(r.BinaryTimeoutSeconds) * time.Second,
			},
		}
	}
	if r.ExecutorType == string(jobdomain.ExecutorKindShell) {
		sh := r.ShellShell
		if sh == "" {
			sh = "/bin/sh"
		}
		return jobdomain.ExecutorSpec{
			Kind: jobdomain.ExecutorKindShell,
			Shell: &jobdomain.ShellTarget{
				Script:  r.ShellScript,
				Shell:   sh,
				Timeout: time.Duration(r.ShellTimeoutSeconds) * time.Second,
			},
		}
	}

	method := r.SDKMethod
	if method == "" {
		method = http.MethodPost
	}
	return jobdomain.ExecutorSpec{
		Kind: jobdomain.ExecutorKindSDK,
		SDK: &jobdomain.SDKTarget{
			Protocol: r.SDKProtocol,
			URL:      r.SDKURL,
			Method:   method,
			Timeout:  time.Duration(r.SDKTimeoutSeconds) * time.Second,
		},
	}
}
