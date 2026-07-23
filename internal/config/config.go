// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.1

// Package config 定义服务的全局配置结构体。
// 配置通过 etc/mapserver-dev.yaml 文件加载。
package config

import (
	"github.com/zeromicro/go-zero/rest"
)

// Config 是服务的根配置结构体，内嵌 go-zero 的 HTTP 服务配置，
// 并扩展了 MySQL 的连接配置。
type Config struct {
	// RestConf 包含 HTTP 服务的监听地址、端口、超时等配置
	rest.RestConf

	// MySQL 数据库连接配置（业务数据存储，兼容并指向本地 MariaDB 数据库）
	MySQL MySQLConf

	// SignatureSecret 是接口签名校验的盐值/密钥
	SignatureSecret string

	// RateLimit 限流配置 (单机令牌桶防爆流)
	RateLimit struct {
		Limit float64 `json:"Limit"` // 每秒允许的请求数 (QPS)
		Burst int     `json:"Burst"` // 允许的最大突发请求量 (Burst)
	}
}

// MySQLConf 是 MySQL/MariaDB 数据库的连接配置。
type MySQLConf struct {
	// DataSource 是 MySQL/MariaDB 的 DSN 连接字符串
	// 格式：user:password@tcp(host:port)/dbname?charset=utf8mb4&parseTime=true&loc=Local
	DataSource string
}

