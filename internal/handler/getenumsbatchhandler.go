package handler

import (
	"net/http"

	"github.com/zeromicro/go-zero/rest/httpx"
	"map-server/internal/logic"
	"map-server/internal/svc"
	"map-server/internal/types"
)

// GetEnumsBatchHandler 处理批量获取系统字典项的 HTTP 请求
func GetEnumsBatchHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.EnumsBatchReq
		if err := httpx.Parse(r, &req); err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}

		l := logic.NewGetEnumsBatchLogic(r.Context(), svcCtx)
		resp, err := l.GetEnumsBatch(&req)
		if err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
		} else {
			httpx.OkJsonCtx(r.Context(), w, resp)
		}
	}
}
