package vipplans

import (
	"context"
	"fmt"
	"strings"
)

// FindByIds 批量查询会员规格模板
func (m *customVipPlansModel) FindByIds(ctx context.Context, ids []int64) ([]*VipPlans, error) {
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
		vipPlansRows, m.table, strings.Join(placeholders, ","),
	)

	var resp []*VipPlans
	err := m.conn.QueryRowsCtx(ctx, &resp, query, args...)
	if err != nil {
		return nil, fmt.Errorf("VipPlansModel.FindByIds 批量查询失败: %w", err)
	}

	return resp, nil
}
