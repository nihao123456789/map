package offers

import (
	"context"
	"fmt"
)

// FindByLocationIdAndDirection 根据位置ID和交易方向查询有效的买卖交易挂单列表（状态为启用且未过期，且未被逻辑删除的数据）。
// 支持游标分页：如果传入 lastId > 0，则只获取排序在该记录之后的挂单数据。
//
// 参数：
//   - ctx：上下文
//   - locationId：位置/城市ID
//   - direction：交易方向（DirectionBuy=买入，DirectionSell=卖出）
//   - lastId：上一页最后一条记录的ID（第一页传入0或不传）
//   - limit：限制返回的最多条数
//
// 返回：满足条件的挂单列表，以及错误信息。
// 示例sql：SELECT * FROM `offers` WHERE direction = 0 and `type` = 'Trading' and `deleted_at` is null and `status` = 10 and is_expired = 0 limit 10\G;
func (m *customOffersModel) FindByLocationIdAndDirection(ctx context.Context, locationId int64, direction int64, lastId int64, limit int64) ([]*Offers, error) {
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

	// 游标式分页限制
	if lastId > 0 {
		baseWhere += " and (`bumped_at` < (select `bumped_at` from `offers` where `id` = ?) or (`bumped_at` = (select `bumped_at` from `offers` where `id` = ?) and `id` < ?))"
		args = append(args, lastId, lastId, lastId)
	}

	query = fmt.Sprintf(
		"select %s from %s %s order by `bumped_at` desc, `id` desc limit ?",
		offersRows,
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
