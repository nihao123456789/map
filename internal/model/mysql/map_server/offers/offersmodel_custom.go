package offers

import (
	"context"
	"fmt"
)

// offersRowsCustom 用于规避生产环境/老数据中可能为 NULL 的普通数值/字符类型字段导致的 Go 语言 Scan 崩溃问题。
const offersRowsCustom = "`id`,`condition`,`type`,`pickup_location_id`,`dropoff_location_id`,COALESCE(`quantity`, 0) as `quantity`,COALESCE(`pickup_charge`, 0.0) as `pickup_charge`,`free_days`,COALESCE(`per_diems`, 0.0) as `per_diems`,COALESCE(`storage_fee`, 0.0) as `storage_fee`,COALESCE(`dpp`, 0.0) as `dpp`,COALESCE(`premium`, 0.0) as `premium`,COALESCE(`new_build_price`, 0.0) as `new_build_price`,COALESCE(`depreciation_per_year`, 0.0) as `depreciation_per_year`,COALESCE(`minimum_replacement_value`, 0.0) as `minimum_replacement_value`,`additional_information`,COALESCE(`user_id`, 0) as `user_id`,`company_id`,`direction`,COALESCE(`valid_days`, 0) as `valid_days`,COALESCE(`equipment_type`, 0) as `equipment_type`,COALESCE(`commercial_term`, 0) as `commercial_term`,`comments`,COALESCE(`reviews_count`, 0) as `reviews_count`,`prefixes`,`year_of_manufacture`,`manufacturer_id`,COALESCE(`damage_protection_plan`, 0.0) as `damage_protection_plan`,COALESCE(`negotiations_count`, 0) as `negotiations_count`,`category`,`expires_at`,`review_id`,`detail`,COALESCE(`status`, 10) as `status`,`deleted_at`,COALESCE(`expires_in`, 0) as `expires_in`,`ready_for_pickup_on`,`ready_for_pickup_from`,`ready_for_pickup_to`,COALESCE(`with_lock_box`, 0) as `with_lock_box`,COALESCE(`with_forklift_pockets`, 0) as `with_forklift_pockets`,`sell_for`,`location_id`,`name`,COALESCE(`images_count`, 0) as `images_count`,COALESCE(`documents_count`, 0) as `documents_count`,COALESCE(`trade_type`, 0) as `trade_type`,`expected_delivery_on`,`expected_delivery_from`,`expected_delivery_to`,`depot_id`,`unique_number`,`created_at`,`updated_at`,`price`,`color`,`estimated_empty_delivery_date_from`,`offer_type`,COALESCE(`source`, 0) as `source`,`pickup_charge_payer`,COALESCE(`insurance_fee`, 0.0) as `insurance_fee`,`year_of_manufacture_range_from`,`year_of_manufacture_range_to`,COALESCE(`csc_test_certificate`, 0) as `csc_test_certificate`,`equipment_type_id`,`insurance_type`,COALESCE(`insurance_days`, 0) as `insurance_days`,`extra`,`colors`,`pinned_at`,`bumped_at`,COALESCE(`storage_free_days`, 0) as `storage_free_days`,`condition_tag_ids`,`condition_logo`,COALESCE(`number_of_vents`, 0) as `number_of_vents`,COALESCE(`deal_count`, 0) as `deal_count`,`csc_expires_on`,`estimated_empty_delivery_date_to`,COALESCE(`instant_sale`, 0) as `instant_sale`,`consignor_id`,`consignor_name`,`label`,`dropoff_location_ids`,`pickup_location_ids`,`original_price`,`is_special_offer`,`source_proposal_id`,COALESCE(`is_expired`, 0) as `is_expired`,COALESCE(`data_source`, 1) as `data_source`,`meta`,COALESCE(`is_non_negotiable`, 0) as `is_non_negotiable`,COALESCE(`has_damages`, 0) as `has_damages`,`with_easy_open_door`"

// FindByLocationIdAndDirection 根据位置ID和交易方向查询有效的买卖交易挂单列表（状态为启用且未过期，且未被逻辑删除的数据）。
// 支持游标分页：如果传入 lastId > 0，则只获取排序在该记录之后的挂单数据。
func (m *customOffersModel) FindByLocationIdAndDirection(ctx context.Context, locationId int64, direction int64, category int64, lastId int64, limit int64) ([]*Offers, error) {
	var query string
	var args []interface{}

	// 基础条件拼接，固定过滤 Trading 挂单、对应方向、已发布 (status=10)、且未过期 (is_expired=0) 并过滤物理删除
	baseWhere := fmt.Sprintf(
		"where `type` = '%s' and `direction` = ? and `status` = %d and `is_expired` = %d and `deleted_at` is null",
		OfferTypeTrading,
		OfferStatusPublished,
		OfferNotExpired,
	)
	args = append(args, direction)

	// 如果 locationId > 0，则追加 location_id 条件进行过滤；否则（不传或为0）查询所有位置的挂单数据
	if locationId > 0 {
		baseWhere += " and `location_id` = ?"
		args = append(args, locationId)
	}

	// 如果 category > 0，则追加 category 箱型分类条件进行过滤
	if category > 0 {
		baseWhere += " and `category` = ?"
		args = append(args, category)
	}

	// 游标式分页限制
	if lastId > 0 {
		baseWhere += " and (`bumped_at` < (select `bumped_at` from `offers` where `id` = ?) or (`bumped_at` = (select `bumped_at` from `offers` where `id` = ?) and `id` < ?))"
		args = append(args, lastId, lastId, lastId)
	}

	query = fmt.Sprintf(
		"select %s from %s %s order by `bumped_at` desc, `id` desc limit ?",
		offersRowsCustom,
		m.table,
		baseWhere,
	)
	args = append(args, limit)

	var resp []*Offers
	err := m.conn.QueryRowsCtx(ctx, &resp, query, args...)
	if err != nil {
		return nil, fmt.Errorf("OffersModel.FindByLocationIdAndDirection 查询失败: %w", err)
	}
	return resp, nil
}

// CountByLocationIdAndDirection 根据位置ID、交易方向和箱型分类统计符合条件的交易挂单总数
func (m *customOffersModel) CountByLocationIdAndDirection(ctx context.Context, locationId int64, direction int64, category int64) (int64, error) {
	var query string
	var args []interface{}

	// 基础过滤条件，过滤 Trading 挂单、对应方向、已发布 (status=10)、且未过期 (is_expired=0) 并过滤物理删除
	baseWhere := fmt.Sprintf(
		"where `type` = '%s' and `direction` = ? and `status` = %d and `is_expired` = %d and `deleted_at` is null",
		OfferTypeTrading,
		OfferStatusPublished,
		OfferNotExpired,
	)
	args = append(args, direction)

	// 如果 locationId > 0，则追加 location_id 统计过滤；否则不限位置
	if locationId > 0 {
		baseWhere += " and `location_id` = ?"
		args = append(args, locationId)
	}

	// 如果 category > 0，则追加 category 统计过滤
	if category > 0 {
		baseWhere += " and `category` = ?"
		args = append(args, category)
	}

	query = fmt.Sprintf("select count(*) from %s %s", m.table, baseWhere)

	var total int64
	err := m.conn.QueryRowCtx(ctx, &total, query, args...)
	if err != nil {
		return 0, fmt.Errorf("OffersModel.CountByLocationIdAndDirection 统计失败: %w", err)
	}
	return total, nil
}
