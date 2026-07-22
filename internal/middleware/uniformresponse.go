package middleware

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/zeromicro/go-zero/core/logx"
)

// bodyWriter 用于拦截并缓存底层的 Response 输出流字节数据
type bodyWriter struct {
	http.ResponseWriter
	body *bytes.Buffer
}

// Write 拦截重写字节流写入方法，将字节数据缓存至内存 Buffer 中，暂时不真正发送给客户端
func (w *bodyWriter) Write(b []byte) (int, error) {
	return w.body.Write(b)
}

// UniformResponseMiddleware 全局统一成功响应格式包装中间件
func UniformResponseMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// 1. 读取并记录入参日志 (r.Body)
		var reqBody []byte
		if r.Body != nil {
			reqBody, _ = io.ReadAll(r.Body)
			r.Body = io.NopCloser(bytes.NewBuffer(reqBody))
		}

		// 格式化入参，去除换行和多余空格，保证日志单行整洁输出
		reqStr := string(reqBody)
		reqStr = strings.ReplaceAll(reqStr, "\n", "")
		reqStr = strings.ReplaceAll(reqStr, "\r", "")

		logx.WithContext(r.Context()).Infof("HTTP 请求进入 [%s] %s, 入参: %s", r.Method, r.URL.Path, reqStr)

		bw := &bodyWriter{
			ResponseWriter: w,
			body:           bytes.NewBuffer(nil),
		}

		// 执行后续的 Handler 逻辑
		next(bw, r)

		// 获取内部 Handler 写入的原始字节流
		originalBody := bw.body.Bytes()
		if len(originalBody) == 0 {
			return
		}

		var finalResponse []byte
		isError := false

		// 判断是否是已经格式化好的错误输出结构（参数校验错误或自定义业务错误）
		var temp map[string]interface{}
		if err := json.Unmarshal(originalBody, &temp); err == nil {
			// 如果顶级 JSON 结构中包含 "code" 键，且不包含 "data" 键，表明这必然是错误回包，直接原样透传
			if _, hasCode := temp["code"]; hasCode {
				if _, hasData := temp["data"]; !hasData {
					finalResponse = originalBody
					isError = true
				}
			}
		}

		if !isError {
			// 如果是没有包含 code 字段的正常成功数据，则动态将其包裹为统一的规范格式
			var rawData json.RawMessage = originalBody
			wrapped := struct {
				Code int64           `json:"code"`
				Msg  string          `json:"msg"`
				Data json.RawMessage `json:"data"`
			}{
				Code: 200,
				Msg:  "",
				Data: rawData,
			}

			var err error
			finalResponse, err = json.Marshal(wrapped)
			if err != nil {
				finalResponse = originalBody
			}
		}

		// 2. 记录出参日志并对过长内容进行安全截断，防止大容量字典回包把 access.log 撑爆
		resStr := string(finalResponse)
		resStr = strings.ReplaceAll(resStr, "\n", "")
		resStr = strings.ReplaceAll(resStr, "\r", "")
		if len(resStr) > 1024 {
			resStr = resStr[:1024] + "... [truncated, total: " + strconv.Itoa(len(finalResponse)) + " bytes]"
		}
		logx.WithContext(r.Context()).Infof("HTTP 请求结束 [%s] %s, 出参: %s", r.Method, r.URL.Path, resStr)

		// 设置 JSON Content-Type 并向真实客户端写入包装后的数据
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.Write(finalResponse)
	}
}
