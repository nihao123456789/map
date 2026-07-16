// Package map_server 提供对 MySQL map_server 数据库中 sync_failure_log 表的数据访问。
package map_server

import (
	"context"
	"fmt"
	"time"

	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

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
