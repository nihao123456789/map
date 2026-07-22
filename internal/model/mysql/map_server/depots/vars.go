package depots

import "github.com/zeromicro/go-zero/core/stores/sqlx"

var ErrNotFound = sqlx.ErrNotFound

// depotsRowsCustom 用于规避生产环境/老数据中 meta 和 equipment_types 为 NULL 导致的 Go 语言 Scan 崩溃问题。
const depotsRowsCustom = "`id`,`name`,`postal_code`,`website`,`phone_number`,`city`,`country`,`contact_name`,`email`,`location_id`,`local_name`,`local_address`,`language`,`source`,`address_line1`,`address_line2`,`created_at`,`updated_at`,`company_id`,`user_id`,COALESCE(`extra`, '{}') as `extra`,`lat`,`lng`,`opening_hours`,`has_repair`,COALESCE(`equipment_types`, '[]') as `equipment_types`,`unique_number`,`capacity`,`services`,`level`,`container_categories`,`reviews_count`,`reservation_enabled`,`minimum_storage_days`,`cleaning_fee`,`repair_fee`,`towing_fee`,`overdue_storage_fee`,`hazardous_surcharge`,`special_handling_fee`,`review_level`,COALESCE(`meta`, '{}') as `meta`"
