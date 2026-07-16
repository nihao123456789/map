// Package main 包含 Redis 队列消费者的具体业务处理逻辑。
// 本文件专门负责 queue:spatial_sync 队列的消息结构解析与 PostGIS 空间数据库的数据同步。
package main

import (
	"context"
	"encoding/json"
	"time"

	mysqlModel "map-server/internal/model/mysql/map_server"
	"map-server/internal/svc"

	"github.com/zeromicro/go-zero/core/logx"
)

// redisQueueKey 定义空间同步的 Redis 队列 Key
const redisQueueKey = "queue:spatial_sync"

// SyncMessage 队列消息的统一外层结构
type SyncMessage struct {
	Type    string          `json:"type"`    // "yard" (堆场) 或 "container" (集装箱)
	Action  string          `json:"action"`  // "upsert" (插入/更新) 或 "delete" (删除)
	Payload json.RawMessage `json:"payload"` // 业务实体的具体 JSON 数据
}

// YardPayload 堆场同步消息载荷
type YardPayload struct {
	ID        int64   `json:"id"`
	Name      string  `json:"name"`
	Address   string  `json:"address"`
	Longitude float64 `json:"longitude"`
	Latitude  float64 `json:"latitude"`
	Status    int16   `json:"status"`
}

// ContainerPayload 集装箱同步消息载荷
type ContainerPayload struct {
	ID        int64   `json:"id"`
	Number    string  `json:"number"`
	YardID    int64   `json:"yard_id"`
	Longitude float64 `json:"longitude"`
	Latitude  float64 `json:"latitude"`
	Status    int16   `json:"status"`
}

// handleSpatialSyncMessage 解析单条 Redis 队列消息并执行相应的 PostGIS 同步操作
//
// 参数：
//   - ctx：上下文
//   - svcCtx：服务上下文
//   - msgValue：原始 JSON 格式的消息字符串
func handleSpatialSyncMessage(ctx context.Context, svcCtx *svc.ServiceContext, msgValue string) {
	logx.Infof("[Redis-Consumer] 接收到消息: %s", msgValue)

	// 1. 解析外层通用消息结构
	var msg SyncMessage
	if err := json.Unmarshal([]byte(msgValue), &msg); err != nil {
		logx.Errorf("[Redis-Consumer] 消息解析 JSON 失败: %v, 丢弃非法消息", err)
		return
	}

	// 验证消息合法性
	if msg.Type != "yard" && msg.Type != "container" {
		logx.Errorf("[Redis-Consumer] 消息类型错误: %s, 丢弃消息", msg.Type)
		return
	}
	if msg.Action != "upsert" && msg.Action != "delete" {
		logx.Errorf("[Redis-Consumer] 消息动作错误: %s, 丢弃消息", msg.Action)
		return
	}

	const maxRetries = 3
	var dbErr error

	// 2. 执行数据库同步写入，支持最多 3 次重试机制
	for retry := 1; retry <= maxRetries; retry++ {
		if retry > 1 {
			logx.Errorf("[Redis-Consumer] 写入 PostgreSQL 失败，正在进行第 %d/%d 次重试...", retry, maxRetries)
			time.Sleep(200 * time.Millisecond) // 重试退避延时
		}

		switch msg.Type {
		case "yard":
			var yp YardPayload
			if err := json.Unmarshal(msg.Payload, &yp); err != nil {
				logx.Errorf("[Redis-Consumer] 堆场载荷解析 JSON 失败: %v, 丢弃消息", err)
				return
			}

			if msg.Action == "upsert" {
				dbErr = svcCtx.PostGISYardModel.Upsert(ctx, yp.ID, yp.Name, yp.Address, yp.Longitude, yp.Latitude, yp.Status)
			} else {
				dbErr = svcCtx.PostGISYardModel.Delete(ctx, yp.ID)
			}

		case "container":
			var cp ContainerPayload
			if err := json.Unmarshal(msg.Payload, &cp); err != nil {
				logx.Errorf("[Redis-Consumer] 集装箱载荷解析 JSON 失败: %v, 丢弃消息", err)
				return
			}

			if msg.Action == "upsert" {
				dbErr = svcCtx.PostGISContainerModel.Upsert(ctx, cp.ID, cp.Number, cp.YardID, cp.Longitude, cp.Latitude, cp.Status)
			} else {
				dbErr = svcCtx.PostGISContainerModel.Delete(ctx, cp.ID)
			}
		}

		// 写入成功，中断重试循环
		if dbErr == nil {
			break
		}
	}

	// 3. 统计最终同步状态并持久化失败日志
	if dbErr != nil {
		logx.Errorf("[Redis-Consumer] [同步失败] 同步 PostGIS 失败（已达最大重试次数）: type=%s, action=%s, 错误=%v", msg.Type, msg.Action, dbErr)
		// 记录失败日志到 MySQL
		errLog := &mysqlModel.SyncFailureLog{
			DataType: msg.Type,
			Action:   msg.Action,
			Payload:  string(msg.Payload),
			ErrorMsg: dbErr.Error(),
			Status:   1, // 1=未处理
		}
		if insertErr := svcCtx.SyncFailureLogModel.Insert(ctx, errLog); insertErr != nil {
			logx.Errorf("[Redis-Consumer] [记录日志失败] 无法将同步失败日志写入数据库: %v", insertErr)
		}
	} else {
		logx.Infof("[Redis-Consumer] [同步成功] 成功同步 PostGIS 数据库: type=%s, action=%s", msg.Type, msg.Action)
	}
}
