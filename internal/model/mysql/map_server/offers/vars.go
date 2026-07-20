package offers

import "github.com/zeromicro/go-zero/core/stores/sqlx"

var ErrNotFound = sqlx.ErrNotFound

// 全局可维护的常量定义
const (
	// OfferTypeTrading 代表集装箱的买卖交易挂单
	OfferTypeTrading = "Trading"

	// DirectionBuy 代表买入方向挂单（对应客户端入参 supply）
	DirectionBuy = 0
	// DirectionSell 代表卖出方向挂单（对应客户端入参 demand）
	DirectionSell = 1

	// OfferNotExpired 代表挂单未过期
	OfferNotExpired = 0
	// OfferExpired 代表挂单已过期
	OfferExpired = 1

	// OfferStatusActive 代表挂单为有效上架状态
	OfferStatusActive = 10
)
