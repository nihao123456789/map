// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.1

package middleware

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"map-server/internal/consts"
)

// SignatureMiddleware 是签名验证中间件结构体
type SignatureMiddleware struct {
	secret string
}

// NewSignatureMiddleware 创建并返回一个新的签名验证中间件实例
func NewSignatureMiddleware(secret string) *SignatureMiddleware {
	return &SignatureMiddleware{
		secret: secret,
	}
}

// Handle 接管 HTTP 请求流，对 API 请求头中携带的时间戳、随机值和 SHA-256 签名进行安全验证
func (m *SignatureMiddleware) Handle(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// 获取 X-Timestamp 时间戳
		timestampStr := r.Header.Get("X-Timestamp")
		if timestampStr == "" {
			m.writeError(w, "X-Timestamp 不能为空")
			return
		}

		// 解析时间戳数值
		timestamp, err := strconv.ParseInt(timestampStr, 10, 64)
		if err != nil {
			m.writeError(w, "X-Timestamp 格式不正确")
			return
		}

		// 防重放攻击校验：请求发起时间与当前时间差不得超过 300 秒 (5分钟)
		now := time.Now().Unix()
		diff := now - timestamp
		if diff < -consts.SignatureTimeDiffLimit || diff > consts.SignatureTimeDiffLimit {
			m.writeError(w, "请求已过期（防重放）")
			return
		}

		// 获取 X-Nonce 随机字符
		nonce := r.Header.Get("X-Nonce")
		if nonce == "" {
			m.writeError(w, "X-Nonce 不能为空")
			return
		}

		// 获取 X-Signature 签名值
		signature := r.Header.Get("X-Signature")
		if signature == "" {
			m.writeError(w, "X-Signature 不能为空")
			return
		}

		// 按照客户端约定计算本地 SHA-256 签名
		rawStr := fmt.Sprintf("timestamp=%s&nonce=%s&secret=%s", timestampStr, nonce, m.secret)
		hash := sha256.New()
		hash.Write([]byte(rawStr))
		expectedSign := hex.EncodeToString(hash.Sum(nil))

		// 强比对签名是否一致（忽略大小写差异）
		if !strings.EqualFold(signature, expectedSign) {
			m.writeError(w, "签名验证失败")
			return
		}

		// 验证通过，放行后续逻辑
		next(w, r)
	}
}

// writeError 向 HTTP 响应中写回标准 JSON 包装的 401 签名错误回包
func (m *SignatureMiddleware) writeError(w http.ResponseWriter, msg string) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusUnauthorized)
	// 封装格式与项目全局 UniformResponse 的异常分支逻辑对齐
	respJSON := fmt.Sprintf(`{"code":401,"msg":%q}`, msg)
	w.Write([]byte(respJSON))
}
