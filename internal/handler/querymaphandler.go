// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.1

// Package handler 包含 HTTP 请求处理器。
// QueryMapHandler 负责处理地图范围查询的 HTTP 请求，
// 完成参数解析、调用业务逻辑层、返回 JSON 响应的完整流程。
package handler

import (
	"net/http"

	"github.com/zeromicro/go-zero/rest/httpx"
	"map-server/internal/logic"
	"map-server/internal/svc"
	"map-server/internal/types"
)

// QueryMapHandler 是 GET /api/map/query 接口的 HTTP 处理器工厂函数。
// 返回一个标准的 http.HandlerFunc 闭包，通过闭包持有服务上下文 svcCtx。
//
// 请求参数（Query String）：
//   - longitude：中心点经度（必填）
//   - latitude：中心点纬度（必填）
//   - radius：查询半径，单位千米（必填）
//   - minLon/maxLon/minLat/maxLat：矩形视口边界（可选）
//
// 响应（JSON）：
//   - yards：范围内的堆场列表
//   - containers：范围内的集装箱列表
func QueryMapHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// 第一步：解析并绑定请求参数
		// httpx.Parse 会自动从 Query String 中提取参数并校验必填字段
		var req types.MapQueryReq
		if err := httpx.Parse(r, &req); err != nil {
			// 参数校验失败，返回 400 Bad Request
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}

		// 第二步：创建业务逻辑实例并执行查询
		l := logic.NewQueryMapLogic(r.Context(), svcCtx)
		resp, err := l.QueryMap(&req)
		if err != nil {
			// 业务逻辑出错，返回 500 Internal Server Error
			httpx.ErrorCtx(r.Context(), w, err)
		} else {
			// 成功，返回 200 OK 和 JSON 格式的响应体
			httpx.OkJsonCtx(r.Context(), w, resp)
		}
	}
}
