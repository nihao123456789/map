package membershippurchases

import (
	"context"

	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

var _ MembershipPurchasesModel = (*customMembershipPurchasesModel)(nil)

type (
	// MembershipPurchasesModel is an interface to be customized, add more methods here,
	// and implement the added methods in customMembershipPurchasesModel.
	MembershipPurchasesModel interface {
		membershipPurchasesModel
		withSession(session sqlx.Session) MembershipPurchasesModel
		FindActiveByCompanyIds(ctx context.Context, companyIds []int64) ([]*MembershipPurchases, error)
	}

	customMembershipPurchasesModel struct {
		*defaultMembershipPurchasesModel
	}
)

// NewMembershipPurchasesModel returns a model for the database table.
func NewMembershipPurchasesModel(conn sqlx.SqlConn) MembershipPurchasesModel {
	return &customMembershipPurchasesModel{
		defaultMembershipPurchasesModel: newMembershipPurchasesModel(conn),
	}
}

func (m *customMembershipPurchasesModel) withSession(session sqlx.Session) MembershipPurchasesModel {
	return NewMembershipPurchasesModel(sqlx.NewSqlConnFromSession(session))
}
