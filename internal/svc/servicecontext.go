// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.1

// Package svc 定义服务上下文（ServiceContext），
// 用于集中管理所有中间件客户端（MySQL、Redis、PostgreSQL 等）的生命周期，
// 并通过依赖注入的方式提供给各业务逻辑层使用。
package svc

import (
	"fmt"

	"golang.org/x/time/rate"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/zeromicro/go-zero/core/collection"
	"github.com/zeromicro/go-zero/core/stores/sqlx"

	"map-server/internal/config"
	"map-server/internal/consts"
	"map-server/internal/middleware"
	mysqlModel "map-server/internal/model/mysql/map_server"
	"map-server/internal/model/mysql/map_server/offers"
	"map-server/internal/model/mysql/map_server/companies"
	"map-server/internal/model/mysql/map_server/vipplans"
	"map-server/internal/model/mysql/map_server/membershippurchases"
	"map-server/internal/model/mysql/map_server/depots"
	"map-server/internal/model/mysql/map_server/treenodes"
	"map-server/internal/model/mysql/map_server/enums"
	postgisModel "map-server/internal/model/postgis/map_server"

	"github.com/zeromicro/go-zero/rest"
)

// ServiceContext 是服务的全局上下文，持有所有依赖的客户端实例。
// 通过 NewServiceContext 创建，贯穿整个服务生命周期。
type ServiceContext struct {
	// Config 服务配置
	Config config.Config

	// ==================== MySQL 业务存储相关 ====================

	// DB 是 go-zero 封装 of MySQL/MariaDB 数据库连接
	DB sqlx.SqlConn

	// SyncFailureLogModel 提供 MySQL 同步失败日志表的数据访问
	SyncFailureLogModel *mysqlModel.SyncFailureLogModel

	// OffersModel 提供 MySQL 挂单表的数据访问
	OffersModel offers.OffersModel

	// CompaniesModel 提供 MySQL 公司表的数据访问
	CompaniesModel companies.CompaniesModel

	// VipPlansModel 提供 MySQL 会员规格套餐方案表的数据访问
	VipPlansModel vipplans.VipPlansModel

	// MembershipPurchasesModel 提供 MySQL 会员订单购买记录表的数据访问
	MembershipPurchasesModel membershippurchases.MembershipPurchasesModel

	// DepotsModel 提供 MySQL 堆场表的数据访问
	DepotsModel depots.DepotsModel

	// TreeNodesModel 提供 MySQL 地理位置树节点数据表的访问
	TreeNodesModel treenodes.TreeNodesModel

	// EnumsModel 提供 MySQL 数据字典表的访问
	EnumsModel enums.EnumsModel

	// ==================== PostgreSQL + PostGIS 相关 ====================

	// PgPool 是 pgx/v5 的 PostgreSQL 连接池，线程安全，支持高并发
	PgPool *pgxpool.Pool

	// PostGISYardModel 提供 PostgreSQL 堆场空间查询（BBox/Radius）
	PostGISYardModel *postgisModel.PostGISYardModel

	// PostGISContainerModel 提供 PostgreSQL 集装箱空间查询（BBox/Radius）
	PostGISContainerModel *postgisModel.PostGISContainerModel

	// SignatureMiddleware 签名校验路由中间件
	SignatureMiddleware rest.Middleware

	// RateLimiter 全局限流器
	RateLimiter *rate.Limiter

	// EnumsCache 系统字典本地定时缓存，失效时间 5 分钟，防数据库击穿
	EnumsCache *collection.Cache
}

// NewServiceContext 初始化并返回 ServiceContext。
// 在此处完成所有中间件客户端的连接初始化，若初始化失败会直接 panic 终止启动。
//
// 参数：
//   - c：服务配置（从 etc/mapserver-dev.yaml 加载）
func NewServiceContext(c config.Config) *ServiceContext {
	// -----------------------------------------------------------
	// 初始化 MySQL/MariaDB 连接
	// 使用 go-zero 的 sqlx 包，支持连接池管理和慢查询日志
	// 注：此处兼容并使用本地的 MariaDB 数据库（协议与驱动完全相同）。
	// -----------------------------------------------------------
	db := sqlx.NewMysql(c.MySQL.DataSource)
	// 并发架构优化：为底层连接池配置最大打开连接数与空闲数，提升连接的复用与生命周期管理
	if rawDB, err := db.RawDB(); err == nil && rawDB != nil {
		rawDB.SetMaxOpenConns(consts.DBMaxOpenConns)
		rawDB.SetMaxIdleConns(consts.DBMaxIdleConns)
		rawDB.SetConnMaxLifetime(consts.DBConnMaxLifetime)
	}
	fmt.Println("MySQL/MariaDB 连接初始化完成")


	// -----------------------------------------------------------
	// 初始化 MySQL Model
	// -----------------------------------------------------------
	syncFailureLogModel := mysqlModel.NewSyncFailureLogModel(db)
	offersModel := offers.NewOffersModel(db)
	companiesModel := companies.NewCompaniesModel(db)
	vipplansModel := vipplans.NewVipPlansModel(db)
	membershippurchasesModel := membershippurchases.NewMembershipPurchasesModel(db)
	depotsModel := depots.NewDepotsModel(db)
	treenodesModel := treenodes.NewTreeNodesModel(db)
	enumsModel := enums.NewEnumsModel(db)

	// -----------------------------------------------------------
	// 初始化 PostgreSQL 连接池（pgxpool）
	// 注意：当前项目版本暂未使用 PostgreSQL 与 PostGIS，因此跳过 PostgreSQL 连接初始化。
	// -----------------------------------------------------------
	var pgPool *pgxpool.Pool = nil
	fmt.Println("跳过 PostgreSQL 连接初始化（当前项目未使用 PostgreSQL）")

	// -----------------------------------------------------------
	// 初始化 PostGIS Model
	// -----------------------------------------------------------
	postGISYardModel := postgisModel.NewPostGISYardModel(pgPool)
	postGISContainerModel := postgisModel.NewPostGISContainerModel(pgPool)

	enumsCache, err := collection.NewCache(consts.EnumsCacheTTL, collection.WithLimit(consts.EnumsCacheLimit))
	if err != nil {
		panic(fmt.Sprintf("创建字典内存缓存失败: %v", err))
	}

	return &ServiceContext{
		Config:                c,
		DB:                    db,
		PgPool:                pgPool,
		PostGISYardModel:      postGISYardModel,
		PostGISContainerModel: postGISContainerModel,
		SyncFailureLogModel:   syncFailureLogModel,
		OffersModel:              offersModel,
		CompaniesModel:           companiesModel,
		VipPlansModel:            vipplansModel,
		MembershipPurchasesModel: membershippurchasesModel,
		DepotsModel:              depotsModel,
		TreeNodesModel:           treenodesModel,
		EnumsModel:               enumsModel,
		SignatureMiddleware:      middleware.NewSignatureMiddleware(c.SignatureSecret).Handle,
		RateLimiter:              rate.NewLimiter(rate.Limit(c.RateLimit.Limit), c.RateLimit.Burst),
		EnumsCache:               enumsCache,
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
}
