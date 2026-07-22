package membershippurchases

import "github.com/zeromicro/go-zero/core/stores/sqlx"

var ErrNotFound = sqlx.ErrNotFound

// membershipPurchasesRowsCustom 用于规避生产环境/老数据中 meta, status, role_ids 等字段为 NULL 导致的 Go 语言 Scan 崩溃问题。
const membershipPurchasesRowsCustom = "`id`,`company_id`,`user_id`,`vip_plan_id`,`order_id`,COALESCE(`billing_period`, '') as `billing_period`,COALESCE(`price`, 0.0) as `price`,`starts_at`,`expires_at`,COALESCE(`source`, '') as `source`,COALESCE(`status`, '') as `status`,COALESCE(`permission_names`, '[]') as `permission_names`,COALESCE(`role_ids`, '[]') as `role_ids`,`expiry_reminder_sent_at`,COALESCE(`meta`, '{}') as `meta`,`created_at`,`updated_at`,`max_leasing_incomplete_containers`,COALESCE(`activation_method`, '') as `activation_method`,`deleted_at`"
