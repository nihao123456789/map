package companies

import (
	"context"

	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

var _ CompaniesModel = (*customCompaniesModel)(nil)

type (
	// CompaniesModel is an interface to be customized, add more methods here,
	// and implement the added methods in customCompaniesModel.
	CompaniesModel interface {
		companiesModel
		withSession(session sqlx.Session) CompaniesModel
		// FindByIds 根据多个公司ID批量查询公司详细信息
		FindByIds(ctx context.Context, ids []int64) ([]*Companies, error)
	}

	customCompaniesModel struct {
		*defaultCompaniesModel
	}
)

// NewCompaniesModel returns a model for the database table.
func NewCompaniesModel(conn sqlx.SqlConn) CompaniesModel {
	return &customCompaniesModel{
		defaultCompaniesModel: newCompaniesModel(conn),
	}
}

func (m *customCompaniesModel) withSession(session sqlx.Session) CompaniesModel {
	return &customCompaniesModel{
		defaultCompaniesModel: newCompaniesModel(sqlx.NewSqlConnFromSession(session)),
	}
}
