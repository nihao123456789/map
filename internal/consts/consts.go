// Package consts 集中定义和管理整个项目中的全局常量、业务配置参数以及系统调优常数。
package consts

import "time"

const (
	// EnumsCacheTTL 定义数据字典本地缓存的过期寿命 (5分钟)
	EnumsCacheTTL = 5 * time.Minute
	// EnumsCacheLimit 定义数据字典本地缓存的最大 Key 数量容量上限 (1000个)
	EnumsCacheLimit = 1000

	// DBMaxOpenConns 数据库最大打开连接数限制 (150个)
	DBMaxOpenConns = 150
	// DBMaxIdleConns 数据库最大闲置连接数限制 (50个)
	DBMaxIdleConns = 50
	// DBConnMaxLifetime 数据库连接最大生命周期 (1小时)
	DBConnMaxLifetime = 1 * time.Hour

	// SignatureTimeDiffLimit 定义签名防重放校验的最大允许时间差值上限 (300秒)
	SignatureTimeDiffLimit = 300

	// DefaultPageSize 默认单页数据量限制 (10条)
	DefaultPageSize = 10
	// MaxPageSize 单页最大数据量限制 (100条)
	MaxPageSize = 100

	// DefaultRateLimitQPS 默认每秒限流请求数 (100.0)
	DefaultRateLimitQPS = 100.0
	// DefaultRateLimitBurst 默认限流突发桶容量上限 (20)
	DefaultRateLimitBurst = 20

	// DefaultErrorCode 默认的业务错误状态码 (400 语义)
	DefaultErrorCode = 400
)
