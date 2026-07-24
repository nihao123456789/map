package offers

import "github.com/zeromicro/go-zero/core/stores/sqlx"

var ErrNotFound = sqlx.ErrNotFound

// 全局可维护的常量定义
const (
	// OfferTypeTrading 代表集装箱的买卖交易挂单
	OfferTypeTrading = "Trading"

	// DirectionBuy 代表买入方向挂单（对应客户端入参 supply）
	DirectionBuy = 0
	// DirectionSell 代表卖出方向挂单（对应客户端入参 demand）
	DirectionSell = 1

	// ClientDirectionSupply 客户端传入的供应方向字符串
	ClientDirectionSupply = "supply"
	// ClientDirectionDemand 客户端传入的需求方向字符串
	ClientDirectionDemand = "demand"

	// OfferNotExpired 代表挂单未过期
	OfferNotExpired = 0
	// OfferExpired 代表挂单已过期
	OfferExpired = 1

	// OfferStatusPublished 代表挂单为已发布状态
	OfferStatusPublished = 10

	// offersRowsCustom 用于规避生产环境/老数据中可能为 NULL 的普通数值/字符类型字段导致的 Go 语言 Scan 崩溃问题。
	offersRowsCustom = "`id`,`condition`,`type`,`pickup_location_id`,`dropoff_location_id`,COALESCE(`quantity`, 0) as `quantity`,COALESCE(`pickup_charge`, 0.0) as `pickup_charge`,`free_days`,COALESCE(`per_diems`, 0.0) as `per_diems`,COALESCE(`storage_fee`, 0.0) as `storage_fee`,COALESCE(`dpp`, 0.0) as `dpp`,COALESCE(`premium`, 0.0) as `premium`,COALESCE(`new_build_price`, 0.0) as `new_build_price`,COALESCE(`depreciation_per_year`, 0.0) as `depreciation_per_year`,COALESCE(`minimum_replacement_value`, 0.0) as `minimum_replacement_value`,`additional_information`,COALESCE(`user_id`, 0) as `user_id`,`company_id`,`direction`,COALESCE(`valid_days`, 0) as `valid_days`,COALESCE(`equipment_type`, 0) as `equipment_type`,COALESCE(`commercial_term`, 0) as `commercial_term`,`comments`,COALESCE(`reviews_count`, 0) as `reviews_count`,`prefixes`,`year_of_manufacture`,`manufacturer_id`,COALESCE(`damage_protection_plan`, 0.0) as `damage_protection_plan`,COALESCE(`negotiations_count`, 0) as `negotiations_count`,`category`,`expires_at`,`review_id`,`detail`,COALESCE(`status`, 10) as `status`,`deleted_at`,COALESCE(`expires_in`, 0) as `expires_in`,`ready_for_pickup_on`,`ready_for_pickup_from`,`ready_for_pickup_to`,COALESCE(`with_lock_box`, 0) as `with_lock_box`,COALESCE(`with_forklift_pockets`, 0) as `with_forklift_pockets`,`sell_for`,`location_id`,`name`,COALESCE(`images_count`, 0) as `images_count`,COALESCE(`documents_count`, 0) as `documents_count`,COALESCE(`trade_type`, 0) as `trade_type`,`expected_delivery_on`,`expected_delivery_from`,`expected_delivery_to`,`depot_id`,`unique_number`,`created_at`,`updated_at`,`price`,`color`,`estimated_empty_delivery_date_from`,`offer_type`,COALESCE(`source`, 0) as `source`,`pickup_charge_payer`,COALESCE(`insurance_fee`, 0.0) as `insurance_fee`,`year_of_manufacture_range_from`,`year_of_manufacture_range_to`,COALESCE(`csc_test_certificate`, 0) as `csc_test_certificate`,`equipment_type_id`,`insurance_type`,COALESCE(`insurance_days`, 0) as `insurance_days`,`extra`,`colors`,`pinned_at`,`bumped_at`,COALESCE(`storage_free_days`, 0) as `storage_free_days`,`condition_tag_ids`,`condition_logo`,COALESCE(`number_of_vents`, 0) as `number_of_vents`,COALESCE(`deal_count`, 0) as `deal_count`,`csc_expires_on`,`estimated_empty_delivery_date_to`,COALESCE(`instant_sale`, 0) as `instant_sale`,`consignor_id`,`consignor_name`,`label`,`dropoff_location_ids`,`pickup_location_ids`,`original_price`,`is_special_offer`,`source_proposal_id`,COALESCE(`is_expired`, 0) as `is_expired`,COALESCE(`data_source`, 1) as `data_source`,`meta`,COALESCE(`is_non_negotiable`, 0) as `is_non_negotiable`,COALESCE(`has_damages`, 0) as `has_damages`,`with_easy_open_door`,`attachments_cache`"
)

// ColorMap 颜色映射表（RAL 编码 -> ID）
var ColorMap = map[string]int64{
	"RAL 1015": 10,
	"RAL 5010": 20,
	"RAL 3009": 30,
	"RAL 7035": 40,
	"RAL 5013": 50,
	"RAL 9003": 60,
	"RAL 7015": 70,
	"RAL 9010": 80,
}

// EquipmentTypeMap 箱型规格映射表（箱型字符串 -> ID）
var EquipmentTypeMap = map[string]int64{
	"twenty_dry_container":                             10,
	"forty_dry_container":                              20,
	"forty_high_cube":                                  30,
	"ten_dry_container":                                40,
	"ten_double_door":                                  50,
	"ten_high_cube":                                    60,
	"ten_high_cube_double_door":                        70,
	"twenty_double_door":                               80,
	"twenty_double_door_open_side_two_doors":           90,
	"twenty_double_door_open_side_full_open":           100,
	"twenty_duocon":                                    110,
	"twenty_flatrack":                                  120,
	"twenty_hard_top":                                  130,
	"twenty_open_side_two_doors":                       140,
	"twenty_open_side_full_open":                       150,
	"twenty_open_top":                                  160,
	"twenty_pallet_wide":                               170,
	"twenty_reefer":                                    180,
	"twenty_tri_door":                                  190,
	"twenty_high_cube":                                 200,
	"twenty_high_cube_double_door":                     210,
	"twenty_high_cube_double_door_open_side_two_doors": 220,
	"twenty_high_cube_double_door_open_side_full_open": 230,
	"twenty_high_cube_duocon":                          240,
	"twenty_high_cube_flatrack":                        250,
	"twenty_high_cube_hard_top":                        260,
	"twenty_high_cube_open_side_two_doors":             270,
	"twenty_high_cube_open_side_full_open":             280,
	"twenty_high_cube_open_top":                        290,
	"twenty_high_cube_pallet_wide":                     300,
	"twenty_high_cube_reefer":                          310,
	"twenty_high_cube_tri_door":                        320,
	"forty_high_cube_double_door":                      330,
	"forty_high_cube_double_door_open_side_two_doors":  340,
	"forty_high_cube_double_door_open_side_full_open":  350,
	"forty_high_cube_duocon":                           360,
	"forty_high_cube_flatrack":                         370,
	"forty_high_cube_hard_top":                         380,
	"forty_high_cube_open_side_two_doors":              390,
	"forty_high_cube_open_side_full_open":              400,
	"forty_high_cube_open_top":                         410,
	"forty_high_cube_pallet_wide":                      420,
	"forty_high_cube_reefer":                           430,
	"forty_high_cube_tri_door":                         440,
	"forty_five_high_cube":                             450,
	"forty_five_high_cube_double_door":                 460,
	"forty_five_high_cube_flatrack":                    470,
	"forty_five_high_cube_open_top":                    480,
	"forty_five_high_cube_pallet_wide":                 490,
	"forty_five_high_cube_reefer":                      500,
	"fifty_three_high_cube":                            510,
	"twenty_tank_t1":                                   520,
	"twenty_tank_t11":                                  530,
	"twenty_tank_t14":                                  540,
	"twenty_tank_t2":                                   550,
	"twenty_tank_t20":                                  560,
	"twenty_tank_t22":                                  570,
	"twenty_tank_t3":                                   580,
	"twenty_tank_t4":                                   590,
	"twenty_tank_t50":                                  600,
	"twenty_tank_t7":                                   610,
	"twenty_tank_t75":                                  620,
	"forty_tank_t75":                                   630,
	"tank_container":                                   640,
}
