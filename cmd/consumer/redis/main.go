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
	"sync"
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

	logx.Infof("[Redis-Consumer] 消费者服务启动成功，正在监听 Redis 队列: %s...", spatialSyncQueueKey)

	// 声明 WaitGroup 用于追踪运行中的消息同步任务协程
	var wg sync.WaitGroup

	// 开启消费循环
	for {
		select {
		case <-ctx.Done():
			logx.Info("[Redis-Consumer] 收到退出信号，停止拉取新消息，正在等待进行中的任务完成...")
			wg.Wait()
			svcCtx.Shutdown()
			logx.Info("[Redis-Consumer] 消费者服务已安全平滑退出。")
			return
		default:
			// 从 Redis List 队列中阻塞读取消息
			// BLPop 第二个参数设为 0 表示无限期阻塞，直到有消息或 Context 被取消
			if svcCtx.RedisClient == nil {
				logx.Error("[Redis-Consumer] Redis 客户端未初始化，无法拉取队列消息，将在 5 秒后重试")
				time.Sleep(5 * time.Second)
				continue
			}
			res, err := svcCtx.RedisClient.BLPop(ctx, 0, spatialSyncQueueKey).Result()
			if err != nil {
				// 若 Context 已经取消，则直接等待进行中的任务并退出
				if ctx.Err() != nil {
					logx.Info("[Redis-Consumer] 检测到信号退出，停止拉取新消息，正在等待进行中的任务完成...")
					wg.Wait()
					svcCtx.Shutdown()
					logx.Info("[Redis-Consumer] 消费者服务已安全平滑退出。")
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

			// 使用协程并发处理消息，提高主循环的响应速度与高并发吞吐能力
			wg.Add(1)
			// 在外层提前创建好超时 context 和其 cancel 函数，并一同显式作为形参传入协程，减小隐式闭包捕获造成的变量逃逸
			taskCtx, taskCancel := context.WithTimeout(context.Background(), 10*time.Second)
			go func(ctx context.Context, cancel context.CancelFunc, sc *svc.ServiceContext, msg string) {
				defer wg.Done()
				defer cancel()
				handleSpatialSyncMessage(ctx, sc, msg)
			}(taskCtx, taskCancel, svcCtx, msgValue)
		}
	}
}
