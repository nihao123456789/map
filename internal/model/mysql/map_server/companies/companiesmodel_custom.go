package companies

import (
	"context"
	"fmt"
	"strings"
)

// FindByIds 根据多个公司ID批量查询公司详细信息（状态正常且未被逻辑删除的数据）。
// 采用原生批量占位符拼接，避开额外的第三方包依赖和 Rebind 开销。
//
// 参数：
//   - ctx：上下文
//   - ids：公司ID列表
//
// 返回：查询到的公司详细信息列表，以及错误信息。
func (m *customCompaniesModel) FindByIds(ctx context.Context, ids []int64) ([]*Companies, error) {
	if len(ids) == 0 {
		return nil, nil
	}

	// 拼接占位符，例如 (?, ?, ?)
	placeholders := make([]string, len(ids))
	args := make([]interface{}, len(ids))
	for i, id := range ids {
		placeholders[i] = "?"
		args[i] = id
	}

	query := fmt.Sprintf(
		"select %s from %s where `id` in (%s) and `deleted_at` is null",
		companiesRows,
		m.table,
		strings.Join(placeholders, ","),
	)

	var resp []*Companies
	err := m.conn.QueryRowsCtx(ctx, &resp, query, args...)
	if err != nil {
		return nil, fmt.Errorf("CompaniesModel.FindByIds 批量查询失败: %w", err)
	}
	return resp, nil
}
