package logic

import (
	"context"

	"map-server/internal/svc"
	"map-server/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

// GetTradingLocationCountLogic 提供按位置统计挂单数量的业务逻辑结构体。
type GetTradingLocationCountLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// NewGetTradingLocationCountLogic 初始化业务逻辑对象。
func NewGetTradingLocationCountLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetTradingLocationCountLogic {
	return &GetTradingLocationCountLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

// GetTradingLocationCount 按位置统计挂单数量的业务逻辑
func (l *GetTradingLocationCountLogic) GetTradingLocationCount(req *types.TradingLocationCountReq) (resp *types.TradingLocationCountResp, err error) {
	// 解析 direction 方向：supply -> 0, demand -> 1，非法值默认默认为 0 (supply)
	var dbDirection int64 = 0
	if req.Direction == "demand" {
		dbDirection = 1
	}

	counts, err := l.svcCtx.OffersModel.FindCountGroupByLocationId(l.ctx, dbDirection)
	if err != nil {
		l.Errorf("分组统计挂单位置数量失败: %v", err)
		return nil, err
	}

	list := make([]types.TradingLocationCountItem, 0, len(counts))
	for _, item := range counts {
		list = append(list, types.TradingLocationCountItem{
			LocationId: item.LocationId,
			Count:      item.Count,
		})
	}

	return &types.TradingLocationCountResp{
		List: list,
	}, nil
}
