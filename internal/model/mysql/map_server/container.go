// Package map_server 提供对 MySQL map_server 数据库中 container 表的数据访问。
package map_server

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

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
	// Status 状态：1=在场，2=离场
	Status int8 `db:"status"`
}

// ContainerModel 提供对 container 表的数据访问操作。
type ContainerModel struct {
	// conn 是 go-zero 封装的数据库连接
	conn sqlx.SqlConn
}

// NewContainerModel 创建并返回 ContainerModel 实例。
//
// 参数：
//   - conn：数据库连接
func NewContainerModel(conn sqlx.SqlConn) *ContainerModel {
	return &ContainerModel{conn: conn}
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
