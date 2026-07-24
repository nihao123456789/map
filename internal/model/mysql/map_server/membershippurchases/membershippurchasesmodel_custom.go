package membershippurchases

import (
	"context"
	"fmt"

	"map-server/pkg/slices"
)

type MembershipPurchasesModelCustom interface {
	// FindActiveByCompanyIds 批量查询特定公司拥有的所有激活中、且未过期的会员订单
	FindActiveByCompanyIds(ctx context.Context, companyIds []int64) ([]*MembershipPurchases, error)
}

// FindActiveByCompanyIds 批量查询特定公司拥有的所有激活中、且未过期的会员订单
func (m *customMembershipPurchasesModel) FindActiveByCompanyIds(ctx context.Context, companyIds []int64) ([]*MembershipPurchases, error) {
	if len(companyIds) == 0 {
		return nil, nil
	}

	// 动态拼接 in 占位符以规避 SQL 注入
	placeholders, args := slices.BuildInArgs(companyIds)

	query := fmt.Sprintf(
		"select %s from %s where `company_id` in (%s) and `status` = 'active' and `expires_at` > UTC_TIMESTAMP() and `deleted_at` is null",
		membershipPurchasesRowsCustom, m.table, placeholders,
	)

	var resp []*MembershipPurchases
	err := m.conn.QueryRowsCtx(ctx, &resp, query, args...)
	if err != nil {
		return nil, fmt.Errorf("MembershipPurchasesModel.FindActiveByCompanyIds 批量查询失败: %w", err)
	}

	return resp, nil
}
