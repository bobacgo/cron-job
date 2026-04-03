package errs

import (
	"github.com/bobacgo/cron-job/kit/hs/response/codes"
	"github.com/bobacgo/cron-job/kit/hs/response/status"
)

var (
	BadRequest    = status.New(codes.BadRequest, "请求参数错误")
	InternalError = status.New(codes.InternalServerError, "服务器繁忙")
)
