package handler

import (
	"net/http"

	"github.com/zeromicro/go-zero/rest/httpx"
	"map-server/internal/logic"
	"map-server/internal/svc"
	"map-server/internal/types"
)

// GetTradingLocationCountHandler 处理按位置统计挂单数量的 HTTP 请求
func GetTradingLocationCountHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.TradingLocationCountReq
		if err := httpx.Parse(r, &req); err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}

		l := logic.NewGetTradingLocationCountLogic(r.Context(), svcCtx)
		resp, err := l.GetTradingLocationCount(&req)
		if err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
		} else {
			httpx.OkJsonCtx(r.Context(), w, resp)
		}
	}
}
