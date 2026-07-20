package offers

import (
	"context"

	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

var _ OffersModel = (*customOffersModel)(nil)

type (
	// OffersModel 提供对 offers 数据表的自定义数据访问操作接口。
	OffersModel interface {
		offersModel
		withSession(session sqlx.Session) OffersModel
		// FindByLocationIdAndDirection 根据位置ID和交易方向查询买卖交易挂单列表（支持游标分页）
		// 参数 direction 可取值：DirectionBuy (买入) 或 DirectionSell (卖出)
		FindByLocationIdAndDirection(ctx context.Context, locationId int64, direction int64, lastId int64, limit int64) ([]*Offers, error)
	}

	customOffersModel struct {
		*defaultOffersModel
	}
)

// NewOffersModel returns a model for the database table.
func NewOffersModel(conn sqlx.SqlConn) OffersModel {
	return &customOffersModel{
		defaultOffersModel: newOffersModel(conn),
	}
}

func (m *customOffersModel) withSession(session sqlx.Session) OffersModel {
	return NewOffersModel(sqlx.NewSqlConnFromSession(session))
}
