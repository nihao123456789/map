// Package main 是后台 Redis 消息队列消费者服务的启动入口。
// 负责读取配置、初始化服务上下文、管理服务的平滑退出，并调度具体的队列消费逻辑。
//
// 启动方式：
//
//	# 默认开发环境
//	go run cmd/consumer/redis/*.go -f etc/mapserver-dev.yaml
//
//	# 测试环境
//	go run cmd/consumer/redis/*.go -f etc/mapserver-test.yaml
//
//	# 生产环境
//	go run cmd/consumer/redis/*.go -f etc/mapserver-prod.yaml
package main

import (
	"context"
	"flag"
	"os/signal"
	"syscall"
	"time"

	"map-server/internal/config"
	"map-server/internal/svc"

	"github.com/zeromicro/go-zero/core/conf"
	"github.com/zeromicro/go-zero/core/logx"
)

// configFile 指定配置文件路径，默认为 etc/mapserver-dev.yaml，可由 -f 参数覆盖
var configFile = flag.String("f", "etc/mapserver-dev.yaml", "配置文件路径")

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

	logx.Infof("[Redis-Consumer] 消费者服务启动成功，正在监听 Redis 队列: %s...", redisQueueKey)

	// 开启消费循环
	for {
		select {
		case <-ctx.Done():
			logx.Info("[Redis-Consumer] 收到退出信号，正在安全终止消费者服务...")
			return
		default:
			// 从 Redis List 队列中阻塞读取消息
			// BLPop 第二个参数设为 0 表示无限期阻塞，直到有消息或 Context 被取消
			res, err := svcCtx.RedisClient.BLPop(ctx, 0, redisQueueKey).Result()
			if err != nil {
				// 若 Context 已经取消，则直接退出
				if ctx.Err() != nil {
					logx.Info("[Redis-Consumer] 消费者服务已平滑退出。")
					return
				}
				// 打印错误日志，休眠 2 秒后继续重试（避免 Redis 连接异常时出现 CPU 满载）
				logx.Errorf("[Redis-Consumer] 从 Redis 队列 BLPop 失败: %v，将在 2 秒后重试", err)
				time.Sleep(2 * time.Second)
				continue
			}

			// BLPop 成功时返回的结构为 []string{key, value}
			if len(res) < 2 {
				continue
			}
			msgValue := res[1]

			// 在独立的协程或函数中处理消息，提高主循环的响应速度与健壮性
			handleSpatialSyncMessage(ctx, svcCtx, msgValue)
		}
	}
}
