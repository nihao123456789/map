// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.1

// main 是 map-server 地图服务的入口程序。
//
// 启动方式：
//
//	# 默认开发环境
//	$env:GOTMPDIR = "d:\project\map"; go run mapserver.go -f etc/mapserver-dev.yaml

//	# 测试环境
//	go run mapserver.go -f etc/mapserver-test.yaml
//
//	# 生产环境
//	go run mapserver.go -f etc/mapserver-prod.yaml
//
// 或编译后执行：
//
//	./mapserver -f etc/mapserver-dev.yaml
package main

import (
	"context"
	"flag"
	"fmt"
	"net/http"

	"map-server/internal/config"
	"map-server/internal/errorx"
	"map-server/internal/handler"
	"map-server/internal/middleware"
	"map-server/internal/svc"

	"github.com/zeromicro/go-zero/core/conf"
	"github.com/zeromicro/go-zero/rest"
	"github.com/zeromicro/go-zero/rest/httpx"
)

// configFile 指定配置文件路径，默认为 etc/mapserver-dev.yaml，
// 可通过命令行参数 -f 覆盖。
var configFile = flag.String("f", "etc/mapserver-dev.yaml", "配置文件路径")

func main() {
	// 解析命令行参数
	flag.Parse()

	// 注册全局错误处理器，将参数校验错误及 Logic 层错误统一输出为友好的 JSON 格式
	httpx.SetErrorHandlerCtx(func(ctx context.Context, err error) (int, interface{}) {
		switch e := err.(type) {
		case *errorx.CodeError:
			// 如果是自定义业务错误，则原样输出自定义的错误码和错误信息，HTTP 状态码返回 200 OK
			return http.StatusOK, map[string]interface{}{
				"code": e.Code,
				"msg":  e.Msg,
			}
		default:
			// 否则（如参数校验失败等普通错误），使用默认的 400 错误码
			return http.StatusBadRequest, map[string]interface{}{
				"code": 400,
				"msg":  err.Error(),
			}
		}
	})

	// 从配置文件加载服务配置（加载失败时直接 panic 终止）
	var c config.Config
	conf.MustLoad(*configFile, &c)

	// 初始化 go-zero HTTP 服务器（加载失败时直接 panic 终止）
	server := rest.MustNewServer(c.RestConf)
	defer server.Stop()

	// 初始化服务上下文（包含 MySQL、Redis 连接等所有依赖）
	ctx := svc.NewServiceContext(c)

	// 注册全局成功返回包装中间件
	server.Use(middleware.UniformResponseMiddleware)

	// 注册全局防暴流限流中间件 (基于单机令牌桶保护)
	server.Use(func(next http.HandlerFunc) http.HandlerFunc {
		return middleware.RateLimitMiddleware(ctx.RateLimiter, next)
	})

	// 注册所有 HTTP 路由处理器
	handler.RegisterHandlers(server, ctx)

	// 打印启动信息
	fmt.Printf("地图服务启动成功，监听地址：%s:%d\n", c.Host, c.Port)

	// 启动 HTTP 服务，阻塞直到收到退出信号
	server.Start()
}
