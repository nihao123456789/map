package membershippurchases

import (
	"context"
	"fmt"
	"strings"
)

// membershipPurchasesRowsCustom 用于规避生产环境/老数据中 meta, status, role_ids 等字段为 NULL 导致的 Go 语言 Scan 崩溃问题。
const membershipPurchasesRowsCustom = "`id`,`company_id`,`user_id`,`vip_plan_id`,`order_id`,COALESCE(`billing_period`, '') as `billing_period`,COALESCE(`price`, 0.0) as `price`,`starts_at`,`expires_at`,COALESCE(`source`, '') as `source`,COALESCE(`status`, '') as `status`,COALESCE(`permission_names`, '[]') as `permission_names`,COALESCE(`role_ids`, '[]') as `role_ids`,`expiry_reminder_sent_at`,COALESCE(`meta`, '{}') as `meta`,`created_at`,`updated_at`,`max_leasing_incomplete_containers`,COALESCE(`activation_method`, '') as `activation_method`,`deleted_at`"

// FindActiveByCompanyIds 批量查询特定公司拥有的所有激活中、且未过期的会员订单
func (m *customMembershipPurchasesModel) FindActiveByCompanyIds(ctx context.Context, companyIds []int64) ([]*MembershipPurchases, error) {
	if len(companyIds) == 0 {
		return nil, nil
	}

	// 动态拼接 in 占位符以规避 SQL 注入
	placeholders := make([]string, len(companyIds))
	args := make([]interface{}, 0, len(companyIds))
	for i, id := range companyIds {
		placeholders[i] = "?"
		args = append(args, id)
	}

	query := fmt.Sprintf(
		"select %s from %s where `company_id` in (%s) and `status` = 'active' and `expires_at` > NOW() and `deleted_at` is null",
		membershipPurchasesRowsCustom, m.table, strings.Join(placeholders, ","),
	)

	var resp []*MembershipPurchases
	err := m.conn.QueryRowsCtx(ctx, &resp, query, args...)
	if err != nil {
		return nil, fmt.Errorf("MembershipPurchasesModel.FindActiveByCompanyIds 批量查询失败: %w", err)
	}

	return resp, nil
}
