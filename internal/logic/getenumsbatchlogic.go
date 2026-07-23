package logic

import (
	"context"

	"map-server/internal/svc"
	"map-server/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

// GetEnumsBatchLogic 提供批量拉取字典项的业务逻辑结构体。
type GetEnumsBatchLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// NewGetEnumsBatchLogic 初始化业务逻辑对象。
func NewGetEnumsBatchLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetEnumsBatchLogic {
	return &GetEnumsBatchLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

// GetEnumsBatch 实现批量拉取字典项的业务逻辑，以 category 进行分组返回。
func (l *GetEnumsBatchLogic) GetEnumsBatch(req *types.EnumsBatchReq) (resp *types.EnumsBatchResp, err error) {

	enumsMap := make(map[string][]types.EnumInfo)
	for _, cat := range req.Categories {
		// Take 接口首先从本地 5 分钟 TTL 缓存加载。若未命中，利用 singleflight 保护去数据库加载并回填
		data, err := l.svcCtx.EnumsCache.Take("enums:"+cat, func() (interface{}, error) {
			list, err := l.svcCtx.EnumsModel.FindByCategories(l.ctx, []string{cat})
			if err != nil {
				return nil, err
			}
			res := make([]types.EnumInfo, 0, len(list))
			for _, item := range list {
				res = append(res, types.EnumInfo{
					Category:     item.Category,
					CategoryName: item.CategoryName,
					ItemId:       item.ItemId,
					Value:        item.Value,
					Name:         item.Name,
					NameZh:       item.NameZh,
					Extra:        item.Extra,
				})
			}
			return res, nil
		})
		if err != nil {
			l.Errorf("获取字典 [%s] 失败: %v", cat, err)
			return nil, err
		}
		if cachedList, ok := data.([]types.EnumInfo); ok {
			enumsMap[cat] = cachedList
		} else {
			enumsMap[cat] = make([]types.EnumInfo, 0)
		}
	}

	return &types.EnumsBatchResp{
		Enums: enumsMap,
	}, nil
}
