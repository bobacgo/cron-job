package handler

import (
	"html/template"
	"net/http"
	"net/url"
	"strings"
)

const langCookieName = "cronjob_lang"

var translations = map[string]map[string]string{
	"en": {
		"app_title":              "Cron Job Console",
		"app_subtitle":           "Cloud Task Control Plane",
		"nav_dashboard":          "Dashboard",
		"nav_jobs":               "Jobs",
		"nav_graph":              "Dependency Graph",
		"nav_audit":              "Ops Audit",
		"nav_health":             "Health",
		"nav_logout":             "Logout",
		"lang_zh":                "中文",
		"lang_en":                "EN",
		"login_kicker":           "Access Gateway",
		"login_title":            "Sign In",
		"login_desc":             "Authenticate to access the task control plane.",
		"login_username":         "Username",
		"login_password":         "Password",
		"login_submit":           "Sign In",
		"login_hint":             "Default credentials can be set by ADMIN_USER and ADMIN_PASSWORD.",
		"login_error_invalid":    "Invalid username or password.",
		"dashboard_kicker":       "Mission Control",
		"dashboard_title":        "Operate job orchestration like a cloud control plane, not a generic admin panel.",
		"dashboard_desc":         "The dashboard is now organized like an ops console: live posture, run-state pressure, execution health, and visual signal density that helps you see problems before opening individual jobs.",
		"snapshot":               "Snapshot",
		"generated_at":           "Generated at",
		"mode":                   "Mode",
		"execution":              "Execution",
		"mode_value":             "MVP / Single Node",
		"execution_value":        "SDK + Binary",
		"brand_direction":        "Brand Direction",
		"brand_direction_value":  "Cloud Platform Console",
		"brand_direction_desc":   "Dark control-plane surfaces, electric signal colors, and dense telemetry framing.",
		"total_jobs":             "Total Jobs",
		"enabled_jobs":           "Enabled Jobs",
		"recorded_runs":          "Recorded Runs",
		"active_runs":            "Active Runs",
		"queued_runs":            "Queued Pressure",
		"healthy_runs":           "Healthy Runs",
		"attention_runs":         "Attention Needed",
		"total_jobs_desc":        "Configured task definitions currently loaded by the service.",
		"enabled_jobs_desc":      "Schedules that can still enqueue new runs.",
		"recorded_runs_desc":     "Execution history persisted by the current process lifetime.",
		"signal_band":            "Signal Band",
		"runtime_health":         "Runtime Health",
		"dispatch_load":          "Dispatch Load",
		"dependency_pressure":    "Dependency Pressure",
		"execution_mix":          "Execution Mix",
		"trend_24h":              "Trend Blocks",
		"trend_note":             "Derived from current in-memory runtime state. Useful for posture, not for forensic analytics.",
		"jobs_kicker":            "Task Center",
		"jobs_title":             "Jobs",
		"jobs_desc":              "This page now behaves more like a task operations center: search, filter, inspect status mix, and review a denser execution catalog before opening any single job.",
		"mvp_console":            "Control Plane Beta",
		"inventory":              "Inventory",
		"defined_jobs":           "Defined Jobs",
		"entries":                "entries",
		"visible_jobs":           "Visible",
		"status_summary":         "Status Summary",
		"filter_bar":             "Filter Bar",
		"filter_search":          "Search",
		"filter_search_hint":     "Name, description, or job ID",
		"filter_status":          "Status",
		"filter_executor":        "Executor",
		"filter_apply":           "Apply Filters",
		"filter_reset":           "Reset",
		"status_all":             "All Statuses",
		"executor_all":           "All Executors",
		"automation_mix":         "Automation Mix",
		"table_id":               "ID / Signal",
		"table_name":             "Name",
		"table_executor":         "Executor",
		"table_schedule":         "Schedule",
		"table_next_run":         "Next Run",
		"table_status":           "Status",
		"table_action":           "Action",
		"status_enabled":         "enabled",
		"status_disabled":        "disabled",
		"no_jobs":                "No jobs yet. Create one with the compose panel.",
		"button_open_job":        "Open Job",
		"compose":                "Compose",
		"create_job":             "Create Job",
		"create_job_desc":        "Create a new task definition from the same surface. The right rail stays dense but structured, so operators can add jobs without losing inventory context.",
		"field_name":             "Name",
		"field_description":      "Description",
		"field_interval_seconds": "Interval Seconds",
		"field_cron":             "Cron",
		"field_time_zone":        "Time Zone",
		"field_executor_type":    "Executor Type",
		"sdk_settings":           "SDK Settings",
		"binary_settings":        "Binary Settings",
		"dependencies":           "Dependencies",
		"no_dependency_options":  "No existing jobs available as dependencies.",
		"field_enabled":          "Enabled",
		"button_create_job":      "Create Job",
		"job_detail":             "Job Detail",
		"job_control_center":     "Job Control Center",
		"job_control_desc":       "A focused surface for one task definition: execution posture, schedule metadata, dependency topology, and recent run signals in one place.",
		"button_manual_trigger":  "Manual Trigger",
		"button_pause":           "Pause",
		"button_resume":          "Resume",
		"status_paused":          "paused",
		"overview":               "Overview",
		"execution_profile":      "Execution Profile",
		"last_success":           "Last Success",
		"current_posture":        "Current Posture",
		"total_recent_runs":      "Recent Run Count",
		"dependency_count":       "Dependencies",
		"schedule_lane":          "Schedule Lane",
		"history":                "History",
		"recent_runs":            "Recent Runs",
		"table_trigger":          "Trigger",
		"table_scheduled":        "Scheduled",
		"table_started":          "Started",
		"table_finished":         "Finished",
		"table_message":          "Message",
		"table_logs":             "Logs",
		"button_view":            "View",
		"no_runs":                "No runs yet.",
		"status_pending":         "Pending",
		"status_blocked":         "Blocked",
		"status_ready":           "Ready",
		"status_running":         "Running",
		"status_succeeded":       "Succeeded",
		"status_failed":          "Failed",
		"status_timed_out":       "Timed Out",
		"status_canceled":        "Canceled",
		"status_skipped":         "Skipped",
		"trigger_manual":         "Manual",
		"trigger_schedule":       "Schedule",
		"upstream_jobs":          "Upstream Jobs",
		"no_upstream":            "No upstream dependencies.",
		"execution_output":       "Execution Output",
		"run_log":                "Run Log",
		"run_forensics":          "Execution Forensics",
		"run_forensics_desc":     "Inspect the raw captured output for a single run in a terminal-style panel with the execution identifier visible at all times.",
		"graph_title":            "Dependency Graph",
		"graph_desc":             "Visualize upstream/downstream links for all jobs and inspect source markup for review.",
		"graph_nodes":            "Nodes",
		"graph_edges":            "Edges",
		"graph_canvas":           "Interactive Canvas",
		"graph_interactive_hint": "Drag nodes to rearrange topology. Click a node to highlight upstream and downstream paths.",
		"graph_zoom_pan_hint":    "Use mouse wheel to zoom and drag empty canvas to pan.",
		"graph_reset":            "Reset Layout",
		"graph_source":           "Graph Source",
		"graph_detail_title":     "Node Detail",
		"graph_detail_status":    "Latest Status",
		"graph_detail_failed":    "Failed Runs",
		"graph_detail_next_run":  "Next Run",
		"audit_title":            "Operations Audit",
		"audit_desc":             "Inspect recent run operations and searchable log excerpts from one page.",
		"audit_filters":          "Audit Filters",
		"audit_run_events":       "Run Events",
		"audit_log_hits":         "Log Hits",
		"run_id":                 "Run ID",
		"captured_output":        "Captured Output",
		"terminal_stream":        "Terminal Stream",
		"raw_output":             "Raw Output",
		"no_log_content":         "No log content yet.",
	},
	"zh": {
		"app_title":              "定时任务控制台",
		"app_subtitle":           "云任务控制平面",
		"nav_dashboard":          "仪表盘",
		"nav_jobs":               "任务",
		"nav_graph":              "依赖图",
		"nav_audit":              "运维审计",
		"nav_health":             "健康检查",
		"nav_logout":             "退出登录",
		"lang_zh":                "中文",
		"lang_en":                "EN",
		"login_kicker":           "访问网关",
		"login_title":            "登录",
		"login_desc":             "登录后可访问任务控制平面。",
		"login_username":         "用户名",
		"login_password":         "密码",
		"login_submit":           "登录",
		"login_hint":             "可通过 ADMIN_USER 和 ADMIN_PASSWORD 设置默认账号。",
		"login_error_invalid":    "用户名或密码错误。",
		"dashboard_kicker":       "任务指挥台",
		"dashboard_title":        "把任务编排后台做成云平台控制台，而不是普通管理页。",
		"dashboard_desc":         "新的仪表盘更偏监控台：突出运行态势、执行压力、健康信号和信息密度，让你在点进单个任务前就知道系统现在是否稳定。",
		"snapshot":               "快照",
		"generated_at":           "生成时间",
		"mode":                   "模式",
		"execution":              "执行模型",
		"mode_value":             "MVP / 单节点",
		"execution_value":        "SDK + 二进制",
		"brand_direction":        "品牌方向",
		"brand_direction_value":  "云平台控制台",
		"brand_direction_desc":   "深色控制面、冷色信号光和更高的信息编排密度。",
		"total_jobs":             "任务总数",
		"enabled_jobs":           "启用任务",
		"recorded_runs":          "运行记录",
		"active_runs":            "活跃执行",
		"queued_runs":            "排队压力",
		"healthy_runs":           "健康执行",
		"attention_runs":         "需关注执行",
		"total_jobs_desc":        "当前服务加载的任务定义数量。",
		"enabled_jobs_desc":      "仍然允许产生新执行实例的任务。",
		"recorded_runs_desc":     "当前进程生命周期内记录的执行历史。",
		"signal_band":            "状态信号带",
		"runtime_health":         "运行健康度",
		"dispatch_load":          "分发负载",
		"dependency_pressure":    "依赖压力",
		"execution_mix":          "执行构成",
		"trend_24h":              "趋势模块",
		"trend_note":             "基于当前内存中的运行态计算，适合看态势，不适合做审计分析。",
		"jobs_kicker":            "任务中心",
		"jobs_title":             "任务",
		"jobs_desc":              "这里现在更像专业任务中心：可以搜索、筛选、看状态摘要，再进入单任务详情，而不是单纯的列表加表单。",
		"mvp_console":            "控制平面 Beta",
		"inventory":              "任务目录",
		"defined_jobs":           "已定义任务",
		"entries":                "条",
		"visible_jobs":           "当前可见",
		"status_summary":         "状态摘要",
		"filter_bar":             "筛选条",
		"filter_search":          "搜索",
		"filter_search_hint":     "按名称、描述或任务 ID 搜索",
		"filter_status":          "状态",
		"filter_executor":        "执行器",
		"filter_apply":           "应用筛选",
		"filter_reset":           "重置",
		"status_all":             "全部状态",
		"executor_all":           "全部执行器",
		"automation_mix":         "执行构成",
		"table_id":               "ID / 信号",
		"table_name":             "名称",
		"table_executor":         "执行器",
		"table_schedule":         "调度",
		"table_next_run":         "下次运行",
		"table_status":           "状态",
		"table_action":           "操作",
		"status_enabled":         "启用",
		"status_disabled":        "停用",
		"no_jobs":                "还没有任务，可以直接在右侧创建。",
		"button_open_job":        "进入任务",
		"compose":                "创建面板",
		"create_job":             "新建任务",
		"create_job_desc":        "在同一个控制面里新增任务定义。右侧保持高密度但有清晰分区，避免创建任务时丢失目录上下文。",
		"field_name":             "名称",
		"field_description":      "描述",
		"field_interval_seconds": "间隔秒数",
		"field_cron":             "Cron 表达式",
		"field_time_zone":        "时区",
		"field_executor_type":    "执行类型",
		"sdk_settings":           "SDK 配置",
		"binary_settings":        "二进制配置",
		"dependencies":           "依赖任务",
		"no_dependency_options":  "当前没有可选的依赖任务。",
		"field_enabled":          "启用",
		"button_create_job":      "创建任务",
		"job_detail":             "任务详情",
		"job_control_center":     "单任务指挥台",
		"job_control_desc":       "把单个任务的执行态势、调度元信息、依赖拓扑和最近运行信号集中在一个界面里。",
		"button_manual_trigger":  "手动触发",
		"button_pause":           "暂停",
		"button_resume":          "恢复",
		"status_paused":          "已暂停",
		"overview":               "概览",
		"execution_profile":      "执行画像",
		"last_success":           "最近成功时间",
		"current_posture":        "当前态势",
		"total_recent_runs":      "最近运行数",
		"dependency_count":       "依赖数量",
		"schedule_lane":          "调度通道",
		"history":                "历史记录",
		"recent_runs":            "最近运行",
		"table_trigger":          "触发方式",
		"table_scheduled":        "计划时间",
		"table_started":          "开始时间",
		"table_finished":         "完成时间",
		"table_message":          "消息",
		"table_logs":             "日志",
		"button_view":            "查看",
		"no_runs":                "暂无运行记录。",
		"status_pending":         "待处理",
		"status_blocked":         "阻塞中",
		"status_ready":           "就绪",
		"status_running":         "运行中",
		"status_succeeded":       "成功",
		"status_failed":          "失败",
		"status_timed_out":       "超时",
		"status_canceled":        "已取消",
		"status_skipped":         "已跳过",
		"trigger_manual":         "手动",
		"trigger_schedule":       "调度",
		"upstream_jobs":          "上游任务",
		"no_upstream":            "没有上游依赖。",
		"execution_output":       "执行输出",
		"run_log":                "运行日志",
		"run_forensics":          "执行取证面板",
		"run_forensics_desc":     "以终端风格查看单次运行的原始捕获输出，并持续显示运行标识，便于排查。",
		"graph_title":            "依赖关系图",
		"graph_desc":             "可视化所有任务的上下游依赖关系，并提供图源码用于审阅。",
		"graph_nodes":            "节点",
		"graph_edges":            "边",
		"graph_canvas":           "交互画布",
		"graph_interactive_hint": "可拖拽节点重排拓扑，点击节点可高亮上下游路径。",
		"graph_zoom_pan_hint":    "滚轮缩放，按住画布空白区域可平移。",
		"graph_reset":            "重置布局",
		"graph_source":           "图源码",
		"graph_detail_title":     "节点详情",
		"graph_detail_status":    "最近状态",
		"graph_detail_failed":    "失败次数",
		"graph_detail_next_run":  "下次运行",
		"audit_title":            "运维审计",
		"audit_desc":             "在一个页面查看最近运行事件与可检索日志片段。",
		"audit_filters":          "审计筛选",
		"audit_run_events":       "运行事件",
		"audit_log_hits":         "日志命中",
		"run_id":                 "运行 ID",
		"captured_output":        "捕获输出",
		"terminal_stream":        "终端流",
		"raw_output":             "原始输出",
		"no_log_content":         "暂时没有日志内容。",
	},
}

func templateFuncs() template.FuncMap {
	return template.FuncMap{
		"tr": func(dict map[string]string, key string) string {
			if dict == nil {
				return key
			}
			if value, ok := dict[key]; ok {
				return value
			}
			if value, ok := translations["en"][key]; ok {
				return value
			}
			return key
		},
		"withLang": func(path, lang string) string {
			if lang == "" {
				return path
			}
			separator := "?"
			if strings.Contains(path, "?") {
				separator = "&"
			}
			return path + separator + "lang=" + url.QueryEscape(lang)
		},
		"switchLang": func(path, rawQuery, lang string) string {
			values, err := url.ParseQuery(rawQuery)
			if err != nil {
				values = url.Values{}
			}
			values.Set("lang", lang)
			encoded := values.Encode()
			if encoded == "" {
				return path
			}
			return path + "?" + encoded
		},
		"statusLabel": func(dict map[string]string, status string) string {
			switch status {
			case "Pending":
				return tr(dict, "status_pending", status)
			case "Blocked":
				return tr(dict, "status_blocked", status)
			case "Ready":
				return tr(dict, "status_ready", status)
			case "Running":
				return tr(dict, "status_running", status)
			case "Succeeded":
				return tr(dict, "status_succeeded", status)
			case "Failed":
				return tr(dict, "status_failed", status)
			case "TimedOut":
				return tr(dict, "status_timed_out", status)
			case "Canceled":
				return tr(dict, "status_canceled", status)
			case "Skipped":
				return tr(dict, "status_skipped", status)
			default:
				return status
			}
		},
		"triggerLabel": func(dict map[string]string, triggerType string) string {
			switch triggerType {
			case "manual":
				return tr(dict, "trigger_manual", triggerType)
			case "schedule":
				return tr(dict, "trigger_schedule", triggerType)
			default:
				return triggerType
			}
		},
	}
}

func resolveLang(w http.ResponseWriter, r *http.Request) string {
	lang := currentLang(r)
	if queryLang := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("lang"))); queryLang == "zh" || queryLang == "en" {
		http.SetCookie(w, &http.Cookie{
			Name:     langCookieName,
			Value:    lang,
			Path:     "/",
			MaxAge:   86400 * 365,
			HttpOnly: false,
			SameSite: http.SameSiteLaxMode,
		})
		return lang
	}
	return lang
}

func currentLang(r *http.Request) string {
	if queryLang := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("lang"))); queryLang == "zh" || queryLang == "en" {
		return queryLang
	}

	if cookie, err := r.Cookie(langCookieName); err == nil {
		if cookie.Value == "zh" || cookie.Value == "en" {
			return cookie.Value
		}
	}

	accept := strings.ToLower(r.Header.Get("Accept-Language"))
	if strings.HasPrefix(accept, "zh") || strings.Contains(accept, ",zh") {
		return "zh"
	}
	return "en"
}

func dictionary(lang string) map[string]string {
	if dict, ok := translations[lang]; ok {
		return dict
	}
	return translations["en"]
}

func tr(dict map[string]string, key, fallback string) string {
	if dict == nil {
		return fallback
	}
	if value, ok := dict[key]; ok {
		return value
	}
	if value, ok := translations["en"][key]; ok {
		return value
	}
	return fallback
}
