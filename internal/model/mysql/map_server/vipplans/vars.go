package vipplans

import "github.com/zeromicro/go-zero/core/stores/sqlx"

var ErrNotFound = sqlx.ErrNotFound

// vipPlansRowsCustom 用于规避生产环境/老数据中 meta, role_ids 等为 NULL 导致的 Go 语言 Scan 崩溃问题。
const vipPlansRowsCustom = "`id`,COALESCE(`name`, '') as `name`,COALESCE(`name_en`, '') as `name_en`,COALESCE(`features`, '') as `features`,COALESCE(`features_en`, '') as `features_en`,COALESCE(`price`, 0.0) as `price`,COALESCE(`days`, 0) as `days`,COALESCE(`role_ids`, '[]') as `role_ids`,COALESCE(`status`, 0) as `status`,`created_at`,`updated_at`,`description`,`description_en`,COALESCE(`position`, 0) as `position`,`price_description`,`price_description_en`,`days_description`,`days_description_en`,COALESCE(`account_count`, 1) as `account_count`,`discount`,`payment_period`,COALESCE(`meta`, '{}') as `meta`,`slug`,`membership_kind`,`monthly_price`,`annual_price`,COALESCE(`permission_names`, '[]') as `permission_names`,`max_leasing_incomplete_containers`"
