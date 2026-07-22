package companies

import "github.com/zeromicro/go-zero/core/stores/sqlx"

var ErrNotFound = sqlx.ErrNotFound

// companiesRowsCustom 用于规避生产环境/老数据中 business_types, main_intentions 为 NULL 导致的 Go 语言 Scan 崩溃问题。
const companiesRowsCustom = "`id`,`name`,`location_id`,`telephone`,`created_at`,`updated_at`,`email`,`usci`,`user_id`,`status`,`verify_method`,`first_name`,`last_name`,`review_level`,`reviews_count`,`payment_password`,`audit_status`,`followers_count`,`tradings_count`,`leasings_count`,`trading_demands_count`,`leasing_demands_count`,`trading_supplies_count`,`leasing_supplies_count`,`payment_password_error_count`,`level_id`,`is_official`,`is_quick_response`,`invoice_index`,`address`,COALESCE(`business_types`, '[]') as `business_types`,COALESCE(`main_intentions`, '[]') as `main_intentions`,`position`,`deals_count`,`negotiations_count`,`favorites_count`,`address_line1`,`address_line2`,`payment_password_updated_at`,`meta`,`is_platform_escrow`,`deleted_at`,`telephone_country_code`,`telephone_number`"
