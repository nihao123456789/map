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

	// OfferNotExpired 代表挂单未过期
	OfferNotExpired = 0
	// OfferExpired 代表挂单已过期
	OfferExpired = 1

	// OfferStatusPublished 代表挂单为已发布状态
	OfferStatusPublished = 10

	// offersRowsCustom 用于规避生产环境/老数据中可能为 NULL 的普通数值/字符类型字段导致的 Go 语言 Scan 崩溃问题。
	offersRowsCustom = "`id`,`condition`,`type`,`pickup_location_id`,`dropoff_location_id`,COALESCE(`quantity`, 0) as `quantity`,COALESCE(`pickup_charge`, 0.0) as `pickup_charge`,`free_days`,COALESCE(`per_diems`, 0.0) as `per_diems`,COALESCE(`storage_fee`, 0.0) as `storage_fee`,COALESCE(`dpp`, 0.0) as `dpp`,COALESCE(`premium`, 0.0) as `premium`,COALESCE(`new_build_price`, 0.0) as `new_build_price`,COALESCE(`depreciation_per_year`, 0.0) as `depreciation_per_year`,COALESCE(`minimum_replacement_value`, 0.0) as `minimum_replacement_value`,`additional_information`,COALESCE(`user_id`, 0) as `user_id`,`company_id`,`direction`,COALESCE(`valid_days`, 0) as `valid_days`,COALESCE(`equipment_type`, 0) as `equipment_type`,COALESCE(`commercial_term`, 0) as `commercial_term`,`comments`,COALESCE(`reviews_count`, 0) as `reviews_count`,`prefixes`,`year_of_manufacture`,`manufacturer_id`,COALESCE(`damage_protection_plan`, 0.0) as `damage_protection_plan`,COALESCE(`negotiations_count`, 0) as `negotiations_count`,`category`,`expires_at`,`review_id`,`detail`,COALESCE(`status`, 10) as `status`,`deleted_at`,COALESCE(`expires_in`, 0) as `expires_in`,`ready_for_pickup_on`,`ready_for_pickup_from`,`ready_for_pickup_to`,COALESCE(`with_lock_box`, 0) as `with_lock_box`,COALESCE(`with_forklift_pockets`, 0) as `with_forklift_pockets`,`sell_for`,`location_id`,`name`,COALESCE(`images_count`, 0) as `images_count`,COALESCE(`documents_count`, 0) as `documents_count`,COALESCE(`trade_type`, 0) as `trade_type`,`expected_delivery_on`,`expected_delivery_from`,`expected_delivery_to`,`depot_id`,`unique_number`,`created_at`,`updated_at`,`price`,`color`,`estimated_empty_delivery_date_from`,`offer_type`,COALESCE(`source`, 0) as `source`,`pickup_charge_payer`,COALESCE(`insurance_fee`, 0.0) as `insurance_fee`,`year_of_manufacture_range_from`,`year_of_manufacture_range_to`,COALESCE(`csc_test_certificate`, 0) as `csc_test_certificate`,`equipment_type_id`,`insurance_type`,COALESCE(`insurance_days`, 0) as `insurance_days`,`extra`,`colors`,`pinned_at`,`bumped_at`,COALESCE(`storage_free_days`, 0) as `storage_free_days`,`condition_tag_ids`,`condition_logo`,COALESCE(`number_of_vents`, 0) as `number_of_vents`,COALESCE(`deal_count`, 0) as `deal_count`,`csc_expires_on`,`estimated_empty_delivery_date_to`,COALESCE(`instant_sale`, 0) as `instant_sale`,`consignor_id`,`consignor_name`,`label`,`dropoff_location_ids`,`pickup_location_ids`,`original_price`,`is_special_offer`,`source_proposal_id`,COALESCE(`is_expired`, 0) as `is_expired`,COALESCE(`data_source`, 1) as `data_source`,`meta`,COALESCE(`is_non_negotiable`, 0) as `is_non_negotiable`,COALESCE(`has_damages`, 0) as `has_damages`,`with_easy_open_door`"
)
