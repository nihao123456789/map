package logic

import (
	"context"
	"database/sql"
	"encoding/json"
	"strconv"
	"time"

	"map-server/internal/svc"
	"map-server/internal/types"
	"map-server/internal/consts"
	"map-server/internal/errorx"
	"map-server/pkg/slices"
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

// fetchEnum 封装统一获取并反查数据字典与防穿透缓存逻辑的方法。
// 为了避免隐式闭包捕获外部复杂对象与上下文指针，该方法显式接收外部参数并返回对应的包装执行函数。
func (l *GetTradingsListLogic) fetchEnum(ctx context.Context, category, value, logName string, target **enums.Enums) func() error {
	return func() error {
		key := consts.GetEnumValCacheKey(category, value)
		val, err := l.svcCtx.EnumsCache.Take(key, func() (interface{}, error) {
			res, err := l.svcCtx.EnumsModel.FindOneByCategoryAndValue(ctx, category, value)
			if err != nil && err != enums.ErrNotFound {
				return nil, err
			}
			if err == enums.ErrNotFound {
				return &enums.Enums{ItemId: "0"}, nil // 哨兵值防御缓存穿透
			}
			return res, nil
		})
		if err != nil {
			l.Errorf("从 enums 数据表获取%s失败: value=%s, err=%v", logName, value, err)
			return err
		}
		if cached, ok := val.(*enums.Enums); ok && cached.ItemId != "0" {
			*target = cached
		}
		return nil
	}
}

// fetchEnumMap 封装统一从本地缓存批量加载指定分类数据字典项详情，并过滤填充到目标映射中的逻辑方法。
// 为了避免隐式闭包捕获外部复杂对象与上下文指针，该方法显式接收所有必需的上下文、过滤映射和目标映射。
func (l *GetTradingsListLogic) fetchEnumMap(ctx context.Context, category, logName string, filterMap map[string]struct{}, targetMap map[string]*types.EnumInfo) func() error {
	return func() error {
		data, err := l.svcCtx.EnumsCache.Take(consts.GetEnumsCacheKey(category), func() (interface{}, error) {
			list, err := l.svcCtx.EnumsModel.FindByCategories(ctx, []string{category})
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
			l.Errorf("并发获取%s字典数据失败: %v", logName, err)
			return err
		}
		if cachedList, ok := data.([]types.EnumInfo); ok {
			for _, val := range cachedList {
				if _, exists := filterMap[val.ItemId]; exists {
					valCopy := val
					targetMap[val.ItemId] = &valCopy
				}
			}
		}
		return nil
	}
}

// parseEnumItemID 辅助解析数据字典项中的 ItemId 字符串为 int64 整型数字。如果指针为空或解析失败，默认返回 0。
func parseEnumItemID(enum *enums.Enums) int64 {
	if enum == nil {
		return 0
	}
	val, err := strconv.ParseInt(enum.ItemId, 10, 64)
	if err != nil {
		return 0
	}
	return val
}

// GetTradingsList 获取集装箱交易挂单列表。
//
// 参数：
//   - req：请求参数（包含 LocationIds、Direction、Category、Condition、Color、EquipmentType、CommercialTerm、YearOfManufactureRangeFrom 以及分页参数 LastId 和 PageSize 等）
//
// 返回：响应结果，以及错误信息。
func (l *GetTradingsListLogic) GetTradingsList(req *types.TradingListReq) (resp *types.TradingListResp, err error) {
	// 对传入的位置 ID 列表进行去重处理，规避多余重复值占位符开销
	req.LocationIds = slices.Unique(req.LocationIds)

	// 校验过滤的位置 ID 数量是否超出系统最大限制，防止 SQL IN 解析及数据库引擎负载过高
	if len(req.LocationIds) > consts.MaxLocationIdsLimit {
		l.Errorf("位置过滤参数 location_ids 数量超限: 传入数量=%d, 上限=%d", len(req.LocationIds), consts.MaxLocationIdsLimit)
		return nil, errorx.NewCodeError(consts.DefaultErrorCode, "位置过滤数量超出系统限制")
	}
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

	var (
		enumCat   *enums.Enums
		enumEquip *enums.Enums
		enumCond  *enums.Enums
		enumTerm  *enums.Enums
	)

	// 并发解析各字典分类参数，加速反查 item_id 流程
	gParam, gCtxParam := errgroup.WithContext(l.ctx)

	if len(req.Category) > 0 {
		gParam.Go(l.fetchEnum(gCtxParam, enums.CategoryContainerCategory, req.Category, "箱型分类", &enumCat))
	}

	if len(req.EquipmentType) > 0 {
		gParam.Go(l.fetchEnum(gCtxParam, enums.CategoryEquipmentTypes, req.EquipmentType, "箱型规格", &enumEquip))
	}

	if len(req.Condition) > 0 {
		gParam.Go(l.fetchEnum(gCtxParam, enums.CategoryConditions, req.Condition, "箱况", &enumCond))
	}

	if len(req.CommercialTerm) > 0 {
		gParam.Go(l.fetchEnum(gCtxParam, enums.CategoryCommercialTerm, req.CommercialTerm, "提箱方式", &enumTerm))
	}

	if err := gParam.Wait(); err != nil {
		return nil, err
	}

	dbCategory := parseEnumItemID(enumCat)
	dbEquipmentType := parseEnumItemID(enumEquip)
	dbCondition := parseEnumItemID(enumCond)
	dbCommercialTerm := parseEnumItemID(enumTerm)

	// 解析 color 颜色参数：直接使用传入的颜色标识（如 "RAL 1015"），供底层 colors 字段的模糊匹配使用
	dbColor := req.Color

	// 游标分页限制机制，防范海量数据查询导致内存溢出 (OOM) 与 GC 压力
	limit := req.PageSize
	if limit <= 0 {
		limit = consts.DefaultPageSize
	} else if limit > consts.MaxPageSize {
		limit = consts.MaxPageSize
	}

	var (
		totalCount int64
		offersData []*offers.Offers
	)

	// 并发查询挂单总数及当前页列表数据，降低首次查询的主链路耗时
	gOffers, gCtxOffers := errgroup.WithContext(l.ctx)

	gOffers.Go(func() error {
		var err error
		totalCount, err = l.svcCtx.OffersModel.CountByLocationIdAndDirection(gCtxOffers, req.LocationIds, dbDirection, dbCategory, dbCondition, dbColor, dbEquipmentType, dbCommercialTerm, int64(req.YearOfManufactureRangeFrom))
		if err != nil {
			l.Errorf("统计挂单总数失败: %v", err)
			return err
		}
		return nil
	})

	gOffers.Go(func() error {
		var err error
		offersData, err = l.svcCtx.OffersModel.FindByLocationIdAndDirection(gCtxOffers, req.LocationIds, dbDirection, dbCategory, dbCondition, dbColor, dbEquipmentType, dbCommercialTerm, int64(req.YearOfManufactureRangeFrom), req.LastId, limit)
		if err != nil {
			l.Errorf("查询挂单列表失败: %v", err)
			return err
		}
		return nil
	})

	if err := gOffers.Wait(); err != nil {
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
		companiesMap       = make(map[int64]*companies.Companies, len(companyIdsMap))
		purchasesMap       = make(map[int64][]*membershippurchases.MembershipPurchases, len(companyIdsMap))
		vipPlansMap        = make(map[int64]*vipplans.VipPlans)
		depotsMap          = make(map[int64]*depots.Depots, len(depotIdsMap))
		locationsMap       = make(map[int64]*treenodes.TreeNodes, len(locationIdsMap))
		conditionsMap      = make(map[string]*types.EnumInfo, len(conditionIdsMap))
		equipmentTypesMap  = make(map[string]*types.EnumInfo, len(equipmentTypeIdsMap))
		commercialTermsMap = make(map[string]*types.EnumInfo, len(commercialTermIdsMap))
		categoriesMap      = make(map[string]*types.EnumInfo, len(categoryIdsMap))
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

	// 4. 并发从本地缓存加载箱况字典项详情
	if len(conditionIdsMap) > 0 {
		g.Go(l.fetchEnumMap(gCtx, enums.CategoryConditions, "箱况", conditionIdsMap, conditionsMap))
	}

	// 5. 并发从本地缓存加载箱型规格字典项详情
	if len(equipmentTypeIdsMap) > 0 {
		g.Go(l.fetchEnumMap(gCtx, enums.CategoryEquipmentTypes, "箱型规格", equipmentTypeIdsMap, equipmentTypesMap))
	}

	// 6. 并发从本地缓存加载提箱方式字典项详情
	if len(commercialTermIdsMap) > 0 {
		g.Go(l.fetchEnumMap(gCtx, enums.CategoryCommercialTerm, "提箱方式", commercialTermIdsMap, commercialTermsMap))
	}

	// 7. 并发从本地缓存加载箱型大类字典项详情
	if len(categoryIdsMap) > 0 {
		g.Go(l.fetchEnumMap(gCtx, enums.CategoryContainerCategory, "箱型分类", categoryIdsMap, categoriesMap))
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
	var attachCache *types.AttachmentsCacheInfo
	if len(item.AttachmentsCache) > 0 {
		var cache struct {
			Images []struct {
				SignedId    string `json:"signedId"`
				Filename    string `json:"filename"`
				ContentType string `json:"contentType"`
				ByteSize    int64  `json:"byteSize"`
			} `json:"images"`
			Documents []struct {
				SignedId    string `json:"signedId"`
				Filename    string `json:"filename"`
				ContentType string `json:"contentType"`
				ByteSize    int64  `json:"byteSize"`
			} `json:"documents"`
		}
		if err := json.Unmarshal([]byte(item.AttachmentsCache), &cache); err == nil {
			var imagesList []types.OfferImage
			for _, img := range cache.Images {
				imagesList = append(imagesList, types.OfferImage{
					SignedId:    img.SignedId,
					Filename:    img.Filename,
					ContentType: img.ContentType,
					ByteSize:    img.ByteSize,
					Url:         "http://api.cgboxx.com/blobs/" + img.SignedId,
				})
			}
			var docsList []types.DocumentInfo
			for _, doc := range cache.Documents {
				docsList = append(docsList, types.DocumentInfo{
					SignedId:    doc.SignedId,
					Filename:    doc.Filename,
					ContentType: doc.ContentType,
					ByteSize:    doc.ByteSize,
				})
			}
			attachCache = &types.AttachmentsCacheInfo{
				Images:    imagesList,
				Documents: docsList,
			}
		}
	}

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
		CreatedAt:                      item.CreatedAt.In(time.FixedZone("CST", 8*3600)).Format("2006-01-02T15:04:05.000-07:00"),
		UpdatedAt:                      item.UpdatedAt.In(time.FixedZone("CST", 8*3600)).Format("2006-01-02T15:04:05.000-07:00"),
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
		AttachmentsCache:               attachCache,
	}
}

// formatTime 辅助格式化 NullTime。
func formatTime(nt sql.NullTime) string {
	if nt.Valid {
		shanghaiZone := time.FixedZone("CST", 8*3600)
		return nt.Time.In(shanghaiZone).Format("2006-01-02T15:04:05.000-07:00")
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

