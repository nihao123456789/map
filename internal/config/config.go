// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.1

// Package config 定义服务的全局配置结构体。
// 配置通过 etc/mapserver-dev.yaml 文件加载。
package config

import (
	"github.com/zeromicro/go-zero/core/stores/cache"
	"github.com/zeromicro/go-zero/rest"
)

// Config 是服务的根配置结构体，内嵌 go-zero 的 HTTP 服务配置，
// 并扩展了 MySQL、Redis 和 PostgreSQL 的连接配置。
type Config struct {
	// RestConf 包含 HTTP 服务的监听地址、端口、超时等配置
	rest.RestConf

	// MySQL 数据库连接配置（业务数据存储，兼容并指向本地 MariaDB 数据库）
	MySQL MySQLConf

	// Redis 连接配置（Redis-GEO 空间缓存，使用 go-zero 内置缓存配置结构）
	Redis cache.CacheConf

	// PostgreSQL 连接配置（PostGIS 精确空间查询）
	PostgreSQL PostgreSQLConf

	// SignatureSecret 是接口签名校验的盐值/密钥
	SignatureSecret string
}

// MySQLConf 是 MySQL/MariaDB 数据库的连接配置。
type MySQLConf struct {
	// DataSource 是 MySQL/MariaDB 的 DSN 连接字符串
	// 格式：user:password@tcp(host:port)/dbname?charset=utf8mb4&parseTime=true&loc=Local
	DataSource string
}

// PostgreSQLConf 是 PostgreSQL+PostGIS 的连接配置。
type PostgreSQLConf struct {
	// DataSource 是 PostgreSQL 的 DSN 连接字符串
	// 格式：host=主机 port=端口 user=用户名 password=密码 dbname=数据库名 sslmode=disable
	DataSource string

	// MaxConns 是连接池的最大连接数，默认为 10
	MaxConns int32
}
