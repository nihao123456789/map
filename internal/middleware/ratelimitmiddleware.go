package middleware

import (
	"encoding/json"
	"net/http"

	"github.com/zeromicro/go-zero/core/logx"
	"golang.org/x/time/rate"
)

// RateLimitMiddleware 提供了基于单机令牌桶限流算法的 HTTP 全局接口限流拦截功能。
func RateLimitMiddleware(limiter *rate.Limiter, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if limiter != nil && !limiter.Allow() {
			logx.WithContext(r.Context()).Errorf("接口请求触发全局限流拦截: [%s] %s", r.Method, r.URL.Path)

			w.Header().Set("Content-Type", "application/json; charset=utf-8")
			w.WriteHeader(http.StatusTooManyRequests)

			resp := struct {
				Code int64  `json:"code"`
				Msg  string `json:"msg"`
			}{
				Code: 429,
				Msg:  "请求过于频繁，请稍后再试",
			}
			
			eb, _ := json.Marshal(resp)
			_, _ = w.Write(eb)
			return
		}
		next(w, r)
	}
}
