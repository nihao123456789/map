// Package model 中的 PostGIS Model 层，
// 封装基于 PostgreSQL+PostGIS 的空间查询操作。
// 使用 jackc/pgx/v5 驱动的连接池（pgxpool）执行 SQL。
package model

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

// PostGISYard 对应 PostgreSQL 中的 yard 堆场空间表查询结果。
type PostGISYard struct {
	// Id 堆场主键ID
	Id int64
	// Name 堆场名称
	Name string
	// Address 详细地址
	Address string
	// Longitude 经度（从 GEOGRAPHY 列解析）
	Longitude float64
	// Latitude 纬度（从 GEOGRAPHY 列解析）
	Latitude float64
	// DistanceKm 距查询中心点的距离（千米），由 PostGIS ST_Distance 计算
	DistanceKm float64
}

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

// PostGISYardModel 提供对 PostgreSQL yard 表的空间查询操作。
type PostGISYardModel struct {
	// pool 是 pgx 的连接池，线程安全，可并发使用
	pool *pgxpool.Pool
}

// PostGISContainerModel 提供对 PostgreSQL container 表的空间查询操作。
type PostGISContainerModel struct {
	// pool 是 pgx 的连接池
	pool *pgxpool.Pool
}

// NewPostGISYardModel 创建并返回 PostGISYardModel 实例。
//
// 参数：
//   - pool：pgx 连接池
func NewPostGISYardModel(pool *pgxpool.Pool) *PostGISYardModel {
	return &PostGISYardModel{pool: pool}
}

// NewPostGISContainerModel 创建并返回 PostGISContainerModel 实例。
//
// 参数：
//   - pool：pgx 连接池
func NewPostGISContainerModel(pool *pgxpool.Pool) *PostGISContainerModel {
	return &PostGISContainerModel{pool: pool}
}

// FindByBBox 使用矩形视口（Bounding Box）查询范围内的所有启用状态堆场。
//
// 使用 PostGIS 的 ST_Within + ST_MakeEnvelope 实现精确矩形查询，
// 同时通过 ST_Distance 计算每个堆场到中心点的距离（千米）。
//
// 参数：
//   - ctx：上下文
//   - centerLon：中心点经度（用于计算距离）
//   - centerLat：中心点纬度（用于计算距离）
//   - minLon/maxLon：经度范围（矩形左右边界）
//   - minLat/maxLat：纬度范围（矩形上下边界）
//
// 返回：矩形范围内的堆场列表（按距离升序），以及错误信息。
func (m *PostGISYardModel) FindByBBox(
	ctx context.Context,
	centerLon, centerLat float64,
	minLon, maxLon, minLat, maxLat float64,
) ([]*PostGISYard, error) {
	// SQL 说明：
	//   ST_X(location::geometry)  — 从 GEOGRAPHY 中提取经度
	//   ST_Y(location::geometry)  — 从 GEOGRAPHY 中提取纬度
	//   ST_Distance(location, 中心点) — 球面距离（米），除以 1000 转换为千米
	//   ST_Within(location::geometry, ST_MakeEnvelope(...)) — 判断点是否在矩形内
	//   ST_MakeEnvelope(minLon, minLat, maxLon, maxLat, 4326) — 构建矩形范围（WGS84）
	query := `
		SELECT
			id,
			name,
			address,
			ST_X(location::geometry)  AS longitude,
			ST_Y(location::geometry)  AS latitude,
			ST_Distance(location, ST_MakePoint($1, $2)::geography) / 1000.0 AS distance_km
		FROM yard
		WHERE status = 1
		  AND ST_Within(
			  location::geometry,
			  ST_MakeEnvelope($3, $4, $5, $6, 4326)
		  )
		ORDER BY distance_km ASC`

	rows, err := m.pool.Query(ctx, query, centerLon, centerLat, minLon, minLat, maxLon, maxLat)
	if err != nil {
		return nil, fmt.Errorf("PostGISYardModel.FindByBBox 查询失败: %w", err)
	}
	defer rows.Close()

	return scanYardRows(rows)
}

// FindByRadius 使用圆形半径查询范围内的所有启用状态堆场。
//
// 使用 PostGIS 的 ST_DWithin（基于 GEOGRAPHY 类型，单位：米）实现精确球面距离查询。
//
// 参数：
//   - ctx：上下文
//   - centerLon：中心点经度
//   - centerLat：中心点纬度
//   - radiusKm：查询半径（千米）
//
// 返回：半径范围内的堆场列表（按距离升序），以及错误信息。
func (m *PostGISYardModel) FindByRadius(
	ctx context.Context,
	centerLon, centerLat, radiusKm float64,
) ([]*PostGISYard, error) {
	// ST_DWithin(geography_a, geography_b, 距离米) — 判断两个地理点的球面距离是否在范围内
	// 注意：GEOGRAPHY 类型的 ST_DWithin 参数单位是【米】，需将千米转换为米（×1000）
	query := `
		SELECT
			id,
			name,
			address,
			ST_X(location::geometry)  AS longitude,
			ST_Y(location::geometry)  AS latitude,
			ST_Distance(location, ST_MakePoint($1, $2)::geography) / 1000.0 AS distance_km
		FROM yard
		WHERE status = 1
		  AND ST_DWithin(location, ST_MakePoint($1, $2)::geography, $3 * 1000)
		ORDER BY distance_km ASC`

	rows, err := m.pool.Query(ctx, query, centerLon, centerLat, radiusKm)
	if err != nil {
		return nil, fmt.Errorf("PostGISYardModel.FindByRadius 查询失败: %w", err)
	}
	defer rows.Close()

	return scanYardRows(rows)
}

// scanYardRows 将 pgx Rows 扫描为 PostGISYard 列表（内部复用函数）。
func scanYardRows(rows interface{ Next() bool; Scan(...any) error; Err() error }) ([]*PostGISYard, error) {
	var yards []*PostGISYard
	for rows.Next() {
		y := &PostGISYard{}
		if err := rows.Scan(&y.Id, &y.Name, &y.Address, &y.Longitude, &y.Latitude, &y.DistanceKm); err != nil {
			return nil, fmt.Errorf("扫描堆场行数据失败: %w", err)
		}
		yards = append(yards, y)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("读取堆场查询结果失败: %w", err)
	}
	return yards, nil
}

// FindByBBox 使用矩形视口查询范围内的所有在场状态集装箱。
//
// 参数说明同 PostGISYardModel.FindByBBox。
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
// 参数说明同 PostGISYardModel.FindByRadius。
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

// Upsert 插入或更新堆场空间信息。
// 如果 ID 已存在，则更新名称、地址、地理位置、状态及更新时间。
func (m *PostGISYardModel) Upsert(ctx context.Context, id int64, name, address string, longitude, latitude float64, status int16) error {
	query := `
		INSERT INTO yard (id, name, address, location, status, created_at, updated_at)
		VALUES ($1, $2, $3, ST_MakePoint($4, $5)::geography, $6, NOW(), NOW())
		ON CONFLICT (id) DO UPDATE SET
			name = EXCLUDED.name,
			address = EXCLUDED.address,
			location = EXCLUDED.location,
			status = EXCLUDED.status,
			updated_at = NOW()`
	_, err := m.pool.Exec(ctx, query, id, name, address, longitude, latitude, status)
	if err != nil {
		return fmt.Errorf("PostGISYardModel.Upsert 失败: %w", err)
	}
	return nil
}

// Delete 根据 ID 物理删除堆场空间数据。
func (m *PostGISYardModel) Delete(ctx context.Context, id int64) error {
	query := `DELETE FROM yard WHERE id = $1`
	_, err := m.pool.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("PostGISYardModel.Delete 失败: %w", err)
	}
	return nil
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

