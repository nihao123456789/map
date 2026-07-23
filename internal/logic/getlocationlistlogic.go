package logic

import (
	"context"

	"map-server/internal/svc"
	"map-server/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

// GetLocationListLogic 负责处理获取热门位置列表的业务逻辑。
type GetLocationListLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// NewGetLocationListLogic 初始化业务逻辑对象。
func NewGetLocationListLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetLocationListLogic {
	return &GetLocationListLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

// GetLocationList 拉取热门地理位置列表，按使用频率降序排列并过滤
func (l *GetLocationListLogic) GetLocationList() (resp *types.LocationListResp, err error) {
	list, err := l.svcCtx.TreeNodesModel.FindLocationsOrderByUsage(l.ctx)
	if err != nil {
		l.Errorf("拉取热门地理位置列表失败: %v", err)
		return nil, err
	}

	// 转换为 API 响应定义的 LocationInfo 列表，容量预分配降低 GC 压力
	locationList := make([]types.LocationInfo, 0, len(list))
	for _, item := range list {
		locationList = append(locationList, types.LocationInfo{
			Id:          item.Id,
			Name:        item.Name.String,
			Type:        item.Type.String,
			EnglishName: item.EnglishName.String,
			Lat:         float32(item.Lat.Float64),
			Lng:         float32(item.Lng.Float64),
			FullName:    item.FullName.String,
			FullNameCn:  item.FullNameCn.String,
		})
	}

	return &types.LocationListResp{
		List: locationList,
	}, nil
}
