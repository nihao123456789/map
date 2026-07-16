// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.1

// Package logic 包含所有业务逻辑的实现。
// QueryMapLogic 实现了根据地图经纬度范围查询堆场和集装箱的核心逻辑。
package logic

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/zeromicro/go-zero/core/logx"

	mysqlModel "map-server/internal/model/mysql/map_server"
	"map-server/internal/svc"
	"map-server/internal/types"
	"map-server/pkg/geoutil"
)

// Redis GEO key 常量定义
const (
	// geoKeyYards 是存储所有堆场位置的 Redis GEO key
	// 成员格式：堆场 ID（如 "1"、"2"）
	geoKeyYards = "geo:yards"

	// geoKeyContainers 是存储所有集装箱位置的 Redis GEO key
	// 成员格式：集装箱 ID（如 "1"、"2"）
	geoKeyContainers = "geo:containers"
)

// QueryMapLogic 是地图范围查询的业务逻辑结构体，
// 内嵌 logx.Logger 用于日志输出，持有服务上下文和请求上下文。
type QueryMapLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// NewQueryMapLogic 创建并返回 QueryMapLogic 实例。
//
// 参数：
//   - ctx：请求上下文（包含 traceId 等信息）
//   - svcCtx：服务上下文（包含数据库、Redis 等依赖）
func NewQueryMapLogic(ctx context.Context, svcCtx *svc.ServiceContext) *QueryMapLogic {
	return &QueryMapLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

// QueryMap 根据中心点经纬度和查询半径，返回范围内的堆场与集装箱列表。
//
// 查询流程：
//  1. 以中心点（longitude, latitude）为圆心，radius 千米为半径，
//     从 Redis GEO 中查询范围内的堆场和集装箱成员 ID。
//  2. 若请求中提供了矩形视口边界（minLon/maxLon/minLat/maxLat），
//     则对 Redis 查询结果进行矩形过滤，去除不在视口内的数据。
//  3. 根据过滤后的 ID 列表，从 MySQL 批量查询详细信息。
//  4. 将 Redis 中的距离信息合并到 MySQL 查询结果中，组装最终响应。
//
// 参数：
//   - req：查询请求参数（中心点经纬度、半径、可选矩形边界）
//
// 返回：查询响应（堆场列表 + 集装箱列表），以及错误信息。
func (l *QueryMapLogic) QueryMap(req *types.MapQueryReq) (resp *types.MapQueryResp, err error) {
	// 打印查询参数日志
	l.Logger.Infof("地图查询请求：中心点=[%.6f, %.6f] 半径=%.2fkm", req.Longitude, req.Latitude, req.Radius)

	// ------------------------------------------------------------------
	// 第一步：从 Redis GEO 查询指定半径范围内的堆场
	// ------------------------------------------------------------------
	yardGeoResults, err := l.svcCtx.GeoClient.SearchRadius(
		l.ctx,
		geoKeyYards,
		req.Longitude,
		req.Latitude,
		req.Radius,
	)
	if err != nil {
		l.Logger.Errorf("查询 Redis 堆场位置失败: %v", err)
		return nil, fmt.Errorf("查询堆场位置失败: %w", err)
	}
	l.Logger.Infof("Redis GEO 查询到 %d 个堆场", len(yardGeoResults))

	// ------------------------------------------------------------------
	// 第二步：从 Redis GEO 查询指定半径范围内的集装箱
	// ------------------------------------------------------------------
	containerGeoResults, err := l.svcCtx.GeoClient.SearchRadius(
		l.ctx,
		geoKeyContainers,
		req.Longitude,
		req.Latitude,
		req.Radius,
	)
	if err != nil {
		l.Logger.Errorf("查询 Redis 集装箱位置失败: %v", err)
		return nil, fmt.Errorf("查询集装箱位置失败: %w", err)
	}
	l.Logger.Infof("Redis GEO 查询到 %d 个集装箱", len(containerGeoResults))

	// ------------------------------------------------------------------
	// 第三步：判断是否需要进行矩形视口过滤
	// 当请求中同时提供了 minLon/maxLon/minLat/maxLat 时才执行过滤
	// ------------------------------------------------------------------
	needBBoxFilter := req.MinLon != 0 || req.MaxLon != 0 || req.MinLat != 0 || req.MaxLat != 0
	if needBBoxFilter {
		l.Logger.Infof("启用矩形视口过滤：经度[%.6f~%.6f] 纬度[%.6f~%.6f]",
			req.MinLon, req.MaxLon, req.MinLat, req.MaxLat)
		yardGeoResults = filterByBBox(yardGeoResults, req.MinLon, req.MaxLon, req.MinLat, req.MaxLat)
		containerGeoResults = filterByBBox(containerGeoResults, req.MinLon, req.MaxLon, req.MinLat, req.MaxLat)
		l.Logger.Infof("矩形过滤后：堆场 %d 个，集装箱 %d 个",
			len(yardGeoResults), len(containerGeoResults))
	}

	// ------------------------------------------------------------------
	// 第四步：解析 GEO 查询结果中的 ID，并构建 ID->距离 的映射表
	// ------------------------------------------------------------------
	// 堆场 ID 列表 和 ID->距离 映射
	yardIds, yardDistMap := parseIdsAndDistMap(yardGeoResults)
	// 集装箱 ID 列表 和 ID->距离 映射
	containerIds, containerDistMap := parseIdsAndDistMap(containerGeoResults)

	// ------------------------------------------------------------------
	// 第五步：根据 ID 列表从 MySQL 批量查询堆场详细信息
	// ------------------------------------------------------------------
	yards, err := l.svcCtx.YardModel.FindByIds(l.ctx, yardIds)
	if err != nil {
		l.Logger.Errorf("MySQL 查询堆场失败: %v", err)
		return nil, fmt.Errorf("查询堆场详情失败: %w", err)
	}

	// ------------------------------------------------------------------
	// 第六步：根据 ID 列表从 MySQL 批量查询集装箱详细信息
	// ------------------------------------------------------------------
	containers, err := l.svcCtx.ContainerModel.FindByIds(l.ctx, containerIds)
	if err != nil {
		l.Logger.Errorf("MySQL 查询集装箱失败: %v", err)
		return nil, fmt.Errorf("查询集装箱详情失败: %w", err)
	}

	// ------------------------------------------------------------------
	// 第七步：组装响应数据，将 MySQL 详情与 Redis 距离信息合并
	// ------------------------------------------------------------------
	resp = &types.MapQueryResp{
		Yards:      buildYardInfoList(yards, yardDistMap),
		Containers: buildContainerInfoList(containers, containerDistMap),
	}

	l.Logger.Infof("地图查询完成：返回 %d 个堆场，%d 个集装箱", len(resp.Yards), len(resp.Containers))
	return resp, nil
}

// filterByBBox 对 GEO 查询结果按矩形视口进行过滤，
// 只保留经纬度在 [minLon,maxLon] x [minLat,maxLat] 范围内的成员。
//
// 参数：
//   - results：待过滤的 GEO 结果列表
//   - minLon/maxLon：经度范围
//   - minLat/maxLat：纬度范围
//
// 返回：过滤后的 GEO 结果列表。
func filterByBBox(results []geoutil.GeoResult, minLon, maxLon, minLat, maxLat float64) []geoutil.GeoResult {
	filtered := make([]geoutil.GeoResult, 0, len(results))
	for _, r := range results {
		// 判断该成员是否在矩形视口范围内
		if r.Longitude >= minLon && r.Longitude <= maxLon &&
			r.Latitude >= minLat && r.Latitude <= maxLat {
			filtered = append(filtered, r)
		}
	}
	return filtered
}

// parseIdsAndDistMap 从 GEO 查询结果中提取 ID 列表，
// 并构建 ID -> 距离（千米）的映射表，供后续使用。
//
// GEO 成员的 Member 字段存储的是字符串格式的 ID（如 "1"、"2"）。
//
// 参数：
//   - results：GEO 查询结果列表
//
// 返回：ID 列表，以及 ID 到距离的映射。
func parseIdsAndDistMap(results []geoutil.GeoResult) ([]int64, map[int64]float64) {
	ids := make([]int64, 0, len(results))
	distMap := make(map[int64]float64, len(results))

	for _, r := range results {
		// 去除可能的前缀（成员名如果是纯数字 ID）
		idStr := strings.TrimSpace(r.Member)
		id, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			// 若成员名不是有效的 ID，跳过并记录日志
			logx.Errorf("解析 GEO 成员 ID 失败，Member=%s: %v", r.Member, err)
			continue
		}
		ids = append(ids, id)
		distMap[id] = r.Distance
	}
	return ids, distMap
}

// buildYardInfoList 将 MySQL 查询到的堆场列表与距离映射合并，
// 构建接口响应所需的 YardInfo 列表。
//
// 参数：
//   - yards：MySQL 查询的堆场实体列表
//   - distMap：堆场 ID -> 距离（千米）的映射表
//
// 返回：YardInfo 列表。
func buildYardInfoList(yards []*mysqlModel.Yard, distMap map[int64]float64) []types.YardInfo {
	list := make([]types.YardInfo, 0, len(yards))
	for _, y := range yards {
		list = append(list, types.YardInfo{
			Id:        y.Id,
			Name:      y.Name,
			Longitude: y.Longitude,
			Latitude:  y.Latitude,
			Distance:  distMap[y.Id], // 从 Redis GEO 查询结果中获取距离
		})
	}
	return list
}

// buildContainerInfoList 将 MySQL 查询到的集装箱列表与距离映射合并，
// 构建接口响应所需的 ContainerInfo 列表。
//
// 参数：
//   - containers：MySQL 查询的集装箱实体列表
//   - distMap：集装箱 ID -> 距离（千米）的映射表
//
// 返回：ContainerInfo 列表。
func buildContainerInfoList(containers []*mysqlModel.Container, distMap map[int64]float64) []types.ContainerInfo {
	list := make([]types.ContainerInfo, 0, len(containers))
	for _, c := range containers {
		list = append(list, types.ContainerInfo{
			Id:        c.Id,
			Number:    c.Number,
			YardId:    c.YardId,
			Longitude: c.Longitude,
			Latitude:  c.Latitude,
			Distance:  distMap[c.Id], // 从 Redis GEO 查询结果中获取距离
		})
	}
	return list
}
