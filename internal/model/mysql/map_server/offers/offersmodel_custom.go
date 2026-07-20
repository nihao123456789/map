package offers

import (
	"context"
	"fmt"
)

// FindByLocationIdAndDirection 根据位置ID和交易方向查询有效的买卖交易挂单列表（状态为启用且未过期，且未被逻辑删除的数据）。
//
// 参数：
//   - ctx：上下文
//   - locationId：位置/城市ID
//   - direction：交易方向（DirectionBuy=买入，DirectionSell=卖出）
//
// 返回：满足条件的挂单列表，以及错误信息。
// 示例sql：SELECT * FROM `offers` WHERE direction = 0 and `type` = 'Trading' and `deleted_at` is null and `status` = 10 and is_expired = 0 limit 10\G;
func (m *customOffersModel) FindByLocationIdAndDirection(ctx context.Context, locationId int64, direction int64) ([]*Offers, error) {
	query := fmt.Sprintf(
		"select %s from %s where `location_id` = ? and `type` = '%s' and `direction` = ? and `status` = %d and `is_expired` = %d and `deleted_at` is null order by `bumped_at` desc",
		offersRows,
		m.table,
		OfferTypeTrading,
		OfferStatusPublished,
		OfferNotExpired,
	)
	var resp []*Offers
	err := m.conn.QueryRowsCtx(ctx, &resp, query, locationId, direction)
	if err != nil {
		return nil, fmt.Errorf("OffersModel.FindByLocationIdAndDirection 查询失败: %w", err)
	}
	return resp, nil
}
