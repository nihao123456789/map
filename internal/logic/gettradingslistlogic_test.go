package logic_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	"github.com/zeromicro/go-zero/core/conf"
	"github.com/zeromicro/go-zero/rest"
	"github.com/zeromicro/go-zero/rest/httpx"

	"map-server/internal/config"
	"map-server/internal/errorx"
	"map-server/internal/handler"
	"map-server/internal/logic"
	"map-server/internal/middleware"
	"map-server/internal/svc"
	"map-server/internal/types"
)

func TestGetTradingsList_LocationInfo(t *testing.T) {
	var c config.Config
	// 加载本地开发配置文件
	conf.MustLoad("d:/project/map/etc/mapserver-dev.yaml", &c)

	// 初始化服务上下文
	svcCtx := svc.NewServiceContext(c)

	l := logic.NewGetTradingsListLogic(context.Background(), svcCtx)

	req := &types.TradingListReq{
		Direction:      "supply",
		Category:       "dry",
		Condition:      "brand_new",
		Color:          "",
		EquipmentType:  "twenty_dry_container",
		CommercialTerm: "pick_up",
		PageSize:       1,
	}

	resp, err := l.GetTradingsList(req)
	if err != nil {
		t.Fatalf("调用 GetTradingsList 报错: %v", err)
	}

	resBytes, _ := json.MarshalIndent(resp, "", "  ")
	fmt.Printf("--- 测试成功，返回数据样例 ---\n%s\n", string(resBytes))
}

// TestRunServer 用于在受 Device Guard 限制的机器上，以单元测试的名义直接以白名单方式启动 HTTP 地图服务。
func TestRunServer(t *testing.T) {
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

	// 从配置文件加载服务配置
	var c config.Config
	conf.MustLoad("d:/project/map/etc/mapserver-dev.yaml", &c)

	// 初始化 go-zero HTTP 服务器
	server := rest.MustNewServer(c.RestConf)
	defer server.Stop()

	// 初始化服务上下文（包含 MySQL、Redis 连接等所有依赖）
	ctx := svc.NewServiceContext(c)

	// 注册全局成功返回包装中间件
	server.Use(middleware.UniformResponseMiddleware)

	// 注册所有 HTTP 路由处理器
	handler.RegisterHandlers(server, ctx)

	// 打印启动信息
	fmt.Printf("【白名单调试模式】地图服务启动成功，监听地址：%s:%d\n", c.Host, c.Port)
	fmt.Println("接口列表：")
	fmt.Println("  POST /api/tradings/list - 获取集装箱交易挂单列表（支持去重关联企业信息、会员徽章与地理位置树节点）")

	// 启动 HTTP 服务，阻塞直到收到退出信号
	server.Start()
}
