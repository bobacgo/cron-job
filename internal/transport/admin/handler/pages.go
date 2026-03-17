package handler

import (
	"html/template"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	jobapp "github.com/bobacgo/cron-job/internal/app/job"
	jobdomain "github.com/bobacgo/cron-job/internal/domain/job"
	jobrundomain "github.com/bobacgo/cron-job/internal/domain/jobrun"
	jobrunrepo "github.com/bobacgo/cron-job/internal/repository/jobrun"
	logrepo "github.com/bobacgo/cron-job/internal/repository/log"
	"github.com/bobacgo/cron-job/internal/transport/admin/viewmodel"
)

type PageHandler struct {
	service   *jobapp.Service
	logs      logrepo.Repository
	runs      jobrunrepo.Repository
	templates *template.Template
}

func NewPageHandler(service *jobapp.Service, runs jobrunrepo.Repository, logs logrepo.Repository) *PageHandler {
	return &PageHandler{
		service:   service,
		logs:      logs,
		runs:      runs,
		templates: mustParseTemplates(),
	}
}

func (h *PageHandler) Dashboard(w http.ResponseWriter, r *http.Request) {
	lang := currentLang(r)
	dict := dictionary(lang)
	jobs, err := h.service.List(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	runs, err := h.runs.List(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	enabled := 0
	for _, item := range jobs {
		if item.Enabled {
			enabled++
		}
	}
	disabled := len(jobs) - enabled
	running := 0
	waiting := 0
	succeeded := 0
	failed := 0
	for _, run := range runs {
		switch run.Status {
		case jobrundomain.StatusRunning:
			running++
		case jobrundomain.StatusPending, jobrundomain.StatusReady, jobrundomain.StatusBlocked:
			waiting++
		case jobrundomain.StatusSucceeded:
			succeeded++
		case jobrundomain.StatusFailed, jobrundomain.StatusTimedOut, jobrundomain.StatusCanceled:
			failed++
		}
	}
	data := map[string]any{
		"Title": tr(dict, "nav_dashboard", "Dashboard"),
		"Dashboard": viewmodel.Dashboard{
			JobCount:      len(jobs),
			EnabledJobs:   enabled,
			DisabledJobs:  disabled,
			RunCount:      len(runs),
			RunningRuns:   running,
			WaitingRuns:   waiting,
			SucceededRuns: succeeded,
			FailedRuns:    failed,
			EnabledRate:   pct(enabled, len(jobs)),
			SuccessRate:   pct(succeeded, len(runs)),
			AttentionRate: pct(failed, len(runs)),
			GeneratedAt:   time.Now().UTC(),
		},
	}
	h.render(w, r, "dashboard", data)
}

func (h *PageHandler) Jobs(w http.ResponseWriter, r *http.Request) {
	lang := currentLang(r)
	dict := dictionary(lang)
	if r.Method == http.MethodPost {
		h.createJob(w, r)
		return
	}
	jobs, err := h.service.List(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	query := strings.TrimSpace(r.URL.Query().Get("q"))
	statusFilter := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("status")))
	executorFilter := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("executor")))
	filteredJobs := filterJobs(jobs, query, statusFilter, executorFilter)
	enabledCount, disabledCount, sdkCount, binaryCount := summarizeJobs(jobs)
	data := map[string]any{
		"Title":             tr(dict, "nav_jobs", "Jobs"),
		"Jobs":              viewmodel.JobItems(filteredJobs),
		"DependencyOptions": viewmodel.DependencyOptions(jobs),
		"FilterQuery":       query,
		"FilterStatus":      statusFilter,
		"FilterExecutor":    executorFilter,
		"VisibleJobs":       len(filteredJobs),
		"TotalJobs":         len(jobs),
		"EnabledCount":      enabledCount,
		"DisabledCount":     disabledCount,
		"SDKCount":          sdkCount,
		"BinaryCount":       binaryCount,
	}
	h.render(w, r, "jobs", data)
}

func (h *PageHandler) JobRoutes(w http.ResponseWriter, r *http.Request) {
	path := strings.Trim(strings.TrimPrefix(r.URL.Path, "/jobs/"), "/")
	if path == "" {
		http.Redirect(w, r, "/jobs", http.StatusSeeOther)
		return
	}
	parts := strings.Split(path, "/")
	id := parts[0]
	if len(parts) == 2 && parts[1] == "trigger" {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		h.triggerJob(w, r, id)
		return
	}
	if len(parts) == 2 && parts[1] == "pause" {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		h.pauseJob(w, r, id)
		return
	}
	if len(parts) == 2 && parts[1] == "resume" {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		h.resumeJob(w, r, id)
		return
	}
	if len(parts) == 1 && r.Method == http.MethodGet {
		h.jobDetail(w, r, id)
		return
	}
	http.NotFound(w, r)
}

func (h *PageHandler) render(w http.ResponseWriter, r *http.Request, name string, data map[string]any) {
	lang := resolveLang(w, r)
	dict := dictionary(lang)
	data["Lang"] = lang
	data["Dict"] = dict
	data["CurrentPath"] = r.URL.Path
	data["CurrentQuery"] = r.URL.RawQuery
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := h.templates.ExecuteTemplate(w, name, data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func mustParseTemplates() *template.Template {
	baseDir, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	paths := []string{
		filepath.Join(baseDir, "web/templates/layout/base.html"),
		filepath.Join(baseDir, "web/templates/pages/dashboard.html"),
		filepath.Join(baseDir, "web/templates/pages/jobs.html"),
		filepath.Join(baseDir, "web/templates/pages/job_detail.html"),
		filepath.Join(baseDir, "web/templates/pages/run_log.html"),
	}
	return template.Must(template.New("admin").Funcs(templateFuncs()).ParseFiles(paths...))
}

func (h *PageHandler) createJob(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	job := jobdomain.Job{
		Name:        strings.TrimSpace(r.FormValue("name")),
		Description: strings.TrimSpace(r.FormValue("description")),
		Enabled:     r.FormValue("enabled") == "on",
		Schedule: jobdomain.Schedule{
			Cron:     strings.TrimSpace(r.FormValue("cron")),
			Interval: time.Duration(parseInt(r.FormValue("interval_seconds"))) * time.Second,
			TimeZone: strings.TrimSpace(r.FormValue("time_zone")),
		},
		Executor:          buildExecutorSpec(r),
		ConcurrencyPolicy: jobdomain.ConcurrencyPolicy(defaultString(r.FormValue("concurrency_policy"), string(jobdomain.ConcurrencyForbid))),
	}
	created, err := h.service.Create(r.Context(), job, r.Form["dependency_id"])
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	lang := currentLang(r)
	http.Redirect(w, r, "/jobs/"+created.ID+"?lang="+lang, http.StatusSeeOther)
}

func (h *PageHandler) jobDetail(w http.ResponseWriter, r *http.Request, id string) {
	detail, err := h.service.GetDetail(r.Context(), id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	data := map[string]any{
		"Title":  detail.Job.Name,
		"Detail": viewmodel.NewJobDetail(detail.Job, detail.DependencyJobs, detail.Runs),
	}
	h.render(w, r, "job_detail", data)
}

func (h *PageHandler) triggerJob(w http.ResponseWriter, r *http.Request, id string) {
	if _, err := h.service.Trigger(r.Context(), id); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	lang := currentLang(r)
	http.Redirect(w, r, "/jobs/"+id+"?lang="+lang, http.StatusSeeOther)
}

func (h *PageHandler) pauseJob(w http.ResponseWriter, r *http.Request, id string) {
	if _, err := h.service.Pause(r.Context(), id); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	lang := currentLang(r)
	http.Redirect(w, r, "/jobs/"+id+"?lang="+lang, http.StatusSeeOther)
}

func (h *PageHandler) resumeJob(w http.ResponseWriter, r *http.Request, id string) {
	if _, err := h.service.Resume(r.Context(), id); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	lang := currentLang(r)
	http.Redirect(w, r, "/jobs/"+id+"?lang="+lang, http.StatusSeeOther)
}

func (h *PageHandler) RunLog(w http.ResponseWriter, r *http.Request) {
	path := strings.Trim(strings.TrimPrefix(r.URL.Path, "/job-runs/"), "/")
	parts := strings.Split(path, "/")
	if len(parts) != 2 || parts[1] != "logs" || r.Method != http.MethodGet {
		http.NotFound(w, r)
		return
	}
	content, err := h.logs.Read(r.Context(), parts[0])
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	data := map[string]any{
		"Title":   tr(dictionary(currentLang(r)), "run_log", "Run Log"),
		"RunID":   parts[0],
		"Content": content,
	}
	h.render(w, r, "run_log", data)
}

func buildExecutorSpec(r *http.Request) jobdomain.ExecutorSpec {
	if r.FormValue("executor_type") == string(jobdomain.ExecutorKindBinary) {
		return jobdomain.ExecutorSpec{
			Kind: jobdomain.ExecutorKindBinary,
			Binary: &jobdomain.BinaryTarget{
				Command: strings.TrimSpace(r.FormValue("binary_command")),
				Args:    parseArgs(r.FormValue("binary_args")),
				Timeout: time.Duration(parseInt(r.FormValue("binary_timeout_seconds"))) * time.Second,
			},
		}
	}
	return jobdomain.ExecutorSpec{
		Kind: jobdomain.ExecutorKindSDK,
		SDK: &jobdomain.SDKTarget{
			Protocol: defaultString(strings.TrimSpace(r.FormValue("sdk_protocol")), "http"),
			URL:      strings.TrimSpace(r.FormValue("sdk_url")),
			Method:   defaultString(strings.TrimSpace(r.FormValue("sdk_method")), http.MethodPost),
			Timeout:  time.Duration(parseInt(r.FormValue("sdk_timeout_seconds"))) * time.Second,
		},
	}
}

func parseArgs(raw string) []string {
	fields := strings.FieldsFunc(raw, func(r rune) bool {
		return r == ',' || r == '\n' || r == '\r'
	})
	result := make([]string, 0, len(fields))
	for _, field := range fields {
		field = strings.TrimSpace(field)
		if field != "" {
			result = append(result, field)
		}
	}
	return result
}

func parseInt(raw string) int {
	value, _ := strconv.Atoi(strings.TrimSpace(raw))
	return value
}

func defaultString(value, fallback string) string {
	if value == "" {
		return fallback
	}
	return value
}

func pct(part, total int) int {
	if total <= 0 || part <= 0 {
		return 0
	}
	value := part * 100 / total
	if value < 0 {
		return 0
	}
	if value > 100 {
		return 100
	}
	return value
}

func filterJobs(jobs []jobdomain.Job, query, statusFilter, executorFilter string) []jobdomain.Job {
	query = strings.ToLower(strings.TrimSpace(query))
	statusFilter = strings.ToLower(strings.TrimSpace(statusFilter))
	executorFilter = strings.ToLower(strings.TrimSpace(executorFilter))

	filtered := make([]jobdomain.Job, 0, len(jobs))
	for _, item := range jobs {
		if query != "" {
			haystack := strings.ToLower(item.Name + " " + item.Description + " " + item.ID)
			if !strings.Contains(haystack, query) {
				continue
			}
		}
		if statusFilter == "enabled" && !item.Enabled {
			continue
		}
		if statusFilter == "disabled" && item.Enabled {
			continue
		}
		if executorFilter != "" && executorFilter != "all" {
			kind := strings.ToLower(string(item.Executor.Kind))
			if executorFilter == "sdk" && kind != "sdk" {
				continue
			}
			if executorFilter == "binary" && kind != "binary" {
				continue
			}
		}
		filtered = append(filtered, item)
	}
	return filtered
}

func summarizeJobs(jobs []jobdomain.Job) (enabledCount, disabledCount, sdkCount, binaryCount int) {
	for _, item := range jobs {
		if item.Enabled {
			enabledCount++
		} else {
			disabledCount++
		}
		switch item.Executor.Kind {
		case jobdomain.ExecutorKindBinary:
			binaryCount++
		default:
			sdkCount++
		}
	}
	return enabledCount, disabledCount, sdkCount, binaryCount
}
