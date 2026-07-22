package treenodes

import (
	"context"
	"fmt"
	"strings"
)

// FindByIds 批量查询地理位置树节点详细信息
func (m *customTreeNodesModel) FindByIds(ctx context.Context, ids []int64) ([]*TreeNodes, error) {
	if len(ids) == 0 {
		return nil, nil
	}

	// 动态拼接 in 占位符以规避 SQL 注入
	placeholders := make([]string, len(ids))
	args := make([]interface{}, len(ids))
	for i, id := range ids {
		placeholders[i] = "?"
		args[i] = id
	}

	query := fmt.Sprintf(
		"select %s from %s where `id` in (%s)",
		treeNodesRowsCustom,
		m.table,
		strings.Join(placeholders, ","),
	)

	var resp []*TreeNodes
	err := m.conn.QueryRowsCtx(ctx, &resp, query, args...)
	if err != nil {
		return nil, fmt.Errorf("TreeNodesModel.FindByIds 批量查询失败: %w", err)
	}

	return resp, nil
}
