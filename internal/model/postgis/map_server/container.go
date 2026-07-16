// Package map_server 提供对 PostgreSQL+PostGIS 空间数据库 map_server 中 container 表的数据访问。
package map_server

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

// PostGISContainer 对应 PostgreSQL 中的 container 集装箱空间表查询结果。
type PostGISContainer struct {
	// Id 集装箱主键ID
	Id int64
	// Number 集装箱编号
	Number string
	// YardId 所属堆场ID
	YardId int64
	// Longitude 经度
	Longitude float64
	// Latitude 纬度
	Latitude float64
	// DistanceKm 距查询中心点的距离（千米）
	DistanceKm float64
}

// PostGISContainerModel 提供对 PostgreSQL container 表的空间查询操作。
type PostGISContainerModel struct {
	// pool 是 pgx 的连接池
	pool *pgxpool.Pool
}

// NewPostGISContainerModel 创建并返回 PostGISContainerModel 实例。
//
// 参数：
//   - pool：pgx 连接池
func NewPostGISContainerModel(pool *pgxpool.Pool) *PostGISContainerModel {
	return &PostGISContainerModel{pool: pool}
}

// FindByBBox 使用矩形视口查询范围内的所有在场状态集装箱。
//
// 参数：
//   - ctx：上下文
//   - centerLon：中心点经度
//   - centerLat：中心点纬度
//   - minLon/maxLon：经度边界
//   - minLat/maxLat：纬度边界
//
// 返回：矩形范围内的集装箱列表（按距离升序），以及错误信息。
func (m *PostGISContainerModel) FindByBBox(
	ctx context.Context,
	centerLon, centerLat float64,
	minLon, maxLon, minLat, maxLat float64,
) ([]*PostGISContainer, error) {
	query := `
		SELECT
			id,
			number,
			yard_id,
			ST_X(location::geometry)  AS longitude,
			ST_Y(location::geometry)  AS latitude,
			ST_Distance(location, ST_MakePoint($1, $2)::geography) / 1000.0 AS distance_km
		FROM container
		WHERE status = 1
		  AND ST_Within(
			  location::geometry,
			  ST_MakeEnvelope($3, $4, $5, $6, 4326)
		  )
		ORDER BY distance_km ASC`

	rows, err := m.pool.Query(ctx, query, centerLon, centerLat, minLon, minLat, maxLon, maxLat)
	if err != nil {
		return nil, fmt.Errorf("PostGISContainerModel.FindByBBox 查询失败: %w", err)
	}
	defer rows.Close()

	return scanContainerRows(rows)
}

// FindByRadius 使用圆形半径查询范围内的所有在场状态集装箱。
//
// 参数：
//   - ctx：上下文
//   - centerLon：中心点经度
//   - centerLat：中心点纬度
//   - radiusKm：查询半径（千米）
//
// 返回：半径范围内的集装箱列表（按距离升序），以及错误信息。
func (m *PostGISContainerModel) FindByRadius(
	ctx context.Context,
	centerLon, centerLat, radiusKm float64,
) ([]*PostGISContainer, error) {
	query := `
		SELECT
			id,
			number,
			yard_id,
			ST_X(location::geometry)  AS longitude,
			ST_Y(location::geometry)  AS latitude,
			ST_Distance(location, ST_MakePoint($1, $2)::geography) / 1000.0 AS distance_km
		FROM container
		WHERE status = 1
		  AND ST_DWithin(location, ST_MakePoint($1, $2)::geography, $3 * 1000)
		ORDER BY distance_km ASC`

	rows, err := m.pool.Query(ctx, query, centerLon, centerLat, radiusKm)
	if err != nil {
		return nil, fmt.Errorf("PostGISContainerModel.FindByRadius 查询失败: %w", err)
	}
	defer rows.Close()

	return scanContainerRows(rows)
}

// scanContainerRows 将 pgx Rows 扫描为 PostGISContainer 列表（内部复用函数）。
func scanContainerRows(rows interface{ Next() bool; Scan(...any) error; Err() error }) ([]*PostGISContainer, error) {
	var containers []*PostGISContainer
	for rows.Next() {
		c := &PostGISContainer{}
		if err := rows.Scan(&c.Id, &c.Number, &c.YardId, &c.Longitude, &c.Latitude, &c.DistanceKm); err != nil {
			return nil, fmt.Errorf("扫描集装箱行数据失败: %w", err)
		}
		containers = append(containers, c)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("读取集装箱查询结果失败: %w", err)
	}
	return containers, nil
}

// Upsert 插入或更新集装箱空间信息。
// 如果 ID 已存在，则更新箱号、堆场 ID、地理位置、状态及更新时间。
func (m *PostGISContainerModel) Upsert(ctx context.Context, id int64, number string, yardId int64, longitude, latitude float64, status int16) error {
	query := `
		INSERT INTO container (id, number, yard_id, location, status, created_at, updated_at)
		VALUES ($1, $2, $3, ST_MakePoint($4, $5)::geography, $6, NOW(), NOW())
		ON CONFLICT (id) DO UPDATE SET
			number = EXCLUDED.number,
			yard_id = EXCLUDED.yard_id,
			location = EXCLUDED.location,
			status = EXCLUDED.status,
			updated_at = NOW()`
	_, err := m.pool.Exec(ctx, query, id, number, yardId, longitude, latitude, status)
	if err != nil {
		return fmt.Errorf("PostGISContainerModel.Upsert 失败: %w", err)
	}
	return nil
}

// Delete 根据 ID 物理删除集装箱空间数据。
func (m *PostGISContainerModel) Delete(ctx context.Context, id int64) error {
	query := `DELETE FROM container WHERE id = $1`
	_, err := m.pool.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("PostGISContainerModel.Delete 失败: %w", err)
	}
	return nil
}
