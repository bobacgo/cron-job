package router

import (
	"net/http"

	adminhandler "github.com/bobacgo/cron-job/internal/transport/admin/handler"
	httpapihandler "github.com/bobacgo/cron-job/internal/transport/httpapi/handler"
)

func New(jobHandler *httpapihandler.JobHandler, pages *adminhandler.PageHandler) http.Handler {
	runLogHandler := httpapihandler.NewRunLogHandler(jobHandler.Service())
	authRequired := pages.RequireAuth
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/healthz", httpapihandler.Health)
	mux.HandleFunc("/api/v1/jobs", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			jobHandler.List(w, r)
		case http.MethodPost:
			jobHandler.Create(w, r)
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	})
	mux.HandleFunc("/api/v1/jobs/", jobHandler.HandleByID)
	mux.HandleFunc("/api/v1/job-runs/", runLogHandler.Handle)
	mux.HandleFunc("/api/v1/logs/search", runLogHandler.Search)
	mux.HandleFunc("/login", pages.Login)
	mux.HandleFunc("/logout", pages.Logout)
	mux.HandleFunc("/", authRequired(pages.Dashboard))
	mux.HandleFunc("/jobs", authRequired(pages.Jobs))
	mux.HandleFunc("/jobs/", authRequired(pages.JobRoutes))
	mux.HandleFunc("/job-runs/", authRequired(pages.RunLog))
	return mux
}
