package treenodes

import (
	"context"
	"fmt"
	"strings"
)

// treeNodesRowsCustom 用于规避生产环境/老数据中 meta, extra 等原生类型字段为 NULL 导致的 Go 语言 Scan 崩溃问题。
const treeNodesRowsCustom = "`id`,`name`,`description`,`user_id`,COALESCE(`objects_count`, 0) as `objects_count`,`ancestry`,`ancestry_depth`,COALESCE(`position`, 0) as `position`,COALESCE(`status`, 0) as `status`,`value`,COALESCE(`visible`, 1) as `visible`,COALESCE(`children_count`, 0) as `children_count`,`data`,`created_at`,`updated_at`,`type`,`english_name`,COALESCE(`extra`, '{}') as `extra`,`lat`,`lng`,`icon`,COALESCE(`level`, 0) as `level`,`level1`,`level2`,`level3`,`level4`,`level5`,COALESCE(`is_primary`, 0) as `is_primary`,COALESCE(`meta`, '{}') as `meta`,COALESCE(`usage_count`, 0) as `usage_count`,`timezone`,`full_name`,`full_name_cn`,`search_text`"

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
