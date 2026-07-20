package vipplans

import (
	"context"

	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

var _ VipPlansModel = (*customVipPlansModel)(nil)

type (
	// VipPlansModel is an interface to be customized, add more methods here,
	// and implement the added methods in customVipPlansModel.
	VipPlansModel interface {
		vipPlansModel
		withSession(session sqlx.Session) VipPlansModel
		FindByIds(ctx context.Context, ids []int64) ([]*VipPlans, error)
	}

	customVipPlansModel struct {
		*defaultVipPlansModel
	}
)

// NewVipPlansModel returns a model for the database table.
func NewVipPlansModel(conn sqlx.SqlConn) VipPlansModel {
	return &customVipPlansModel{
		defaultVipPlansModel: newVipPlansModel(conn),
	}
}

func (m *customVipPlansModel) withSession(session sqlx.Session) VipPlansModel {
	return NewVipPlansModel(sqlx.NewSqlConnFromSession(session))
}
