package logic

import (
	"context"
	"database/sql"
	"strconv"

	"map-server/internal/svc"
	"map-server/internal/types"
	"map-server/internal/model/mysql/map_server/offers"
	"map-server/internal/model/mysql/map_server/companies"
	"map-server/internal/model/mysql/map_server/vipplans"
	"map-server/internal/model/mysql/map_server/membershippurchases"
	"map-server/internal/model/mysql/map_server/depots"
	"map-server/internal/model/mysql/map_server/treenodes"
	"map-server/internal/model/mysql/map_server/enums"

	"github.com/zeromicro/go-zero/core/logx"
	"golang.org/x/sync/errgroup"
)

// GetTradingsListLogic 负责处理获取交易挂单列表的业务逻辑。
type GetTradingsListLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// NewGetTradingsListLogic 创建并返回 GetTradingsListLogic 实例。
func NewGetTradingsListLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetTradingsListLogic {
	return &GetTradingsListLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

// GetTradingsList 获取集装箱交易挂单列表。
//
// 参数：
//   - req：请求参数（包含 location_id 和 direction）
//
// 返回：响应结果，以及错误信息。
func (l *GetTradingsListLogic) GetTradingsList(req *types.TradingListReq) (resp *types.TradingListResp, err error) {
	l.Infof("获取交易挂单列表请求: location_id=%d, direction=%s, last_id=%d, page_size=%d", req.LocationId, req.Direction, req.LastId, req.PageSize)
	// 根据 direction 映射数据库中的整数 direction 值
	var dbDirection int64
	switch req.Direction {
	case offers.ClientDirectionSupply:
		dbDirection = offers.DirectionBuy
	case offers.ClientDirectionDemand:
		dbDirection = offers.DirectionSell
	default:
		l.Errorf("无效的交易方向参数: %s", req.Direction)
		return &types.TradingListResp{List: []types.OfferInfo{}}, nil
	}

	// 解析 category 箱型分类：通过 EnumsModel 从 enums 数据表中根据 category = enums.CategoryContainerCategory 和 value 动态获取 item_id 的值
	var categoryItemIDStr string = "0"
	if len(req.Category) > 0 {
		enumCat, err := l.svcCtx.EnumsModel.FindOneByCategoryAndValue(l.ctx, enums.CategoryContainerCategory, req.Category)
		if err != nil {
			if err != enums.ErrNotFound {
				l.Errorf("从 enums 数据表获取箱型分类失败: value=%s, err=%v", req.Category, err)
				return nil, err
			}
		} else {
			categoryItemIDStr = enumCat.ItemId
		}
	}

	dbCategory, err := strconv.ParseInt(categoryItemIDStr, 10, 64)
	if err != nil {
		dbCategory = 0
	}

	// 解析 equipmentType 箱型规格：通过 EnumsModel 从 enums 数据表中根据 category = enums.CategoryEquipmentTypes 和 value 动态获取 item_id 的值
	var equipItemIDStr string = "0"
	if len(req.EquipmentType) > 0 {
		enumEquip, err := l.svcCtx.EnumsModel.FindOneByCategoryAndValue(l.ctx, enums.CategoryEquipmentTypes, req.EquipmentType)
		if err != nil {
			if err != enums.ErrNotFound {
				l.Errorf("从 enums 数据表获取箱型规格失败: value=%s, err=%v", req.EquipmentType, err)
				return nil, err
			}
		} else {
			equipItemIDStr = enumEquip.ItemId
		}
	}

	dbEquipmentType, err := strconv.ParseInt(equipItemIDStr, 10, 64)
	if err != nil {
		dbEquipmentType = 0
	}

	// 解析 condition 箱况：通过 EnumsModel 从 enums 数据表中根据 category = enums.CategoryConditions 和 value 动态获取 item_id 的值
	var itemIDStr string = "0"
	enumItem, err := l.svcCtx.EnumsModel.FindOneByCategoryAndValue(l.ctx, enums.CategoryConditions, req.Condition)
	if err != nil {
		if err != enums.ErrNotFound {
			l.Errorf("从 enums 数据表获取箱况失败: value=%s, err=%v", req.Condition, err)
			return nil, err
		}
	} else {
		itemIDStr = enumItem.ItemId
	}

	dbCondition, err := strconv.ParseInt(itemIDStr, 10, 64)
	if err != nil {
		dbCondition = 0
	}

	// 解析 commercialTerm 提箱方式：通过 EnumsModel 从 enums 数据表中根据 category = enums.CategoryCommercialTerm 和 value 动态获取 item_id 的值
	var termItemIDStr string = "0"
	if len(req.CommercialTerm) > 0 {
		enumTerm, err := l.svcCtx.EnumsModel.FindOneByCategoryAndValue(l.ctx, enums.CategoryCommercialTerm, req.CommercialTerm)
		if err != nil {
			if err != enums.ErrNotFound {
				l.Errorf("从 enums 数据表获取提箱方式失败: value=%s, err=%v", req.CommercialTerm, err)
				return nil, err
			}
		} else {
			termItemIDStr = enumTerm.ItemId
		}
	}

	dbCommercialTerm, err := strconv.ParseInt(termItemIDStr, 10, 64)
	if err != nil {
		dbCommercialTerm = 0
	}

	// 解析 color 颜色参数：直接使用传入的颜色标识（如 "RAL 1015"），供底层 colors 字段的模糊匹配使用
	dbColor := req.Color




	// 游标分页限制机制，防范海量数据查询导致内存溢出 (OOM) 与 GC 压力
	limit := req.PageSize
	if limit <= 0 {
		limit = 10 // 默认每页10条
	} else if limit > 100 {
		limit = 100 // 最大限制单页100条
	}

	// 统计满足条件的挂单总记录数
	totalCount, err := l.svcCtx.OffersModel.CountByLocationIdAndDirection(l.ctx, req.LocationId, dbDirection, dbCategory, dbCondition, dbColor, dbEquipmentType, dbCommercialTerm, int64(req.YearOfManufactureRangeFrom))
	if err != nil {
		l.Errorf("统计挂单总数失败: %v", err)
		return nil, err
	}

	// 从 MySQL 中查询挂单列表（支持游标分页）
	offersData, err := l.svcCtx.OffersModel.FindByLocationIdAndDirection(l.ctx, req.LocationId, dbDirection, dbCategory, dbCondition, dbColor, dbEquipmentType, dbCommercialTerm, int64(req.YearOfManufactureRangeFrom), req.LastId, limit)
	if err != nil {
		l.Errorf("查询挂单列表失败: %v", err)
		return nil, err
	}

	// 收集并去重有效的关联 ID，防范 N+1 次查询带来的数据库严重压力与 GC 消耗
	companyIdsMap := make(map[int64]struct{})
	depotIdsMap := make(map[int64]struct{})
	locationIdsMap := make(map[int64]struct{})
	conditionIdsMap := make(map[string]struct{})
	equipmentTypeIdsMap := make(map[string]struct{})
	commercialTermIdsMap := make(map[string]struct{})
	categoryIdsMap := make(map[string]struct{})

	for _, item := range offersData {
		if item.CompanyId.Valid && item.CompanyId.Int64 > 0 {
			companyIdsMap[item.CompanyId.Int64] = struct{}{}
		}
		if item.DepotId.Valid && item.DepotId.Int64 > 0 {
			depotIdsMap[item.DepotId.Int64] = struct{}{}
		}
		if item.LocationId.Valid && item.LocationId.Int64 > 0 {
			locationIdsMap[item.LocationId.Int64] = struct{}{}
		}
		if item.Condition.Valid && item.Condition.Int64 > 0 {
			conditionIdsMap[strconv.FormatInt(item.Condition.Int64, 10)] = struct{}{}
		}
		if item.EquipmentType > 0 {
			equipmentTypeIdsMap[strconv.FormatInt(item.EquipmentType, 10)] = struct{}{}
		}
		if item.CommercialTerm > 0 {
			commercialTermIdsMap[strconv.FormatInt(item.CommercialTerm, 10)] = struct{}{}
		}
		if item.Category.Valid && item.Category.Int64 > 0 {
			categoryIdsMap[strconv.FormatInt(item.Category.Int64, 10)] = struct{}{}
		}
	}

	// 使用 errgroup 开启并发查询任务，绑定请求 Context 控制生命周期，规避协程泄露与多表串行慢查询
	g, gCtx := errgroup.WithContext(l.ctx)

	var (
		companiesMap       = make(map[int64]*companies.Companies)
		purchasesMap       = make(map[int64][]*membershippurchases.MembershipPurchases)
		vipPlansMap        = make(map[int64]*vipplans.VipPlans)
		depotsMap          = make(map[int64]*depots.Depots)
		locationsMap       = make(map[int64]*treenodes.TreeNodes)
		conditionsMap      = make(map[string]*types.EnumInfo)
		equipmentTypesMap  = make(map[string]*types.EnumInfo)
		commercialTermsMap = make(map[string]*types.EnumInfo)
		categoriesMap      = make(map[string]*types.EnumInfo)
	)

	// 1. 并发拉取企业详细信息、激活会员记录及对应会员套餐配置
	if len(companyIdsMap) > 0 {
		companyIds := make([]int64, 0, len(companyIdsMap))
		for id := range companyIdsMap {
			companyIds = append(companyIds, id)
		}
		g.Go(func() error {
			// 1.1 批量拉取公司详细数据
			companiesData, err := l.svcCtx.CompaniesModel.FindByIds(gCtx, companyIds)
			if err != nil {
				l.Errorf("批量拉取企业详细信息失败: %v", err)
				return err
			}
			for _, comp := range companiesData {
				companiesMap[comp.Id] = comp
			}

			// 1.2 批量拉取激活状态的会员购买记录
			purchasesData, err := l.svcCtx.MembershipPurchasesModel.FindActiveByCompanyIds(gCtx, companyIds)
			if err != nil {
				l.Errorf("批量拉取企业会员购买记录失败: %v", err)
				return err
			}

			// 1.3 将 purchasesData 按 company_id 划分并收集涉及的 plan_ids
			vipPlanIdsMap := make(map[int64]struct{})
			for _, p := range purchasesData {
				purchasesMap[p.CompanyId] = append(purchasesMap[p.CompanyId], p)
				vipPlanIdsMap[p.VipPlanId] = struct{}{}
			}

			// 1.4 批量拉取所涉及的 vip_plans 套餐配置
			if len(vipPlanIdsMap) > 0 {
				vipPlanIds := make([]int64, 0, len(vipPlanIdsMap))
				for pid := range vipPlanIdsMap {
					vipPlanIds = append(vipPlanIds, pid)
				}
				plansData, err := l.svcCtx.VipPlansModel.FindByIds(gCtx, vipPlanIds)
				if err != nil {
					l.Errorf("批量拉取会员套餐信息失败: %v", err)
					return err
				}
				for _, plan := range plansData {
					vipPlansMap[plan.Id] = plan
				}
			}
			return nil
		})
	}

	// 2. 并发批量拉取堆场详情数据
	if len(depotIdsMap) > 0 {
		depotIds := make([]int64, 0, len(depotIdsMap))
		for id := range depotIdsMap {
			depotIds = append(depotIds, id)
		}
		g.Go(func() error {
			depotsData, err := l.svcCtx.DepotsModel.FindByIds(gCtx, depotIds)
			if err != nil {
				l.Errorf("批量拉取堆场详细信息失败: %v", err)
				return err
			}
			for _, dep := range depotsData {
				depotsMap[dep.Id] = dep
			}
			return nil
		})
	}

	// 3. 并发批量拉取地理位置树节点详情数据
	if len(locationIdsMap) > 0 {
		locationIds := make([]int64, 0, len(locationIdsMap))
		for id := range locationIdsMap {
			locationIds = append(locationIds, id)
		}
		g.Go(func() error {
			locationsData, err := l.svcCtx.TreeNodesModel.FindByIds(gCtx, locationIds)
			if err != nil {
				l.Errorf("批量拉取地理位置树节点信息失败: %v", err)
				return err
			}
			for _, node := range locationsData {
				locationsMap[node.Id] = node
			}
			return nil
		})
	}

	// 4. 并发批量拉取箱况字典项详情
	if len(conditionIdsMap) > 0 {
		conditionIds := make([]string, 0, len(conditionIdsMap))
		for id := range conditionIdsMap {
			conditionIds = append(conditionIds, id)
		}
		g.Go(func() error {
			enumsData, err := l.svcCtx.EnumsModel.FindByCategoryAndItemIds(gCtx, enums.CategoryConditions, conditionIds)
			if err != nil {
				l.Errorf("批量拉取箱况字典数据失败: %v", err)
				return err
			}
			for _, val := range enumsData {
				conditionsMap[val.ItemId] = toEnumInfo(val)
			}
			return nil
		})
	}

	// 5. 并发批量拉取箱型规格字典项详情
	if len(equipmentTypeIdsMap) > 0 {
		equipmentTypeIds := make([]string, 0, len(equipmentTypeIdsMap))
		for id := range equipmentTypeIdsMap {
			equipmentTypeIds = append(equipmentTypeIds, id)
		}
		g.Go(func() error {
			enumsData, err := l.svcCtx.EnumsModel.FindByCategoryAndItemIds(gCtx, enums.CategoryEquipmentTypes, equipmentTypeIds)
			if err != nil {
				l.Errorf("批量拉取箱型字典数据失败: %v", err)
				return err
			}
			for _, val := range enumsData {
				equipmentTypesMap[val.ItemId] = toEnumInfo(val)
			}
			return nil
		})
	}

	// 6. 并发批量拉取提箱方式字典项详情
	if len(commercialTermIdsMap) > 0 {
		commercialTermIds := make([]string, 0, len(commercialTermIdsMap))
		for id := range commercialTermIdsMap {
			commercialTermIds = append(commercialTermIds, id)
		}
		g.Go(func() error {
			enumsData, err := l.svcCtx.EnumsModel.FindByCategoryAndItemIds(gCtx, enums.CategoryCommercialTerm, commercialTermIds)
			if err != nil {
				l.Errorf("批量拉取贸易条款字典数据失败: %v", err)
				return err
			}
			for _, val := range enumsData {
				commercialTermsMap[val.ItemId] = toEnumInfo(val)
			}
			return nil
		})
	}

	// 7. 并发批量拉取箱型大类字典项详情
	if len(categoryIdsMap) > 0 {
		categoryIds := make([]string, 0, len(categoryIdsMap))
		for id := range categoryIdsMap {
			categoryIds = append(categoryIds, id)
		}
		g.Go(func() error {
			enumsData, err := l.svcCtx.EnumsModel.FindByCategoryAndItemIds(gCtx, enums.CategoryContainerCategory, categoryIds)
			if err != nil {
				l.Errorf("批量拉取箱型分类字典数据失败: %v", err)
				return err
			}
			for _, val := range enumsData {
				categoriesMap[val.ItemId] = toEnumInfo(val)
			}
			return nil
		})
	}

	// 等待所有并发协程完成，如果任何协程返回错误，则在这里被捕捉并返回
	if err := g.Wait(); err != nil {
		return nil, err
	}

	// 转换为 API 响应所定义的 OfferInfo 列表，采用容量预分配降低 GC 压力
	offersList := make([]types.OfferInfo, 0, len(offersData))
	for _, item := range offersData {
		info := toOfferInfo(item)
		if item.CompanyId.Valid && item.CompanyId.Int64 > 0 {
			if comp, exists := companiesMap[item.CompanyId.Int64]; exists {
				compPurchases := purchasesMap[item.CompanyId.Int64]
				info.CompanyInfo = toCompanyInfo(comp, compPurchases, vipPlansMap)
			}
		}
		if item.DepotId.Valid && item.DepotId.Int64 > 0 {
			if dep, exists := depotsMap[item.DepotId.Int64]; exists {
				info.DepotInfo = toDepotInfo(dep)
			}
		}
		if item.LocationId.Valid && item.LocationId.Int64 > 0 {
			if loc, exists := locationsMap[item.LocationId.Int64]; exists {
				info.LocationInfo = toLocationInfo(loc)
			}
		}
		if item.Condition.Valid && item.Condition.Int64 > 0 {
			idStr := strconv.FormatInt(item.Condition.Int64, 10)
			if cond, exists := conditionsMap[idStr]; exists {
				info.ConditionInfo = cond
			}
		}
		if item.EquipmentType > 0 {
			idStr := strconv.FormatInt(item.EquipmentType, 10)
			if eqType, exists := equipmentTypesMap[idStr]; exists {
				info.EquipmentTypeInfo = eqType
			}
		}
		if item.CommercialTerm > 0 {
			idStr := strconv.FormatInt(item.CommercialTerm, 10)
			if term, exists := commercialTermsMap[idStr]; exists {
				info.CommercialTermInfo = term
			}
		}
		if item.Category.Valid && item.Category.Int64 > 0 {
			idStr := strconv.FormatInt(item.Category.Int64, 10)
			if cat, exists := categoriesMap[idStr]; exists {
				info.CategoryInfo = cat
			}
		}
		offersList = append(offersList, info)
	}

	// 获取最后一条记录的 ID
	var lastId int64
	if len(offersData) > 0 {
		lastId = offersData[len(offersData)-1].Id
	}

	return &types.TradingListResp{
		Total:    totalCount,
		LastId:   lastId,
		PageSize: limit,
		List:     offersList,
	}, nil
}

// toOfferInfo 将数据库模型转换成 API 响应对应的 OfferInfo 结构体。
func toOfferInfo(item *offers.Offers) types.OfferInfo {
	return types.OfferInfo{
		Id:                             item.Id,
		Condition:                      int32(item.Condition.Int64),
		Type:                           item.Type,
		// PickupLocationId:               int32(item.PickupLocationId.Int64),
		// DropoffLocationId:              int32(item.DropoffLocationId.Int64),
		Quantity:                       int32(item.Quantity),
		// PickupCharge:                   float32(item.PickupCharge),
		// FreeDays:                       int32(item.FreeDays.Int64),
		// PerDiems:                       float32(item.PerDiems),
		// StorageFee:                     float32(item.StorageFee),
		// Dpp:                            float32(item.Dpp),
		// Premium:                        float32(item.Premium),
		// NewBuildPrice:                  float32(item.NewBuildPrice),
		// DepreciationPerYear:            float32(item.DepreciationPerYear),
		// MinimumReplacementValue:        float32(item.MinimumReplacementValue),
		// AdditionalInformation:          item.AdditionalInformation.String,
		UserId:                         int32(item.UserId),
		CompanyId:                      int32(item.CompanyId.Int64),
		Direction:                      int32(item.Direction.Int64),
		// ValidDays:                      int32(item.ValidDays),
		EquipmentType:                  int32(item.EquipmentType),
		CommercialTerm:                 int32(item.CommercialTerm),
		// Comments:                       item.Comments.String,
		// ReviewsCount:                   int32(item.ReviewsCount),
		// Prefixes:                       item.Prefixes.String,
		// YearOfManufacture:              int32(item.YearOfManufacture.Int64),
		// ManufacturerId:                 int32(item.ManufacturerId.Int64),
		// DamageProtectionPlan:           float32(item.DamageProtectionPlan),
		// NegotiationsCount:              int32(item.NegotiationsCount),
		Category:                       int32(item.Category.Int64),
		// ExpiresAt:                      formatTime(item.ExpiresAt),
		// ReviewId:                       int32(item.ReviewId.Int64),
		// Detail:                         item.Detail.String,
		Status:                         int32(item.Status),
		// DeletedAt:                      formatTime(item.DeletedAt),
		// ExpiresIn:                      int32(item.ExpiresIn),
		// ReadyForPickupOn:               formatTime(item.ReadyForPickupOn),
		// ReadyForPickupFrom:             formatTime(item.ReadyForPickupFrom),
		// ReadyForPickupTo:               formatTime(item.ReadyForPickupTo),
		// WithLockBox:                    item.WithLockBox != 0,
		// WithForkliftPockets:            item.WithForkliftPockets != 0,
		// SellFor:                        int32(item.SellFor.Int64),
		LocationId:                     int32(item.LocationId.Int64),
		// Name:                           item.Name.String,
		ImagesCount:                    int32(item.ImagesCount),
		// DocumentsCount:                 int32(item.DocumentsCount),
		// TradeType:                      int32(item.TradeType),
		// ExpectedDeliveryOn:             formatTime(item.ExpectedDeliveryOn),
		// ExpectedDeliveryFrom:           formatTime(item.ExpectedDeliveryFrom),
		// ExpectedDeliveryTo:             formatTime(item.ExpectedDeliveryTo),
		DepotId:                        int32(item.DepotId.Int64),
		UniqueNumber:                   item.UniqueNumber.String,
		CreatedAt:                      item.CreatedAt.Format("2006-01-02 15:04:05"),
		UpdatedAt:                      item.UpdatedAt.Format("2006-01-02 15:04:05"),
		Price:                          float32(item.Price.Float64),
		// Color:                          int32(item.Color.Int64),
		// EstimatedEmptyDeliveryDateFrom: formatTime(item.EstimatedEmptyDeliveryDateFrom),
		// OfferType:                      int32(item.OfferType.Int64),
		// Source:                         int32(item.Source),
		// PickupChargePayer:              int32(item.PickupChargePayer.Int64),
		// InsuranceFee:                   float32(item.InsuranceFee),
		YearOfManufactureRangeFrom:     int32(item.YearOfManufactureRangeFrom.Int64),
		YearOfManufactureRangeTo:       int32(item.YearOfManufactureRangeTo.Int64),
		// CscTestCertificate:             item.CscTestCertificate != 0,
		// EquipmentTypeId:                int32(item.EquipmentTypeId.Int64),
		// InsuranceType:                  int32(item.InsuranceType.Int64),
		// InsuranceDays:                  int32(item.InsuranceDays),
		// Extra:                          item.Extra.String,
		Colors:                         item.Colors.String,
		// PinnedAt:                       formatTime(item.PinnedAt),
		BumpedAt:                       formatTime(item.BumpedAt),
		// StorageFreeDays:                int32(item.StorageFreeDays),
		// ConditionTagIds:                item.ConditionTagIds.String,
		// ConditionLogo:                  int32(item.ConditionLogo.Int64),
		// NumberOfVents:                  int32(item.NumberOfVents),
		// DealCount:                      int32(item.DealCount),
		// CscExpiresOn:                   formatTime(item.CscExpiresOn),
		// EstimatedEmptyDeliveryDateTo:   formatTime(item.EstimatedEmptyDeliveryDateTo),
		// InstantSale:                    item.InstantSale != 0,
		// ConsignorId:                    int32(item.ConsignorId.Int64),
		// ConsignorName:                  item.ConsignorName.String,
		// Label:                          item.Label.String,
		// DropoffLocationIds:             item.DropoffLocationIds.String,
		// PickupLocationIds:              item.PickupLocationIds.String,
		// OriginalPrice:                  float32(item.OriginalPrice.Float64),
		// IsSpecialOffer:                 item.IsSpecialOffer.Int64 != 0,
		// SourceProposalId:               int32(item.SourceProposalId.Int64),
		// IsExpired:                      item.IsExpired != 0,
		// DataSource:                     int32(item.DataSource),
		// Meta:                           item.Meta.String,
		IsNonNegotiable:                item.IsNonNegotiable != 0,
		// HasDamages:                     item.HasDamages != 0,
		// WithEasyOpenDoor:               item.WithEasyOpenDoor.Int64 != 0,
	}
}

// formatTime 辅助格式化 NullTime。
func formatTime(nt sql.NullTime) string {
	if nt.Valid {
		return nt.Time.Format("2006-01-02 15:04:05")
	}
	return ""
}

// toCompanyInfo 将企业数据库模型映射转换为 API 返回的 types.CompanyInfo 结构体指针。
func toCompanyInfo(
	item *companies.Companies,
	purchases []*membershippurchases.MembershipPurchases,
	vipPlans map[int64]*vipplans.VipPlans,
) *types.CompanyInfo {
	if item == nil {
		return nil
	}

	// 内存中装配该企业的有效会员徽章列表，采用容量预分配
	badges := make([]types.MembershipBadge, 0, len(purchases))
	for _, p := range purchases {
		if plan, exists := vipPlans[p.VipPlanId]; exists {
			badges = append(badges, types.MembershipBadge{
				Kind:        plan.MembershipKind.String,
				Slug:        plan.Slug.String,
				Name:        plan.Name,
				NameEn:      plan.NameEn,
				// DisplayName: plan.Name,
				// ExpiresAt:   p.ExpiresAt.Format("2006-01-02T15:04:05+08:00"),
			})
		}
	}

	return &types.CompanyInfo{
		Id:               item.Id,
		Name:             item.Name.String,
		LocationId:       item.LocationId.Int64,
		// Telephone:        item.Telephone.String,
		// Email:            item.Email.String,
		// Usci:             item.Usci.String,
		Status:           item.Status,
		ReviewLevel:      float32(item.ReviewLevel),
		// ReviewsCount:     item.ReviewsCount,
		// IsOfficial:       item.IsOfficial == 1,
		// Address:          item.Address.String,
		MembershipBadges: badges,
	}
}

// toDepotInfo 将堆场数据库模型映射转换为 API 返回的 types.DepotInfo 结构体指针。
func toDepotInfo(item *depots.Depots) *types.DepotInfo {
	if item == nil {
		return nil
	}
	return &types.DepotInfo{
		Id:           item.Id,
		Name:         item.Name.String,
		// PostalCode:   item.PostalCode.String,
		// Website:      item.Website.String,
		// PhoneNumber:  item.PhoneNumber.String,
		City:         item.City.String,
		Country:      item.Country.String,
		// ContactName:  item.ContactName.String,
		// Email:        item.Email.String,
		LocationId:   item.LocationId.Int64,
		// LocalName:    item.LocalName.String,
		// LocalAddress: item.LocalAddress.String,
		AddressLine1: item.AddressLine1.String,
		AddressLine2: item.AddressLine2.String,
		Lat:          float32(item.Lat.Float64),
		Lng:          float32(item.Lng.Float64),
	}
}

// toLocationInfo 将数据库 treenodes 模型转换为 API 层的 LocationInfo 实体
func toLocationInfo(item *treenodes.TreeNodes) *types.LocationInfo {
	if item == nil {
		return nil
	}
	return &types.LocationInfo{
		Id:          item.Id,
		Name:        item.Name.String,
		Type:        item.Type.String,
		EnglishName: item.EnglishName.String,
		Lat:         float32(item.Lat.Float64),
		Lng:         float32(item.Lng.Float64),
		FullName:    item.FullName.String,
		FullNameCn:  item.FullNameCn.String,
	}
}

// toEnumInfo 将 enums 数据库模型转换为 API 层的 EnumInfo 实体
func toEnumInfo(item *enums.Enums) *types.EnumInfo {
	if item == nil {
		return nil
	}
	return &types.EnumInfo{
		// Id:            item.Id,
		Category:      item.Category,
		CategoryName:  item.CategoryName,
		ItemId:        item.ItemId,
		Value:         item.Value,
		Name:          item.Name,
		NameZh:        item.NameZh,
		// Description:   item.Description,
		// DescriptionZh: item.DescriptionZh,
		Extra:         item.Extra,
	}
}

