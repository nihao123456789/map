package handler

import (
	"net/http"

	"github.com/zeromicro/go-zero/rest/httpx"
	"map-server/internal/logic"
	"map-server/internal/svc"
)

// GetLocationListHandler 处理获取热门位置列表的 HTTP 请求
func GetLocationListHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		l := logic.NewGetLocationListLogic(r.Context(), svcCtx)
		resp, err := l.GetLocationList()
		if err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
		} else {
			httpx.OkJsonCtx(r.Context(), w, resp)
		}
	}
}
