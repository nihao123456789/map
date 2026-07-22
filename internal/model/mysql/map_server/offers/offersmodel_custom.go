package offers

import (
	"context"
	"fmt"
)

// buildWhereAndArgs 拼装查询/统计集装箱交易挂单的公共 WHERE 条件和 SQL 参数
func (m *customOffersModel) buildWhereAndArgs(locationId int64, direction int64, category int64, condition int64, color int64, equipmentType int64, commercialTerm int64, yearOfManufactureRangeFrom int64) (string, []interface{}) {
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

	// 如果 condition > 0，则追加 condition 箱况条件进行过滤
	if condition > 0 {
		baseWhere += " and `condition` = ?"
		args = append(args, condition)
	}

	// 如果 color > 0，则追加 color 颜色条件进行过滤
	if color > 0 {
		baseWhere += " and `color` = ?"
		args = append(args, color)
	}

	// 如果 equipmentType > 0，则追加 equipment_type 箱型规格条件进行过滤
	if equipmentType > 0 {
		baseWhere += " and `equipment_type` = ?"
		args = append(args, equipmentType)
	}

	// 如果 commercialTerm > 0，则追加 commercial_term 提箱方式条件进行过滤
	if commercialTerm > 0 {
		baseWhere += " and `commercial_term` = ?"
		args = append(args, commercialTerm)
	}

	// 如果 yearOfManufactureRangeFrom > 0，则追加 year_of_manufacture_range_from 生产年份起步条件进行过滤
	if yearOfManufactureRangeFrom > 0 {
		baseWhere += " and `year_of_manufacture_range_from` >= ?"
		args = append(args, yearOfManufactureRangeFrom)
	}

	return baseWhere, args
}

// FindByLocationIdAndDirection 根据位置ID和交易方向查询有效的买卖交易挂单列表（状态为启用且未过期，且未被逻辑删除的数据）。
// 支持游标分页：如果传入 lastId > 0，则只获取排序在该记录之后的挂单数据。
func (m *customOffersModel) FindByLocationIdAndDirection(ctx context.Context, locationId int64, direction int64, category int64, condition int64, color int64, equipmentType int64, commercialTerm int64, yearOfManufactureRangeFrom int64, lastId int64, limit int64) ([]*Offers, error) {
	baseWhere, args := m.buildWhereAndArgs(locationId, direction, category, condition, color, equipmentType, commercialTerm, yearOfManufactureRangeFrom)

	// 游标式分页限制
	if lastId > 0 {
		baseWhere += " and (`bumped_at` < (select `bumped_at` from `offers` where `id` = ?) or (`bumped_at` = (select `bumped_at` from `offers` where `id` = ?) and `id` < ?))"
		args = append(args, lastId, lastId, lastId)
	}

	query := fmt.Sprintf(
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

// CountByLocationIdAndDirection 根据位置ID、交易方向、箱型分类、箱况、颜色、规格箱型、提箱方式和生产年份起步统计符合条件的交易挂单总数
func (m *customOffersModel) CountByLocationIdAndDirection(ctx context.Context, locationId int64, direction int64, category int64, condition int64, color int64, equipmentType int64, commercialTerm int64, yearOfManufactureRangeFrom int64) (int64, error) {
	baseWhere, args := m.buildWhereAndArgs(locationId, direction, category, condition, color, equipmentType, commercialTerm, yearOfManufactureRangeFrom)

	query := fmt.Sprintf("select count(*) from %s %s", m.table, baseWhere)

	var total int64
	err := m.conn.QueryRowCtx(ctx, &total, query, args...)
	if err != nil {
		return 0, fmt.Errorf("OffersModel.CountByLocationIdAndDirection 统计失败: %w", err)
	}
	return total, nil
}
