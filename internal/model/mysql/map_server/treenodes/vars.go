package treenodes

import "github.com/zeromicro/go-zero/core/stores/sqlx"

var ErrNotFound = sqlx.ErrNotFound

// treeNodesRowsCustom 用于规避生产环境/老数据中 meta, extra 等原生类型字段为 NULL 导致的 Go 语言 Scan 崩溃问题。
const treeNodesRowsCustom = "`id`,`name`,`description`,`user_id`,COALESCE(`objects_count`, 0) as `objects_count`,`ancestry`,`ancestry_depth`,COALESCE(`position`, 0) as `position`,COALESCE(`status`, 0) as `status`,`value`,COALESCE(`visible`, 1) as `visible`,COALESCE(`children_count`, 0) as `children_count`,`data`,`created_at`,`updated_at`,`type`,`english_name`,COALESCE(`extra`, '{}') as `extra`,`lat`,`lng`,`icon`,COALESCE(`level`, 0) as `level`,`level1`,`level2`,`level3`,`level4`,`level5`,COALESCE(`is_primary`, 0) as `is_primary`,COALESCE(`meta`, '{}') as `meta`,COALESCE(`usage_count`, 0) as `usage_count`,`timezone`,`full_name`,`full_name_cn`,`search_text`"
