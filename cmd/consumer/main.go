// main 是后台 Redis 消息队列消费者脚本的入口程序。
// 用于从 Redis 阻塞队列消费堆场和集装箱的同步消息，并将地理空间数据同步写入 PostgreSQL (PostGIS)。
//
// 启动方式：
//
//	# 默认开发环境
//	go run cmd/consumer/main.go -f etc/mapserver-dev.yaml
//
//	# 测试环境
//	go run cmd/consumer/main.go -f etc/mapserver-test.yaml
//
//	# 生产环境
//	go run cmd/consumer/main.go -f etc/mapserver-prod.yaml
package main

import (
	"context"
	"encoding/json"
	"flag"
	"os/signal"
	"syscall"
	"time"

	"map-server/internal/config"
	"map-server/internal/model"
	"map-server/internal/svc"

	"github.com/zeromicro/go-zero/core/conf"
	"github.com/zeromicro/go-zero/core/logx"
)

// configFile 指定配置文件路径，默认为 etc/mapserver-dev.yaml，可由 -f 参数覆盖
var configFile = flag.String("f", "etc/mapserver-dev.yaml", "配置文件路径")

// 队列 Key 定义
const queueKey = "queue:spatial_sync"

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

func main() {
	// 解析命令行参数
	flag.Parse()

	// 加载服务配置
	var c config.Config
	conf.MustLoad(*configFile, &c)

	// 初始化 logx 日志系统，使消费者的日志也能按环境配置自动落地、轮转和压缩
	logx.MustSetup(c.Log)

	// 初始化服务上下文（包含 Redis 和 PostgreSQL 连接池）
	svcCtx := svc.NewServiceContext(c)

	// 设置平滑退出的 Context，接收到操作系统终止信号时自动取消
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	logx.Infof("[Consumer] 消费者服务启动成功，正在监听 Redis 队列: %s...", queueKey)

	// 消费循环
	for {
		select {
		case <-ctx.Done():
			logx.Info("[Consumer] 收到退出信号，正在安全终止消费者服务...")
			return
		default:
			// 从 Redis List 队列中阻塞读取消息
			// BLPop 第二个参数设为 0 表示无限期阻塞，直到有消息或 Context 被取消
			res, err := svcCtx.RedisClient.BLPop(ctx, 0, queueKey).Result()
			if err != nil {
				// 若 Context 已经取消，则直接退出
				if ctx.Err() != nil {
					logx.Info("[Consumer] 消费者服务已平滑退出。")
					return
				}
				// 打印错误日志，休眠 2 秒后继续重试（避免 Redis 连接异常时出现 CPU 满载）
				logx.Errorf("[Consumer] 从 Redis 队列 BLPop 失败: %v，将在 2 秒后重试", err)
				time.Sleep(2 * time.Second)
				continue
			}

			// BLPop 成功时返回的结构为 []string{key, value}
			if len(res) < 2 {
				continue
			}
			msgValue := res[1]

			// 在独立的协程或函数中处理消息，提高主循环的响应速度与健壮性
			handleMessage(ctx, svcCtx, msgValue)
		}
	}
}

// handleMessage 解析单条队列消息并执行相应的 PostGIS 同步操作
func handleMessage(ctx context.Context, svcCtx *svc.ServiceContext, msgValue string) {
	logx.Infof("[Consumer] 接收到消息: %s", msgValue)

	// 1. 解析外层通用消息结构
	var msg SyncMessage
	if err := json.Unmarshal([]byte(msgValue), &msg); err != nil {
		logx.Errorf("[Consumer] 消息解析 JSON 失败: %v, 丢弃非法消息", err)
		return
	}

	// 验证消息合法性
	if msg.Type != "yard" && msg.Type != "container" {
		logx.Errorf("[Consumer] 消息类型错误: %s, 丢弃消息", msg.Type)
		return
	}
	if msg.Action != "upsert" && msg.Action != "delete" {
		logx.Errorf("[Consumer] 消息动作错误: %s, 丢弃消息", msg.Action)
		return
	}

	const maxRetries = 3
	var dbErr error

	// 2. 执行数据库写入，支持最多 3 次重试机制
	for retry := 1; retry <= maxRetries; retry++ {
		if retry > 1 {
			logx.Errorf("[Consumer] 写入 PostgreSQL 失败，正在进行第 %d/%d 次重试...", retry, maxRetries)
			time.Sleep(200 * time.Millisecond) // 重试退避延时
		}

		switch msg.Type {
		case "yard":
			var yp YardPayload
			if err := json.Unmarshal(msg.Payload, &yp); err != nil {
				logx.Errorf("[Consumer] 堆场载荷解析 JSON 失败: %v, 丢弃消息", err)
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
				logx.Errorf("[Consumer] 集装箱载荷解析 JSON 失败: %v, 丢弃消息", err)
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

	// 3. 统计最终同步状态
	if dbErr != nil {
		logx.Errorf("[Consumer] [同步失败] 同步 PostGIS 失败（已达最大重试次数）: type=%s, action=%s, 错误=%v", msg.Type, msg.Action, dbErr)
		// 记录失败日志到 MySQL
		errLog := &model.SyncFailureLog{
			DataType: msg.Type,
			Action:   msg.Action,
			Payload:  string(msg.Payload),
			ErrorMsg: dbErr.Error(),
			Status:   1, // 1=未处理
		}
		if insertErr := svcCtx.SyncFailureLogModel.Insert(ctx, errLog); insertErr != nil {
			logx.Errorf("[Consumer] [记录日志失败] 无法将同步失败日志写入数据库: %v", insertErr)
		}
	} else {
		logx.Infof("[Consumer] [同步成功] 成功同步 PostGIS 数据库: type=%s, action=%s", msg.Type, msg.Action)
	}
}
