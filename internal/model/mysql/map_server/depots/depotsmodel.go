package depots

import (
	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

var _ DepotsModel = (*customDepotsModel)(nil)

type (
	// DepotsModel is an interface to be customized, add more methods here,
	// and implement the added methods in customDepotsModel.
	DepotsModel interface {
		depotsModel
		withSession(session sqlx.Session) DepotsModel
		DepotsModelCustom
	}

	customDepotsModel struct {
		*defaultDepotsModel
	}
)

// NewDepotsModel returns a model for the database table.
func NewDepotsModel(conn sqlx.SqlConn) DepotsModel {
	return &customDepotsModel{
		defaultDepotsModel: newDepotsModel(conn),
	}
}

func (m *customDepotsModel) withSession(session sqlx.Session) DepotsModel {
	return NewDepotsModel(sqlx.NewSqlConnFromSession(session))
}
