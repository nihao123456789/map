// Package model 定义数据库实体模型和数据访问方法。
// 使用 go-zero 内置的 sqlx 包操作 MySQL 数据库。
package model

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

// Yard 对应数据库中的 yard 堆场表。
type Yard struct {
	// Id 堆场主键ID
	Id int64 `db:"id"`
	// Name 堆场名称
	Name string `db:"name"`
	// Longitude 经度
	Longitude float64 `db:"longitude"`
	// Latitude 纬度
	Latitude float64 `db:"latitude"`
	// Address 详细地址
	Address string `db:"address"`
	// Status 状态：1=启用，0=禁用
	Status int8 `db:"status"`
}

// Container 对应数据库中的 container 集装箱表。
type Container struct {
	// Id 集装箱主键ID
	Id int64 `db:"id"`
	// Number 集装箱编号
	Number string `db:"number"`
	// YardId 所属堆场ID
	YardId int64 `db:"yard_id"`
	// Longitude 经度
	Longitude float64 `db:"longitude"`
	// Latitude 纬度
	Latitude float64 `db:"latitude"`
	// Status 状态：1=在场，0=离场
	Status int8 `db:"status"`
}

// YardModel 提供对 yard 表的数据访问操作。
type YardModel struct {
	// conn 是 go-zero 封装的数据库连接
	conn sqlx.SqlConn
}

// ContainerModel 提供对 container 表的数据访问操作。
type ContainerModel struct {
	// conn 是 go-zero 封装的数据库连接
	conn sqlx.SqlConn
}

// NewYardModel 创建并返回 YardModel 实例。
//
// 参数：
//   - conn：数据库连接
func NewYardModel(conn sqlx.SqlConn) *YardModel {
	return &YardModel{conn: conn}
}

// NewContainerModel 创建并返回 ContainerModel 实例。
//
// 参数：
//   - conn：数据库连接
func NewContainerModel(conn sqlx.SqlConn) *ContainerModel {
	return &ContainerModel{conn: conn}
}

// FindByIds 根据 ID 列表批量查询堆场信息。
//
// 参数：
//   - ctx：上下文
//   - ids：堆场 ID 列表
//
// 返回：堆场列表（按 ID 顺序不保证），以及错误信息。
func (m *YardModel) FindByIds(ctx context.Context, ids []int64) ([]*Yard, error) {
	if len(ids) == 0 {
		return []*Yard{}, nil
	}

	// 构造 IN 查询的占位符，例如 "?,?,?"
	placeholders := make([]string, len(ids))
	args := make([]interface{}, len(ids))
	for i, id := range ids {
		placeholders[i] = "?"
		args[i] = id
	}

	query := fmt.Sprintf(
		"SELECT `id`,`name`,`longitude`,`latitude`,`address`,`status` FROM `yard` WHERE `id` IN (%s) AND `status` = 1",
		strings.Join(placeholders, ","),
	)

	var yards []*Yard
	err := m.conn.QueryRowsCtx(ctx, &yards, query, args...)
	if err != nil {
		// 若查询结果为空，sqlx 会返回 sql.ErrNoRows，此处视为正常情况
		if err == sql.ErrNoRows {
			return []*Yard{}, nil
		}
		return nil, fmt.Errorf("YardModel.FindByIds 查询失败: %w", err)
	}
	return yards, nil
}

// FindAllActive 查询所有启用状态的堆场（用于初始化 Redis GEO 数据）。
//
// 参数：
//   - ctx：上下文
//
// 返回：所有启用状态的堆场列表，以及错误信息。
func (m *YardModel) FindAllActive(ctx context.Context) ([]*Yard, error) {
	query := "SELECT `id`,`name`,`longitude`,`latitude`,`address`,`status` FROM `yard` WHERE `status` = 1"
	var yards []*Yard
	err := m.conn.QueryRowsCtx(ctx, &yards, query)
	if err != nil {
		if err == sql.ErrNoRows {
			return []*Yard{}, nil
		}
		return nil, fmt.Errorf("YardModel.FindAllActive 查询失败: %w", err)
	}
	return yards, nil
}

// FindByIds 根据 ID 列表批量查询集装箱信息。
//
// 参数：
//   - ctx：上下文
//   - ids：集装箱 ID 列表
//
// 返回：集装箱列表，以及错误信息。
func (m *ContainerModel) FindByIds(ctx context.Context, ids []int64) ([]*Container, error) {
	if len(ids) == 0 {
		return []*Container{}, nil
	}

	// 构造 IN 查询的占位符
	placeholders := make([]string, len(ids))
	args := make([]interface{}, len(ids))
	for i, id := range ids {
		placeholders[i] = "?"
		args[i] = id
	}

	query := fmt.Sprintf(
		"SELECT `id`,`number`,`yard_id`,`longitude`,`latitude`,`status` FROM `container` WHERE `id` IN (%s) AND `status` = 1",
		strings.Join(placeholders, ","),
	)

	var containers []*Container
	err := m.conn.QueryRowsCtx(ctx, &containers, query, args...)
	if err != nil {
		if err == sql.ErrNoRows {
			return []*Container{}, nil
		}
		return nil, fmt.Errorf("ContainerModel.FindByIds 查询失败: %w", err)
	}
	return containers, nil
}

// FindAllActive 查询所有在场状态的集装箱（用于初始化 Redis GEO 数据）。
//
// 参数：
//   - ctx：上下文
//
// 返回：所有在场状态的集装箱列表，以及错误信息。
func (m *ContainerModel) FindAllActive(ctx context.Context) ([]*Container, error) {
	query := "SELECT `id`,`number`,`yard_id`,`longitude`,`latitude`,`status` FROM `container` WHERE `status` = 1"
	var containers []*Container
	err := m.conn.QueryRowsCtx(ctx, &containers, query)
	if err != nil {
		if err == sql.ErrNoRows {
			return []*Container{}, nil
		}
		return nil, fmt.Errorf("ContainerModel.FindAllActive 查询失败: %w", err)
	}
	return containers, nil
}

// SyncFailureLog 对应数据库中的 sync_failure_log 表，用于记录 PostGIS 同步失败的信息。
type SyncFailureLog struct {
	// Id 失败记录主键ID
	Id int64 `db:"id"`
	// DataType 数据类型：yard=堆场，container=集装箱
	DataType string `db:"data_type"`
	// Action 操作动作：upsert=新增或更新，delete=删除
	Action string `db:"action"`
	// Payload 同步时的数据载荷(JSON)
	Payload string `db:"payload"`
	// ErrorMsg 具体的错误信息
	ErrorMsg string `db:"error_msg"`
	// Status 处理状态：1=未处理，2=已处理
	Status int8 `db:"status"`
	// CreatedAt 同步失败记录时间
	CreatedAt time.Time `db:"created_at"`
}

// SyncFailureLogModel 提供对 sync_failure_log 表的数据访问操作。
type SyncFailureLogModel struct {
	// conn 是 go-zero 封装的数据库连接
	conn sqlx.SqlConn
}

// NewSyncFailureLogModel 创建并返回 SyncFailureLogModel 实例。
//
// 参数：
//   - conn：数据库连接
func NewSyncFailureLogModel(conn sqlx.SqlConn) *SyncFailureLogModel {
	return &SyncFailureLogModel{conn: conn}
}

// Insert 插入一条同步失败记录。
//
// 参数：
//   - ctx：上下文
//   - log：同步失败日志结构体
//
// 返回：错误信息。
func (m *SyncFailureLogModel) Insert(ctx context.Context, log *SyncFailureLog) error {
	query := "INSERT INTO `sync_failure_log` (`data_type`, `action`, `payload`, `error_msg`, `status`) VALUES (?, ?, ?, ?, ?)"
	_, err := m.conn.ExecCtx(ctx, query, log.DataType, log.Action, log.Payload, log.ErrorMsg, log.Status)
	if err != nil {
		return fmt.Errorf("SyncFailureLogModel.Insert 插入失败: %w", err)
	}
	return nil
}
