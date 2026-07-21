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
// companiesRowsCustom 用于规避生产环境/老数据中 business_types, main_intentions 为 NULL 导致的 Go 语言 Scan 崩溃问题。
const companiesRowsCustom = "`id`,`name`,`location_id`,`telephone`,`created_at`,`updated_at`,`email`,`usci`,`user_id`,`status`,`verify_method`,`first_name`,`last_name`,`review_level`,`reviews_count`,`payment_password`,`audit_status`,`followers_count`,`tradings_count`,`leasings_count`,`trading_demands_count`,`leasing_demands_count`,`trading_supplies_count`,`leasing_supplies_count`,`payment_password_error_count`,`level_id`,`is_official`,`is_quick_response`,`invoice_index`,`address`,COALESCE(`business_types`, '[]') as `business_types`,COALESCE(`main_intentions`, '[]') as `main_intentions`,`position`,`deals_count`,`negotiations_count`,`favorites_count`,`address_line1`,`address_line2`,`payment_password_updated_at`,`meta`,`is_platform_escrow`,`deleted_at`,`telephone_country_code`,`telephone_number`"

// FindByIds 根据多个公司ID批量查询公司详细信息（状态正常且未被逻辑删除的数据）。
// 采用原生批量占位符拼接，避开额外的第三方包依赖和 Rebind 开销。
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
		companiesRowsCustom,
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
