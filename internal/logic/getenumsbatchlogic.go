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
	l.Infof("批量获取字典数据请求: categories=%v", req.Categories)

	if len(req.Categories) == 0 {
		return &types.EnumsBatchResp{
			Enums: make(map[string][]types.EnumItem),
		}, nil
	}

	// 批量拉取数据，规避多次循环查库的 N+1 慢查询问题
	enumsData, err := l.svcCtx.EnumsModel.FindByCategories(l.ctx, req.Categories)
	if err != nil {
		l.Errorf("批量获取字典数据失败: %v", err)
		return nil, err
	}

	enumsMap := make(map[string][]types.EnumItem)
	// 初始化所有请求中传入的分类 key，即使某个分类下无任何数据，也向客户端返回空切片而非 null 或未定义，提高前端健壮性
	for _, cat := range req.Categories {
		enumsMap[cat] = make([]types.EnumItem, 0)
	}

	for _, item := range enumsData {
		enumsMap[item.Category] = append(enumsMap[item.Category], types.EnumItem{
			Id:            item.Id,
			Category:      item.Category,
			CategoryName:  item.CategoryName,
			ItemId:        item.ItemId,
			Value:         item.Value,
			Name:          item.Name,
			NameZh:        item.NameZh,
			Description:   item.Description,
			DescriptionZh: item.DescriptionZh,
			Extra:         item.Extra,
		})
	}

	return &types.EnumsBatchResp{
		Enums: enumsMap,
	}, nil
}
