package logic_test

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/zeromicro/go-zero/core/conf"
	"map-server/internal/config"
	"map-server/internal/logic"
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
		Direction: "supply",
		PageSize:  1,
	}

	resp, err := l.GetTradingsList(req)
	if err != nil {
		t.Fatalf("调用 GetTradingsList 报错: %v", err)
	}

	resBytes, _ := json.MarshalIndent(resp, "", "  ")
	fmt.Printf("--- 测试成功，返回数据样例 ---\n%s\n", string(resBytes))
}
