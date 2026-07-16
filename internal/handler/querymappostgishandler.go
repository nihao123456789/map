// Package handler 包含 HTTP 请求处理器。
// QueryMapPostGISHandler 处理基于 PostgreSQL+PostGIS 的精确空间查询请求。
package handler

import (
	"net/http"

	"github.com/zeromicro/go-zero/rest/httpx"

	"map-server/internal/logic"
	"map-server/internal/svc"
	"map-server/internal/types"
)

// QueryMapPostGISHandler 是 GET /api/map/query/postgis 接口的 HTTP 处理器工厂函数。
//
// 与 QueryMapHandler（Redis-GEO 版）对比：
//   - 共用相同的请求/响应结构体（MapQueryReq / MapQueryResp）
//   - 内部调用 PostGIS 空间查询，支持精确矩形视口和球面距离计算
//
// 请求参数（Query String）：
//   - longitude：中心点经度（必填）
//   - latitude：中心点纬度（必填）
//   - radius：查询半径，单位千米（BBox 模式可选，Radius 模式必填）
//   - minLon/maxLon/minLat/maxLat：矩形视口边界（提供时优先使用 BBox 查询）
//
// 查询策略：
//   - 提供 minLon/maxLon/minLat/maxLat → 使用 PostGIS ST_Within 矩形查询
//   - 仅提供 longitude/latitude/radius  → 使用 PostGIS ST_DWithin 球面半径查询
func QueryMapPostGISHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// 第一步：解析并绑定请求参数
		var req types.MapQueryReq
		if err := httpx.Parse(r, &req); err != nil {
			// 参数校验失败，返回 400 Bad Request
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}

		// 第二步：创建 PostGIS 查询逻辑实例并执行
		l := logic.NewQueryMapPostGISLogic(r.Context(), svcCtx)
		resp, err := l.QueryMapPostGIS(&req)
		if err != nil {
			// 查询失败，返回 500 Internal Server Error
			httpx.ErrorCtx(r.Context(), w, err)
		} else {
			// 成功，返回 200 OK 和 JSON 格式响应体
			httpx.OkJsonCtx(r.Context(), w, resp)
		}
	}
}
