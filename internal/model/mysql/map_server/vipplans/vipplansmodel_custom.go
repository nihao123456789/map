package vipplans

import (
	"context"
	"fmt"
	"strings"
)

// vipPlansRowsCustom 用于规避生产环境/老数据中 meta, role_ids 等为 NULL 导致的 Go 语言 Scan 崩溃问题。
const vipPlansRowsCustom = "`id`,COALESCE(`name`, '') as `name`,COALESCE(`name_en`, '') as `name_en`,COALESCE(`features`, '') as `features`,COALESCE(`features_en`, '') as `features_en`,COALESCE(`price`, 0.0) as `price`,COALESCE(`days`, 0) as `days`,COALESCE(`role_ids`, '[]') as `role_ids`,COALESCE(`status`, 0) as `status`,`created_at`,`updated_at`,`description`,`description_en`,COALESCE(`position`, 0) as `position`,`price_description`,`price_description_en`,`days_description`,`days_description_en`,COALESCE(`account_count`, 1) as `account_count`,`discount`,`payment_period`,COALESCE(`meta`, '{}') as `meta`,`slug`,`membership_kind`,`monthly_price`,`annual_price`,COALESCE(`permission_names`, '[]') as `permission_names`,`max_leasing_incomplete_containers`"

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
		vipPlansRowsCustom, m.table, strings.Join(placeholders, ","),
	)

	var resp []*VipPlans
	err := m.conn.QueryRowsCtx(ctx, &resp, query, args...)
	if err != nil {
		return nil, fmt.Errorf("VipPlansModel.FindByIds 批量查询失败: %w", err)
	}

	return resp, nil
}
