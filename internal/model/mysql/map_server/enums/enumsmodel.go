package enums

import (
	"context"

	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

var _ EnumsModel = (*customEnumsModel)(nil)

type (
	// EnumsModel 提供对 enums 数据表的自定义数据访问操作接口。
	EnumsModel interface {
		enumsModel
		withSession(session sqlx.Session) EnumsModel
		// FindOneByCategoryAndValue 根据分类和值查找单条字典项记录
		FindOneByCategoryAndValue(ctx context.Context, category, value string) (*Enums, error)
	}

	customEnumsModel struct {
		*defaultEnumsModel
	}
)

// NewEnumsModel 返回一个用于操作 enums 表的 Model。
func NewEnumsModel(conn sqlx.SqlConn) EnumsModel {
	return &customEnumsModel{
		defaultEnumsModel: newEnumsModel(conn),
	}
}

func (m *customEnumsModel) withSession(session sqlx.Session) EnumsModel {
	return NewEnumsModel(sqlx.NewSqlConnFromSession(session))
}
