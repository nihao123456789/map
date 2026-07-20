package offers

import (
	"context"
	"fmt"
)


// FindByLocationIdAndDirection 根据位置ID和交易方向查询买卖交易挂单列表（只查询未被逻辑删除的数据）。
//
// 参数：
//   - ctx：上下文
//   - locationId：位置/城市ID
//   - direction：交易方向（暂定：0=买入，1=卖出）
//
// 返回：满足条件的挂单列表，以及错误信息。
func (m *customOffersModel) FindByLocationIdAndDirection(ctx context.Context, locationId int64, direction int64) ([]*Offers, error) {
	query := fmt.Sprintf("select %s from %s where `location_id` = ? and `type` = ? and `direction` = ? and `deleted_at` is null", offersRows, m.table)
	var resp []*Offers
	err := m.conn.QueryRowsCtx(ctx, &resp, query, locationId, OfferTypeTrading, direction)
	if err != nil {
		return nil, fmt.Errorf("OffersModel.FindByLocationIdAndDirection 查询失败: %w", err)
	}
	return resp, nil
}
