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

	// 批量拉取地理位置树节点详情以去重查询，规避 N+1 慢查询瓶颈
	var locationIDs []int64
	for _, item := range counts {
		if item.LocationId > 0 {
			locationIDs = append(locationIDs, item.LocationId)
		}
	}

	nodesMap := make(map[int64]*types.LocationInfo)
	if len(locationIDs) > 0 {
		nodesData, err := l.svcCtx.TreeNodesModel.FindByIds(l.ctx, locationIDs)
		if err != nil {
			l.Errorf("批量获取统计位置详情失败: %v", err)
			return nil, err
		}
		for _, node := range nodesData {
			nodesMap[node.Id] = &types.LocationInfo{
				Id:          node.Id,
				Name:        node.Name.String,
				Type:        node.Type.String,
				EnglishName: node.EnglishName.String,
				Lat:         float32(node.Lat.Float64),
				Lng:         float32(node.Lng.Float64),
				FullName:    node.FullName.String,
				FullNameCn:  node.FullNameCn.String,
			}
		}
	}

	list := make([]types.TradingLocationCountItem, 0, len(counts))
	for _, item := range counts {
		list = append(list, types.TradingLocationCountItem{
			LocationId:   item.LocationId,
			Count:        item.Count,
			LocationInfo: nodesMap[item.LocationId],
		})
	}

	return &types.TradingLocationCountResp{
		List: list,
	}, nil
}
