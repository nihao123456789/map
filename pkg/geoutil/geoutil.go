// Package geoutil 封装了 Redis GEO 命令的常用操作，
// 提供地理位置的增、删、查功能，便于在业务层中使用。
//
// Redis GEO 基于 Sorted Set 结构存储，底层使用 Geohash 算法编码经纬度，
// 本工具类使用 go-redis/v9 客户端实现。
package geoutil

import (
	"context"
	"fmt"

	"github.com/redis/go-redis/v9"
)

// GeoMember 表示一个地理位置成员，包含成员名称和经纬度坐标。
type GeoMember struct {
	// Member 是唯一标识该位置的成员名，例如 "yard:1"、"container:2"
	Member string
	// Longitude 是经度（WGS84坐标系，范围：-180 ~ +180）
	Longitude float64
	// Latitude 是纬度（WGS84坐标系，范围：-85.05112878 ~ +85.05112878）
	Latitude float64
}

// GeoResult 是查询结果，包含成员名称、坐标及距中心点的距离。
type GeoResult struct {
	// Member 成员名称
	Member string
	// Longitude 经度
	Longitude float64
	// Latitude 纬度
	Latitude float64
	// Distance 距查询中心点的距离，单位为千米（km）
	Distance float64
}

// GeoClient 是 Redis GEO 操作的客户端封装，持有 go-redis 的客户端实例。
type GeoClient struct {
	// rdb 是底层的 go-redis 客户端
	rdb *redis.Client
}

// NewGeoClient 创建并返回一个新的 GeoClient 实例。
//
// 参数：
//   - rdb：已初始化的 go-redis 客户端
func NewGeoClient(rdb *redis.Client) *GeoClient {
	return &GeoClient{rdb: rdb}
}

// AddLocation 向指定的 GEO key 中添加单个地理位置。
//
// 参数：
//   - ctx：上下文
//   - key：Redis GEO key 名称（如 "geo:yards"）
//   - member：成员名称（如 "1"、"yard:1"）
//   - longitude：经度
//   - latitude：纬度
//
// 返回：添加的成员数量（新增为1，更新已有成员为0），以及错误信息。
func (g *GeoClient) AddLocation(ctx context.Context, key, member string, longitude, latitude float64) (int64, error) {
	result, err := g.rdb.GeoAdd(ctx, key, &redis.GeoLocation{
		Name:      member,
		Longitude: longitude,
		Latitude:  latitude,
	}).Result()
	if err != nil {
		return 0, fmt.Errorf("GeoClient.AddLocation 失败，key=%s member=%s: %w", key, member, err)
	}
	return result, nil
}

// BatchAddLocations 向指定的 GEO key 中批量添加多个地理位置。
// 使用单条 GEOADD 命令完成，效率高于逐个添加。
//
// 参数：
//   - ctx：上下文
//   - key：Redis GEO key 名称
//   - members：要添加的地理位置成员列表
//
// 返回：新增的成员数量，以及错误信息。
func (g *GeoClient) BatchAddLocations(ctx context.Context, key string, members []GeoMember) (int64, error) {
	if len(members) == 0 {
		return 0, nil
	}

	// 将 GeoMember 列表转换为 go-redis 所需的 GeoLocation 格式
	locations := make([]*redis.GeoLocation, 0, len(members))
	for _, m := range members {
		locations = append(locations, &redis.GeoLocation{
			Name:      m.Member,
			Longitude: m.Longitude,
			Latitude:  m.Latitude,
		})
	}

	result, err := g.rdb.GeoAdd(ctx, key, locations...).Result()
	if err != nil {
		return 0, fmt.Errorf("GeoClient.BatchAddLocations 失败，key=%s 成员数=%d: %w", key, len(members), err)
	}
	return result, nil
}

// RemoveLocation 从指定的 GEO key 中删除一个成员。
// GEO key 底层是 Sorted Set，删除操作使用 ZREM 命令。
//
// 参数：
//   - ctx：上下文
//   - key：Redis GEO key 名称
//   - member：要删除的成员名称
//
// 返回：删除的成员数量，以及错误信息。
func (g *GeoClient) RemoveLocation(ctx context.Context, key, member string) (int64, error) {
	result, err := g.rdb.ZRem(ctx, key, member).Result()
	if err != nil {
		return 0, fmt.Errorf("GeoClient.RemoveLocation 失败，key=%s member=%s: %w", key, member, err)
	}
	return result, nil
}

// SearchRadius 以指定的中心点经纬度为圆心、radiusKm 为半径（千米），
// 查询 GEO key 中所有位于该范围内的成员，并返回其坐标和距离信息。
//
// 底层使用 Redis 6.2+ 的 GEOSEARCH 命令（FROMLONLAT BYRADIUS 模式）。
//
// 参数：
//   - ctx：上下文
//   - key：Redis GEO key 名称
//   - longitude：中心点经度
//   - latitude：中心点纬度
//   - radiusKm：查询半径，单位：千米
//
// 返回：范围内的成员列表（含坐标和距离），以及错误信息。
func (g *GeoClient) SearchRadius(ctx context.Context, key string, longitude, latitude, radiusKm float64) ([]GeoResult, error) {
	// 使用 GEOSEARCH 命令，返回坐标和距离信息
	locations, err := g.rdb.GeoSearchLocation(ctx, key, &redis.GeoSearchLocationQuery{
		GeoSearchQuery: redis.GeoSearchQuery{
			// 中心点坐标
			Longitude: longitude,
			Latitude:  latitude,
			// 搜索半径（千米）
			Radius:     radiusKm,
			RadiusUnit: "km",
			// 按距离升序排列
			Sort: "ASC",
		},
		// 返回坐标信息
		WithCoord: true,
		// 返回距中心点的距离（千米）
		WithDist: true,
	}).Result()
	if err != nil {
		return nil, fmt.Errorf("GeoClient.SearchRadius 失败，key=%s 中心点=[%.6f,%.6f] 半径=%.2fkm: %w",
			key, longitude, latitude, radiusKm, err)
	}

	// 将查询结果转换为业务层使用的 GeoResult 列表
	results := make([]GeoResult, 0, len(locations))
	for _, loc := range locations {
		results = append(results, GeoResult{
			Member:    loc.Name,
			Longitude: loc.Longitude,
			Latitude:  loc.Latitude,
			Distance:  loc.Dist,
		})
	}
	return results, nil
}

// GetPosition 获取 GEO key 中指定成员的经纬度坐标。
//
// 参数：
//   - ctx：上下文
//   - key：Redis GEO key 名称
//   - member：成员名称
//
// 返回：经度、纬度，以及错误信息。若成员不存在，返回 (0, 0, nil)。
func (g *GeoClient) GetPosition(ctx context.Context, key, member string) (longitude, latitude float64, err error) {
	positions, err := g.rdb.GeoPos(ctx, key, member).Result()
	if err != nil {
		return 0, 0, fmt.Errorf("GeoClient.GetPosition 失败，key=%s member=%s: %w", key, member, err)
	}
	// GeoPos 返回切片，每个元素对应一个成员（不存在时为 nil）
	if len(positions) == 0 || positions[0] == nil {
		return 0, 0, nil
	}
	return positions[0].Longitude, positions[0].Latitude, nil
}

// GetDistance 计算 GEO key 中两个成员之间的距离，单位：千米。
//
// 参数：
//   - ctx：上下文
//   - key：Redis GEO key 名称
//   - member1：第一个成员名称
//   - member2：第二个成员名称
//
// 返回：两成员之间的距离（千米），以及错误信息。
func (g *GeoClient) GetDistance(ctx context.Context, key, member1, member2 string) (float64, error) {
	dist, err := g.rdb.GeoDist(ctx, key, member1, member2, "km").Result()
	if err != nil {
		return 0, fmt.Errorf("GeoClient.GetDistance 失败，key=%s m1=%s m2=%s: %w", key, member1, member2, err)
	}
	return dist, nil
}
