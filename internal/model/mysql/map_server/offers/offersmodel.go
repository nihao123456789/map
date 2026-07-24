package offers

import (
	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

var _ OffersModel = (*customOffersModel)(nil)

type (
	// OffersModel 提供对 offers 数据表的自定义数据访问操作接口。
	OffersModel interface {
		offersModel
		withSession(session sqlx.Session) OffersModel
		OffersModelCustom
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
