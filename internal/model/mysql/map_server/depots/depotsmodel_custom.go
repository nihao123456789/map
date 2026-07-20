package depots

import (
	"context"
	"fmt"
	"strings"
)

// depotsRowsCustom 用于规避生产环境/老数据中 meta 和 equipment_types 为 NULL 导致的 Go 语言 Scan 崩溃问题。
const depotsRowsCustom = "`id`,`name`,`postal_code`,`website`,`phone_number`,`city`,`country`,`contact_name`,`email`,`location_id`,`local_name`,`local_address`,`language`,`source`,`address_line1`,`address_line2`,`created_at`,`updated_at`,`company_id`,`user_id`,COALESCE(`extra`, '{}') as `extra`,`lat`,`lng`,`opening_hours`,`has_repair`,COALESCE(`equipment_types`, '[]') as `equipment_types`,`unique_number`,`capacity`,`services`,`level`,`container_categories`,`reviews_count`,`reservation_enabled`,`minimum_storage_days`,`cleaning_fee`,`repair_fee`,`towing_fee`,`overdue_storage_fee`,`hazardous_surcharge`,`special_handling_fee`,`review_level`,COALESCE(`meta`, '{}') as `meta`"

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
