package depots

import (
	"context"
	"fmt"
	"strings"
)

type DepotsModelCustom interface {
	// FindByIds 批量查询堆场详细信息
	FindByIds(ctx context.Context, ids []int64) ([]*Depots, error)
}

// FindByIds 批量查询堆场详细信息
func (m *customDepotsModel) FindByIds(ctx context.Context, ids []int64) ([]*Depots, error) {
	if len(ids) == 0 {
		return nil, nil
	}

	// 动态拼接 in 占位符以规避 SQL 注入
	placeholders := make([]string, len(ids))
	args := make([]interface{}, 0, len(ids))
	for i, id := range ids {
		placeholders[i] = "?"
		args = append(args, id)
	}

	query := fmt.Sprintf(
		"select %s from %s where `id` in (%s)",
		depotsRowsCustom, m.table, strings.Join(placeholders, ","),
	)

	var resp []*Depots
	err := m.conn.QueryRowsCtx(ctx, &resp, query, args...)
	if err != nil {
		return nil, fmt.Errorf("DepotsModel.FindByIds 批量查询失败: %w", err)
	}

	return resp, nil
}
