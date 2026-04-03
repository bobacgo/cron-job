package router

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"

	httpapihandler "github.com/bobacgo/cron-job/internal/transport/httpapi/handler"
	"github.com/bobacgo/cron-job/kit/hs"
)

func New(jobHandler *httpapihandler.JobHandler, mgrHandler *httpapihandler.MgrHandler) http.Handler {
	runLogHandler := httpapihandler.NewRunLogHandler(jobHandler.Service())
	mux := http.NewServeMux()

	// 统一接住 API 预检请求，避免浏览器在跨域登录时因为 OPTIONS 404 被拦截。
	mux.Handle("/api/", hs.Cors(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		http.NotFound(w, r)
	})))

	// API 路由统一切到 hs 分组，新增接口时直接在这里挂即可。
	authGroup := hs.NewGroup("/api", mux, hs.Cors, hs.RequestID, hs.Logger)
	authGroup.HandleFunc("POST /user/login", mgrHandler.Login)
	authGroup.HandleFunc("GET /user-info", mgrHandler.UserInfo)
	authGroup.HandleFunc("POST /logout", mgrHandler.Logout)
	authGroup.HandleFunc("GET /user/list", mgrHandler.UserList)
	authGroup.HandleFunc("POST /user", mgrHandler.UserCreate)
	authGroup.HandleFunc("PUT /user", mgrHandler.UserUpdate)
	authGroup.HandleFunc("PUT /user/status", mgrHandler.UserUpdateStatus)
	authGroup.HandleFunc("PUT /user/role", mgrHandler.UserUpdateRole)
	authGroup.HandleFunc("PUT /user/password", mgrHandler.UserUpdatePassword)
	authGroup.HandleFunc("DELETE /user", mgrHandler.UserDelete)
	authGroup.HandleFunc("GET /menu/list", mgrHandler.MenuList)
	authGroup.HandleFunc("GET /menu/tree", mgrHandler.MenuTree)
	authGroup.HandleFunc("POST /menu", mgrHandler.MenuCreate)
	authGroup.HandleFunc("PUT /menu", mgrHandler.MenuUpdate)
	authGroup.HandleFunc("DELETE /menu", mgrHandler.MenuDelete)
	authGroup.HandleFunc("GET /role/list", mgrHandler.RoleList)
	authGroup.HandleFunc("GET /role/one", mgrHandler.RoleOne)
	authGroup.HandleFunc("POST /role", mgrHandler.RoleCreate)
	authGroup.HandleFunc("PUT /role", mgrHandler.RoleUpdate)
	authGroup.HandleFunc("DELETE /role", mgrHandler.RoleDelete)
	authGroup.HandleFunc("GET /role/permissions", mgrHandler.RolePermissionsGet)
	authGroup.HandleFunc("POST /role/permissions", mgrHandler.RolePermissionsPost)

	apiGroup := hs.NewGroup("/api/v1", mux, hs.Cors, hs.RequestID, hs.Logger)
	apiGroup.HandleFunc("GET /healthz", httpapihandler.Health)
	apiGroup.HandleFunc("GET /jobs", jobHandler.List)
	apiGroup.HandleFunc("POST /jobs", jobHandler.Create)
	apiGroup.HandleFunc("GET /jobs/{id}", func(w http.ResponseWriter, r *http.Request) {
		jobHandler.Get(w, r, r.PathValue("id"))
	})
	apiGroup.HandleFunc("POST /jobs/{id}/trigger", func(w http.ResponseWriter, r *http.Request) {
		jobHandler.Trigger(w, r, r.PathValue("id"))
	})
	apiGroup.HandleFunc("POST /jobs/{id}/pause", func(w http.ResponseWriter, r *http.Request) {
		jobHandler.Pause(w, r, r.PathValue("id"))
	})
	apiGroup.HandleFunc("POST /jobs/{id}/resume", func(w http.ResponseWriter, r *http.Request) {
		jobHandler.Resume(w, r, r.PathValue("id"))
	})
	apiGroup.HandleFunc("GET /job-runs/{id}/logs", func(w http.ResponseWriter, r *http.Request) {
		runLogHandler.Logs(w, r, r.PathValue("id"))
	})
	apiGroup.HandleFunc("POST /job-runs/{id}/cancel", func(w http.ResponseWriter, r *http.Request) {
		runLogHandler.Cancel(w, r, r.PathValue("id"))
	})
	apiGroup.HandleFunc("POST /job-runs/{id}/retry", func(w http.ResponseWriter, r *http.Request) {
		runLogHandler.Retry(w, r, r.PathValue("id"))
	})
	apiGroup.HandleFunc("GET /logs/search", runLogHandler.Search)

	// 页面入口改为 Vue SPA：生产环境优先读 web/dist，开发环境直接跳 Vite。
	mux.Handle("/", newWebHandler())
	return mux
}

func newWebHandler() http.Handler {
	baseDir, err := os.Getwd()
	if err != nil {
		return redirectToVite()
	}

	distDir := filepath.Join(baseDir, "web", "dist")
	if info, statErr := os.Stat(distDir); statErr == nil && info.IsDir() {
		fileServer := http.FileServer(http.Dir(distDir))
		indexFile := filepath.Join(distDir, "index.html")

		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if strings.HasPrefix(r.URL.Path, "/api/") {
				http.NotFound(w, r)
				return
			}
			if r.URL.Path == "/" {
				http.ServeFile(w, r, indexFile)
				return
			}

			assetPath := filepath.Join(distDir, strings.TrimPrefix(r.URL.Path, "/"))
			if assetInfo, assetErr := os.Stat(assetPath); assetErr == nil && !assetInfo.IsDir() {
				fileServer.ServeHTTP(w, r)
				return
			}

			// Vue Router 的 history 模式回退到 index.html。
			http.ServeFile(w, r, indexFile)
		})
	}

	return redirectToVite()
}

func redirectToVite() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/api/") {
			http.NotFound(w, r)
			return
		}
		target := "http://127.0.0.1:3002" + r.URL.RequestURI()
		http.Redirect(w, r, target, http.StatusTemporaryRedirect)
	})
}
