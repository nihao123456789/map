package treenodes

import (
	"context"
	"fmt"

	"map-server/pkg/slices"
)

type TreeNodesModelCustom interface {
	// FindByIds 批量查询地理位置树节点详细信息
	FindByIds(ctx context.Context, ids []int64) ([]*TreeNodes, error)
	// FindLocationsOrderByUsage 查询所有地理树节点列表，按 usage_count 降序排列
	FindLocationsOrderByUsage(ctx context.Context) ([]*TreeNodes, error)
}

// FindByIds 批量查询地理位置树节点详细信息
func (m *customTreeNodesModel) FindByIds(ctx context.Context, ids []int64) ([]*TreeNodes, error) {
	if len(ids) == 0 {
		return nil, nil
	}

	// 动态拼接 in 占位符以规避 SQL 注入
	placeholders, args := slices.BuildInArgs(ids)

	query := fmt.Sprintf(
		"select %s from %s where `id` in (%s)",
		treeNodesRowsCustom,
		m.table,
		placeholders,
	)

	var resp []*TreeNodes
	err := m.conn.QueryRowsCtx(ctx, &resp, query, args...)
	if err != nil {
		return nil, fmt.Errorf("TreeNodesModel.FindByIds 批量查询失败: %w", err)
	}

	return resp, nil
}

// FindLocationsOrderByUsage 查询热门地理位置列表，按 usage_count 降序排列
func (m *customTreeNodesModel) FindLocationsOrderByUsage(ctx context.Context) ([]*TreeNodes, error) {
	query := fmt.Sprintf(
		"select %s from %s order by `usage_count` desc",
		treeNodesRowsCustom,
		m.table,
	)

	var resp []*TreeNodes
	err := m.conn.QueryRowsCtx(ctx, &resp, query)
	if err != nil {
		return nil, fmt.Errorf("TreeNodesModel.FindLocationsOrderByUsage 查询失败: %w", err)
	}

	return resp, nil
}
