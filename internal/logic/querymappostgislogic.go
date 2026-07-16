// Package logic 包含所有业务逻辑的实现。
// QueryMapPostGISLogic 实现了基于 PostgreSQL+PostGIS 的精确空间查询逻辑。
//
// 与 QueryMapLogic（Redis-GEO 版本）的区别：
//   - 优先使用矩形视口（BBox）查询，完全匹配地图前端的可视范围
//   - 精确球面距离计算，无 Geohash 近似误差
//   - 一次 SQL 同时返回业务数据 + 距离，无需二次查询 MySQL
package logic

import (
	"context"
	"fmt"

	"github.com/zeromicro/go-zero/core/logx"

	postgisModel "map-server/internal/model/postgis/map_server"
	"map-server/internal/svc"
	"map-server/internal/types"
)

// QueryMapPostGISLogic 是基于 PostGIS 的地图范围查询业务逻辑结构体。
type QueryMapPostGISLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// NewQueryMapPostGISLogic 创建并返回 QueryMapPostGISLogic 实例。
//
// 参数：
//   - ctx：请求上下文（包含 traceId 等信息）
//   - svcCtx：服务上下文（包含 PostgreSQL 连接池等依赖）
func NewQueryMapPostGISLogic(ctx context.Context, svcCtx *svc.ServiceContext) *QueryMapPostGISLogic {
	return &QueryMapPostGISLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

// QueryMapPostGIS 根据请求参数，使用 PostgreSQL+PostGIS 执行精确空间查询。
//
// 查询策略（优先级从高到低）：
//  1. 若请求中同时提供了 minLon/maxLon/minLat/maxLat，
//     则使用【矩形视口（BBox）查询】——完全对应地图前端拖拽/缩放的可视区域。
//  2. 否则，使用【圆形半径查询】（longitude/latitude/radius）——兼容 Redis-GEO 接口语义。
//
// 与 Redis-GEO 版本的核心区别：
//   - PostGIS 的 ST_Distance 基于球面精确计算，误差极小（±0.5%以内）
//   - 一次 SQL 同时返回所有字段 + 精确距离，不需要分两步查 Redis + MySQL
//   - 支持原生矩形范围查询，无需近似圆形 + 二次过滤
//
// 参数：
//   - req：查询请求（与 Redis-GEO 接口共用同一结构体）
//
// 返回：查询响应（堆场列表 + 集装箱列表），以及错误信息。
func (l *QueryMapPostGISLogic) QueryMapPostGIS(req *types.MapQueryReq) (resp *types.MapQueryResp, err error) {
	// 判断是否使用矩形视口查询（BBox 四个边界值均不为零时启用）
	useBBox := req.MinLon != 0 || req.MaxLon != 0 || req.MinLat != 0 || req.MaxLat != 0

	if useBBox {
		l.Logger.Infof("[PostGIS] 矩形视口查询：中心点=[%.6f, %.6f] 边界=[%.6f~%.6f, %.6f~%.6f]",
			req.Longitude, req.Latitude, req.MinLon, req.MaxLon, req.MinLat, req.MaxLat)
		return l.queryByBBox(req)
	}

	l.Logger.Infof("[PostGIS] 圆形半径查询：中心点=[%.6f, %.6f] 半径=%.2fkm",
		req.Longitude, req.Latitude, req.Radius)
	return l.queryByRadius(req)
}

// queryByBBox 使用矩形视口（BBox）查询范围内的堆场和集装箱。
//
// 调用 PostGIS ST_Within + ST_MakeEnvelope 实现精确矩形过滤。
func (l *QueryMapPostGISLogic) queryByBBox(req *types.MapQueryReq) (*types.MapQueryResp, error) {
	// 并发查询堆场和集装箱（两张表独立查询，可并行）
	// 通过 channel 收集结果
	type yardResult struct {
		yards []*postgisModel.PostGISYard
		err   error
	}
	type containerResult struct {
		containers []*postgisModel.PostGISContainer
		err        error
	}

	yardCh := make(chan yardResult, 1)
	containerCh := make(chan containerResult, 1)

	// 异步查询堆场
	go func() {
		yards, err := l.svcCtx.PostGISYardModel.FindByBBox(
			l.ctx,
			req.Longitude, req.Latitude,
			req.MinLon, req.MaxLon, req.MinLat, req.MaxLat,
		)
		yardCh <- yardResult{yards: yards, err: err}
	}()

	// 异步查询集装箱
	go func() {
		containers, err := l.svcCtx.PostGISContainerModel.FindByBBox(
			l.ctx,
			req.Longitude, req.Latitude,
			req.MinLon, req.MaxLon, req.MinLat, req.MaxLat,
		)
		containerCh <- containerResult{containers: containers, err: err}
	}()

	// 等待两个查询结果
	yr := <-yardCh
	cr := <-containerCh

	if yr.err != nil {
		l.Logger.Errorf("[PostGIS] BBox 查询堆场失败: %v", yr.err)
		return nil, fmt.Errorf("PostGIS 查询堆场失败: %w", yr.err)
	}
	if cr.err != nil {
		l.Logger.Errorf("[PostGIS] BBox 查询集装箱失败: %v", cr.err)
		return nil, fmt.Errorf("PostGIS 查询集装箱失败: %w", cr.err)
	}

	l.Logger.Infof("[PostGIS] BBox 查询完成：堆场 %d 个，集装箱 %d 个", len(yr.yards), len(cr.containers))

	return buildPostGISResp(yr.yards, cr.containers), nil
}

// queryByRadius 使用圆形半径查询范围内的堆场和集装箱。
//
// 调用 PostGIS ST_DWithin（GEOGRAPHY 类型，球面精确距离）实现。
func (l *QueryMapPostGISLogic) queryByRadius(req *types.MapQueryReq) (*types.MapQueryResp, error) {
	type yardResult struct {
		yards []*postgisModel.PostGISYard
		err   error
	}
	type containerResult struct {
		containers []*postgisModel.PostGISContainer
		err        error
	}

	yardCh := make(chan yardResult, 1)
	containerCh := make(chan containerResult, 1)

	// 异步查询堆场
	go func() {
		yards, err := l.svcCtx.PostGISYardModel.FindByRadius(
			l.ctx,
			req.Longitude, req.Latitude, req.Radius,
		)
		yardCh <- yardResult{yards: yards, err: err}
	}()

	// 异步查询集装箱
	go func() {
		containers, err := l.svcCtx.PostGISContainerModel.FindByRadius(
			l.ctx,
			req.Longitude, req.Latitude, req.Radius,
		)
		containerCh <- containerResult{containers: containers, err: err}
	}()

	yr := <-yardCh
	cr := <-containerCh

	if yr.err != nil {
		l.Logger.Errorf("[PostGIS] Radius 查询堆场失败: %v", yr.err)
		return nil, fmt.Errorf("PostGIS 查询堆场失败: %w", yr.err)
	}
	if cr.err != nil {
		l.Logger.Errorf("[PostGIS] Radius 查询集装箱失败: %v", cr.err)
		return nil, fmt.Errorf("PostGIS 查询集装箱失败: %w", cr.err)
	}

	l.Logger.Infof("[PostGIS] Radius 查询完成：堆场 %d 个，集装箱 %d 个", len(yr.yards), len(cr.containers))

	return buildPostGISResp(yr.yards, cr.containers), nil
}

// buildPostGISResp 将 PostGIS 查询结果转换为统一的 HTTP 响应结构。
// PostGIS 版本在一次 SQL 中已经包含了距离字段，无需合并额外的距离信息。
//
// 参数：
//   - yards：PostGIS 堆场查询结果
//   - containers：PostGIS 集装箱查询结果
//
// 返回：MapQueryResp 响应结构体。
func buildPostGISResp(yards []*postgisModel.PostGISYard, containers []*postgisModel.PostGISContainer) *types.MapQueryResp {
	// 组装堆场列表
	yardInfos := make([]types.YardInfo, 0, len(yards))
	for _, y := range yards {
		yardInfos = append(yardInfos, types.YardInfo{
			Id:        y.Id,
			Name:      y.Name,
			Longitude: y.Longitude,
			Latitude:  y.Latitude,
			Distance:  y.DistanceKm, // 直接来自 PostGIS ST_Distance，精确球面距离
		})
	}

	// 组装集装箱列表
	containerInfos := make([]types.ContainerInfo, 0, len(containers))
	for _, c := range containers {
		containerInfos = append(containerInfos, types.ContainerInfo{
			Id:        c.Id,
			Number:    c.Number,
			YardId:    c.YardId,
			Longitude: c.Longitude,
			Latitude:  c.Latitude,
			Distance:  c.DistanceKm,
		})
	}

	return &types.MapQueryResp{
		Yards:      yardInfos,
		Containers: containerInfos,
	}
}
