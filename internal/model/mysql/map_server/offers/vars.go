package offers

import "github.com/zeromicro/go-zero/core/stores/sqlx"

var ErrNotFound = sqlx.ErrNotFound

// 全局可维护的常量定义
const (
	// OfferTypeTrading 代表集装箱的买卖交易挂单
	OfferTypeTrading = "Trading"
)
