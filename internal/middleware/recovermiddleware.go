package middleware

import (
	"fmt"
	"net/http"
	"runtime/debug"
	"strings"

	"github.com/zeromicro/go-zero/core/logx"
)

// RecoverMiddleware 是用于全局捕获运行时 Panic 的兜底中间件，防止系统崩溃并统一返回友好格式错误。
func RecoverMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				// 获取 panic 调用堆栈，并替换换行符以防日志收集系统（如 ELK、Loki）切分日志导致 Trace 丢失
				stack := strings.ReplaceAll(string(debug.Stack()), "\n", " | ")
				// 打印详细错误日志，便于开发运维人员排查真实故障
				logx.WithContext(r.Context()).Errorf("【系统故障拦截】发生全局 Panic 崩溃风险: %v. 堆栈信息: %s", err, stack)

				// 向客户端响应统一格式的 JSON 友好异常回显，HTTP 状态码返回 200 OK
				w.Header().Set("Content-Type", "application/json; charset=utf-8")
				w.WriteHeader(http.StatusOK)
				_, _ = fmt.Fprint(w, `{"code":500,"msg":"系统繁忙，请稍后再试"}`)
			}
		}()
		next(w, r)
	}
}
