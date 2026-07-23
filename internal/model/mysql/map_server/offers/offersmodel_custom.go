package offers

import (
	"context"
	"fmt"
)

// buildWhereAndArgs 拼装查询/统计集装箱交易挂单的公共 WHERE 条件和 SQL 参数
func (m *customOffersModel) buildWhereAndArgs(locationId int64, direction int64, category int64, condition int64, color string, equipmentType int64, commercialTerm int64, yearOfManufactureRangeFrom int64) (string, []interface{}) {
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

	// 如果 color 不为空，则使用 MariaDB/MySQL 原生的 JSON_CONTAINS 进行规范匹配，防范模糊检索的误匹配风险
	// 【未来高并发大数据量性能调优预案】：
	// 在 MariaDB 11.2+ 中，JSON_CONTAINS 会触发全表扫描。
	// 当后续数据规模增长（如达到数十万条以上）时，可在数据库端执行以下 DDL 命令建立全文索引进行检索性能暴增式提速：
	// DDL 命令：ALTER TABLE `offers` ADD FULLTEXT INDEX `idx_colors_ft` (`colors`);
	// 对应地，本处 Go 过滤逻辑改写为：
	//     baseWhere += " and MATCH(`colors`) AGAINST(? IN BOOLEAN MODE)"
	//     args = append(args, `+"\"`+color+`\""`)
	if len(color) > 0 {
		baseWhere += " and JSON_CONTAINS(`colors`, ?)"
		// 在 JSON_CONTAINS 中，查找的值需为合法的 JSON 片段（即带双引号的颜色字符串）
		args = append(args, `"`+color+`"`)
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
func (m *customOffersModel) FindByLocationIdAndDirection(ctx context.Context, locationId int64, direction int64, category int64, condition int64, color string, equipmentType int64, commercialTerm int64, yearOfManufactureRangeFrom int64, lastId int64, limit int64) ([]*Offers, error) {
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
func (m *customOffersModel) CountByLocationIdAndDirection(ctx context.Context, locationId int64, direction int64, category int64, condition int64, color string, equipmentType int64, commercialTerm int64, yearOfManufactureRangeFrom int64) (int64, error) {
	baseWhere, args := m.buildWhereAndArgs(locationId, direction, category, condition, color, equipmentType, commercialTerm, yearOfManufactureRangeFrom)

	query := fmt.Sprintf("select count(*) from %s %s", m.table, baseWhere)

	var total int64
	err := m.conn.QueryRowCtx(ctx, &total, query, args...)
	if err != nil {
		return 0, fmt.Errorf("OffersModel.CountByLocationIdAndDirection 统计失败: %w", err)
	}
	return total, nil
}

// FindCountGroupByLocationId 根据交易方向按位置ID分组统计有效的买卖交易挂单数量
// 【未来高并发大数据量性能调优预案】：
// 目前数据规模下使用基础索引即可秒级返回。若未来挂单数据量级达到数百万甚至千万级别，
// 为了避免大量回表或临时表排序，建议在数据库中为 `offers` 表创建以下联合覆盖索引（Covering Index）：
// DDL 命令：ALTER TABLE `offers` ADD INDEX `idx_trading_location_count` (`type`, `direction`, `status`, `is_expired`, `location_id`);
// 一旦建立该索引，本查询将被数据库优化器直接识别为 "Using index"（覆盖索引），完全在索引树中完成快速过滤与分组聚合，实现毫秒级极致响应。
func (m *customOffersModel) FindCountGroupByLocationId(ctx context.Context, direction int64) ([]*LocationCountResult, error) {
	// 基础条件拼接，固定过滤 Trading 挂单、对应方向、已发布 (status=10)、且未过期 (is_expired=0) 并过滤物理删除
	query := fmt.Sprintf(
		"select `location_id`, count(*) as `count` from %s where `type` = '%s' and `direction` = ? and `status` = %d and `is_expired` = %d and `deleted_at` is null group by `location_id`",
		m.table,
		OfferTypeTrading,
		OfferStatusPublished,
		OfferNotExpired,
	)

	var resp []*LocationCountResult
	err := m.conn.QueryRowsCtx(ctx, &resp, query, direction)
	if err != nil {
		return nil, fmt.Errorf("OffersModel.FindCountGroupByLocationId 统计失败: %w", err)
	}
	return resp, nil
}
