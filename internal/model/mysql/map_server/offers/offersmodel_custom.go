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

	if lastId > 0 {
		// 游标分页条件：
		// 倒序规则下，上一页最后一条记录的 (bumped_at, id) 组成了游标的分界点。
		// 新一页的数据必须满足 (bumped_at, id) < (last_bumped_at, last_id)
		query = fmt.Sprintf(
			"select %s from %s where `location_id` = ? and `type` = '%s' and `direction` = ? and `status` = %d and `is_expired` = %d and `deleted_at` is null and (`bumped_at` < (select `bumped_at` from `offers` where `id` = ?) or (`bumped_at` = (select `bumped_at` from `offers` where `id` = ?) and `id` < ?)) order by `bumped_at` desc, `id` desc limit ?",
			offersRows,
			m.table,
			OfferTypeTrading,
			OfferStatusPublished,
			OfferNotExpired,
		)
		args = []interface{}{locationId, direction, lastId, lastId, lastId, limit}
	} else {
		query = fmt.Sprintf(
			"select %s from %s where `location_id` = ? and `type` = '%s' and `direction` = ? and `status` = %d and `is_expired` = %d and `deleted_at` is null order by `bumped_at` desc, `id` desc limit ?",
			offersRows,
			m.table,
			OfferTypeTrading,
			OfferStatusPublished,
			OfferNotExpired,
		)
		args = []interface{}{locationId, direction, limit}
	}

	var resp []*Offers
	err := m.conn.QueryRowsCtx(ctx, &resp, query, args...)
	if err != nil {
		return nil, fmt.Errorf("OffersModel.FindByLocationIdAndDirection 查询失败: %w", err)
	}
	return resp, nil
}
