// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.1

// Package svc 定义服务上下文（ServiceContext），
// 用于集中管理所有中间件客户端（MySQL、Redis、PostgreSQL 等）的生命周期，
// 并通过依赖注入的方式提供给各业务逻辑层使用。
package svc

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
	"github.com/zeromicro/go-zero/core/stores/sqlx"

	"map-server/internal/config"
	mysqlModel "map-server/internal/model/mysql/map_server"
	postgisModel "map-server/internal/model/postgis/map_server"
)

// ServiceContext 是服务的全局上下文，持有所有依赖的客户端实例。
// 通过 NewServiceContext 创建，贯穿整个服务生命周期。
type ServiceContext struct {
	// Config 服务配置
	Config config.Config

	// ==================== MySQL + Redis-GEO 相关 ====================

	// DB 是 go-zero 封装的 MySQL 数据库连接
	DB sqlx.SqlConn

	// RedisClient 是原生的 go-redis 客户端，用于执行 GEO 等高级命令
	RedisClient *redis.Client

	// YardModel 提供 MySQL 堆场表的数据访问
	YardModel *mysqlModel.YardModel

	// ContainerModel 提供 MySQL 集装箱表的数据访问
	ContainerModel *mysqlModel.ContainerModel

	// SyncFailureLogModel 提供 MySQL 同步失败日志表的数据访问
	SyncFailureLogModel *mysqlModel.SyncFailureLogModel

	// ==================== PostgreSQL + PostGIS 相关 ====================

	// PgPool 是 pgx/v5 的 PostgreSQL 连接池，线程安全，支持高并发
	PgPool *pgxpool.Pool

	// PostGISYardModel 提供 PostgreSQL 堆场空间查询（BBox/Radius）
	PostGISYardModel *postgisModel.PostGISYardModel

	// PostGISContainerModel 提供 PostgreSQL 集装箱空间查询（BBox/Radius）
	PostGISContainerModel *postgisModel.PostGISContainerModel
}

// NewServiceContext 初始化并返回 ServiceContext。
// 在此处完成所有中间件客户端的连接初始化，若初始化失败会直接 panic 终止启动。
//
// 参数：
//   - c：服务配置（从 etc/mapserver-dev.yaml 加载）
func NewServiceContext(c config.Config) *ServiceContext {
	// -----------------------------------------------------------
	// 初始化 MySQL 连接
	// 使用 go-zero 的 sqlx 包，支持连接池管理和慢查询日志
	// -----------------------------------------------------------
	db := sqlx.NewMysql(c.MySQL.DataSource)
	fmt.Println("MySQL 连接初始化完成")

	// -----------------------------------------------------------
	// 初始化 Redis 客户端（go-redis/v9）
	// 从配置中读取第一个 Redis 节点的地址和密码
	// -----------------------------------------------------------
	if len(c.Redis) == 0 {
		panic("Redis 配置不能为空，请检查 etc/mapserver-dev.yaml 中的 Redis 配置项")
	}
	redisCfg := c.Redis[0]
	redisClient := redis.NewClient(&redis.Options{
		Addr:     redisCfg.Host, // Redis 服务地址，格式：host:port
		Password: redisCfg.Pass, // Redis 密码，无密码时为空字符串
		DB:       0,             // 使用默认的数据库 0
	})
	fmt.Println("Redis 连接初始化完成")

	// -----------------------------------------------------------
	// 初始化 MySQL Model
	// -----------------------------------------------------------
	yardModel := mysqlModel.NewYardModel(db)
	containerModel := mysqlModel.NewContainerModel(db)
	syncFailureLogModel := mysqlModel.NewSyncFailureLogModel(db)

	// -----------------------------------------------------------
	// 初始化 PostgreSQL 连接池（pgxpool）
	// pgxpool 自动管理连接池，支持高并发场景
	// -----------------------------------------------------------
	pgPoolConfig, err := pgxpool.ParseConfig(c.PostgreSQL.DataSource)
	if err != nil {
		panic(fmt.Sprintf("解析 PostgreSQL 连接字符串失败: %v", err))
	}

	// 设置连接池最大连接数，若配置为 0 则使用默认值 10
	maxConns := c.PostgreSQL.MaxConns
	if maxConns <= 0 {
		maxConns = 10
	}
	pgPoolConfig.MaxConns = maxConns

	// 使用 context.Background() 建立连接池（服务启动时执行一次）
	pgPool, err := pgxpool.NewWithConfig(context.Background(), pgPoolConfig)
	if err != nil {
		panic(fmt.Sprintf("初始化 PostgreSQL 连接池失败: %v", err))
	}

	// 验证 PostgreSQL 连接（Ping）
	if err := pgPool.Ping(context.Background()); err != nil {
		// 连接失败仅打印警告，不阻断启动（允许 PostGIS 暂时不可用）
		fmt.Printf("警告：PostgreSQL 连接失败，PostGIS 接口暂不可用: %v\n", err)
	} else {
		fmt.Println("PostgreSQL+PostGIS 连接初始化完成")
	}

	// -----------------------------------------------------------
	// 初始化 PostGIS Model
	// -----------------------------------------------------------
	postGISYardModel := postgisModel.NewPostGISYardModel(pgPool)
	postGISContainerModel := postgisModel.NewPostGISContainerModel(pgPool)

	return &ServiceContext{
		Config:                c,
		DB:                    db,
		RedisClient:           redisClient,
		YardModel:             yardModel,
		ContainerModel:        containerModel,
		PgPool:                pgPool,
		PostGISYardModel:      postGISYardModel,
		PostGISContainerModel: postGISContainerModel,
		SyncFailureLogModel:   syncFailureLogModel,
	}
}

// Shutdown 优雅关闭服务上下文中所有的长连接资源。
func (sc *ServiceContext) Shutdown() {
	fmt.Println("正在释放服务上下文的连接资源...")
	// 1. 关闭 PostgreSQL+PostGIS 连接池
	if sc.PgPool != nil {
		sc.PgPool.Close()
		fmt.Println("PostgreSQL 连接池已安全关闭")
	}
	// 2. 关闭 Redis 客户端
	if sc.RedisClient != nil {
		_ = sc.RedisClient.Close()
		fmt.Println("Redis 客户端连接已安全关闭")
	}
}
