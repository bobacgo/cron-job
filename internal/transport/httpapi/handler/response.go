package handler

import (
	"net/http"

	hsresponse "github.com/bobacgo/cron-job/kit/hs/response"
)

// writeJSONStatus 统一输出 JSON，避免每个 handler 重复写 header 和编码逻辑。
func writeJSONStatus(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	hsresponse.JSON(w, v)
}

func writeError(w http.ResponseWriter, status int, err error) {
	if err == nil {
		return
	}
	writeJSONStatus(w, status, map[string]any{
		"message": err.Error(),
	})
}

func writeAPIResponse(w http.ResponseWriter, status int, code int, data any, msg string) {
	writeJSONStatus(w, status, hsresponse.Resp{
		Code: code,
		Msg:  msg,
		Data: data,
	})
}
