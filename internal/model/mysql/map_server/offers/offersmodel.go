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
		// FindByLocationIdAndDirection 根据位置ID、交易方向和箱型分类查询买卖交易挂单列表（支持游标分页）
		FindByLocationIdAndDirection(ctx context.Context, locationId int64, direction int64, category int64, lastId int64, limit int64) ([]*Offers, error)
		// CountByLocationIdAndDirection 根据位置ID、交易方向和箱型分类统计符合条件的交易挂单总数
		CountByLocationIdAndDirection(ctx context.Context, locationId int64, direction int64, category int64) (int64, error)
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
