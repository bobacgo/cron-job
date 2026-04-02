package handler

import (
	"fmt"
	"html/template"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	jobapp "github.com/bobacgo/cron-job/internal/app/job"
	jobdomain "github.com/bobacgo/cron-job/internal/domain/job"
	jobrundomain "github.com/bobacgo/cron-job/internal/domain/jobrun"
	"github.com/bobacgo/cron-job/internal/repository"
	"github.com/bobacgo/cron-job/internal/transport/admin/viewmodel"
)

type PageHandler struct {
	service   *jobapp.Service
	logs      repository.LogRepository
	runs      repository.JobRunRepository
	auth      *authService
	templates *template.Template
}

func NewPageHandler(service *jobapp.Service, runs repository.JobRunRepository, logs repository.LogRepository) *PageHandler {
	return &PageHandler{
		service:   service,
		logs:      logs,
		runs:      runs,
		auth:      newAuthService(),
		templates: mustParseTemplates(),
	}
}

func (h *PageHandler) RequireAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if _, ok := h.currentUser(r); ok {
			next(w, r)
			return
		}
		nextURL := r.URL.Path
		if r.URL.RawQuery != "" {
			nextURL += "?" + r.URL.RawQuery
		}
		lang := currentLang(r)
		loginURL := "/login?next=" + url.QueryEscape(nextURL) + "&lang=" + url.QueryEscape(lang)
		http.Redirect(w, r, loginURL, http.StatusFound)
	}
}

func (h *PageHandler) Login(w http.ResponseWriter, r *http.Request) {
	lang := currentLang(r)
	dict := dictionary(lang)
	if _, ok := h.currentUser(r); ok && r.Method == http.MethodGet {
		http.Redirect(w, r, appendLang("/", lang), http.StatusSeeOther)
		return
	}

	next := safeNext(r.URL.Query().Get("next"))
	if r.Method == http.MethodPost {
		if err := r.ParseForm(); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		next = safeNext(r.FormValue("next"))
		username := strings.TrimSpace(r.FormValue("username"))
		password := r.FormValue("password")
		token, ok := h.auth.authenticate(username, password)
		if !ok {
			data := map[string]any{
				"Title":        tr(dict, "login_title", "Login"),
				"HideNav":      true,
				"Error":        tr(dict, "login_error_invalid", "Invalid credentials"),
				"Next":         next,
				"DefaultUser":  os.Getenv("ADMIN_USER"),
				"DefaultPass":  os.Getenv("ADMIN_PASSWORD"),
				"CurrentPath":  r.URL.Path,
				"CurrentQuery": r.URL.RawQuery,
			}
			h.render(w, r, "login", data)
			return
		}
		http.SetCookie(w, &http.Cookie{
			Name:     authCookieName,
			Value:    token,
			Path:     "/",
			MaxAge:   int(h.auth.ttl.Seconds()),
			HttpOnly: true,
			SameSite: http.SameSiteLaxMode,
		})
		http.Redirect(w, r, appendLang(next, lang), http.StatusSeeOther)
		return
	}

	data := map[string]any{
		"Title":        tr(dict, "login_title", "Login"),
		"HideNav":      true,
		"Next":         next,
		"DefaultUser":  os.Getenv("ADMIN_USER"),
		"DefaultPass":  os.Getenv("ADMIN_PASSWORD"),
		"CurrentPath":  r.URL.Path,
		"CurrentQuery": r.URL.RawQuery,
	}
	h.render(w, r, "login", data)
}

func (h *PageHandler) Logout(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if cookie, err := r.Cookie(authCookieName); err == nil {
		h.auth.removeSession(cookie.Value)
	}
	http.SetCookie(w, &http.Cookie{
		Name:     authCookieName,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})
	http.Redirect(w, r, appendLang("/login", currentLang(r)), http.StatusSeeOther)
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

func (h *PageHandler) DependencyGraph(w http.ResponseWriter, r *http.Request) {
	jobs, err := h.service.List(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	nodes := make([]map[string]any, 0, len(jobs))
	edges := make([]map[string]string, 0)
	for _, item := range jobs {
		detail, err := h.service.GetDetail(r.Context(), item.ID)
		latestStatus := "Pending"
		failedCount := 0
		if err == nil {
			if len(detail.Runs) > 0 {
				latestStatus = string(detail.Runs[0].Status)
			}
			for _, run := range detail.Runs {
				switch run.Status {
				case jobrundomain.StatusFailed, jobrundomain.StatusTimedOut, jobrundomain.StatusCanceled:
					failedCount++
				}
			}
			for _, edge := range detail.Dependencies {
				edges = append(edges, map[string]string{"job_id": edge.JobID, "depends_on": edge.DependsOnJobID})
			}
		}
		nextRunAt := "-"
		if item.NextRunAt > 0 {
			nextRunAt = time.Unix(item.NextRunAt, 0).UTC().Format("2006-01-02 15:04:05Z07:00")
		}
		nodes = append(nodes, map[string]any{
			"id":           item.ID,
			"name":         item.Name,
			"latestStatus": latestStatus,
			"failedCount":  failedCount,
			"nextRunAt":    nextRunAt,
		})
	}
	sort.Slice(nodes, func(i, j int) bool { return fmt.Sprintf("%v", nodes[i]["name"]) < fmt.Sprintf("%v", nodes[j]["name"]) })
	markup := buildGraphMarkup(nodes, edges)
	data := map[string]any{
		"Title":       tr(dictionary(currentLang(r)), "graph_title", "Dependency Graph"),
		"GraphNodes":  nodes,
		"GraphEdges":  edges,
		"GraphMarkup": markup,
	}
	h.render(w, r, "graph", data)
}

func (h *PageHandler) OpsAudit(w http.ResponseWriter, r *http.Request) {
	runs, err := h.runs.List(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	statusFilter := strings.TrimSpace(r.URL.Query().Get("status"))
	keyword := strings.TrimSpace(r.URL.Query().Get("q"))
	items := make([]jobrundomain.JobRun, 0, len(runs))
	for _, run := range runs {
		if statusFilter != "" && string(run.Status) != statusFilter {
			continue
		}
		if keyword != "" {
			haystack := strings.ToLower(run.JobID + " " + run.ID + " " + run.Message)
			if !strings.Contains(haystack, strings.ToLower(keyword)) {
				continue
			}
		}
		items = append(items, run)
		if len(items) >= 80 {
			break
		}
	}
	logItems, err := h.logs.Search(r.Context(), repository.LogQuery{Contains: keyword, Limit: 50})
	if err != nil {
		logItems = nil
	}
	data := map[string]any{
		"Title":         tr(dictionary(currentLang(r)), "audit_title", "Operations Audit"),
		"AuditRuns":     items,
		"AuditLogItems": logItems,
		"AuditStatus":   statusFilter,
		"AuditQuery":    keyword,
	}
	h.render(w, r, "audit", data)
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
	enabledCount, disabledCount, sdkCount, binaryCount, shellCount := summarizeJobs(jobs)
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
		"ShellCount":        shellCount,
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
	if user, ok := h.currentUser(r); ok {
		data["AuthUser"] = user
	}
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
		filepath.Join(baseDir, "web/templates/pages/login.html"),
		filepath.Join(baseDir, "web/templates/pages/dashboard.html"),
		filepath.Join(baseDir, "web/templates/pages/graph.html"),
		filepath.Join(baseDir, "web/templates/pages/audit.html"),
		filepath.Join(baseDir, "web/templates/pages/jobs.html"),
		filepath.Join(baseDir, "web/templates/pages/job_detail.html"),
		filepath.Join(baseDir, "web/templates/pages/run_log.html"),
	}
	return template.Must(template.New("admin").Funcs(templateFuncs()).ParseFiles(paths...))
}

func buildGraphMarkup(nodes []map[string]any, edges []map[string]string) string {
	var b strings.Builder
	b.WriteString("graph LR\n")
	for _, node := range nodes {
		id := strings.ReplaceAll(fmt.Sprintf("%v", node["id"]), "-", "_")
		name := strings.ReplaceAll(fmt.Sprintf("%v", node["name"]), "\"", "'")
		b.WriteString(fmt.Sprintf("    %s[\"%s\"]\n", id, name))
	}
	for _, edge := range edges {
		from := strings.ReplaceAll(edge["depends_on"], "-", "_")
		to := strings.ReplaceAll(edge["job_id"], "-", "_")
		b.WriteString(fmt.Sprintf("    %s --> %s\n", from, to))
	}
	return b.String()
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
	switch r.FormValue("executor_type") {
	case string(jobdomain.ExecutorKindBinary):
		return jobdomain.ExecutorSpec{
			Kind: jobdomain.ExecutorKindBinary,
			Binary: &jobdomain.BinaryTarget{
				Command: strings.TrimSpace(r.FormValue("binary_command")),
				Args:    parseArgs(r.FormValue("binary_args")),
				Timeout: time.Duration(parseInt(r.FormValue("binary_timeout_seconds"))) * time.Second,
			},
		}
	case string(jobdomain.ExecutorKindShell):
		return jobdomain.ExecutorSpec{
			Kind: jobdomain.ExecutorKindShell,
			Shell: &jobdomain.ShellTarget{
				Script:  r.FormValue("shell_script"),
				Shell:   defaultString(strings.TrimSpace(r.FormValue("shell_shell")), "/bin/sh"),
				Timeout: time.Duration(parseInt(r.FormValue("shell_timeout_seconds"))) * time.Second,
			},
		}
	default:
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

func safeNext(target string) string {
	target = strings.TrimSpace(target)
	if target == "" || !strings.HasPrefix(target, "/") || strings.HasPrefix(target, "//") {
		return "/"
	}
	return target
}

func appendLang(path, lang string) string {
	if lang == "" {
		return path
	}
	separator := "?"
	if strings.Contains(path, "?") {
		separator = "&"
	}
	return path + separator + "lang=" + url.QueryEscape(lang)
}

func (h *PageHandler) currentUser(r *http.Request) (string, bool) {
	cookie, err := r.Cookie(authCookieName)
	if err != nil {
		return "", false
	}
	return h.auth.userByToken(cookie.Value)
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
			if executorFilter != kind {
				continue
			}
		}
		filtered = append(filtered, item)
	}
	return filtered
}

func summarizeJobs(jobs []jobdomain.Job) (enabledCount, disabledCount, sdkCount, binaryCount, shellCount int) {
	for _, item := range jobs {
		if item.Enabled {
			enabledCount++
		} else {
			disabledCount++
		}
		switch item.Executor.Kind {
		case jobdomain.ExecutorKindSDK:
			sdkCount++
		case jobdomain.ExecutorKindBinary:
			binaryCount++
		case jobdomain.ExecutorKindShell:
			shellCount++
		default:
			sdkCount++
		}
	}
	return enabledCount, disabledCount, sdkCount, binaryCount, shellCount
}
