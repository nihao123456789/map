package treenodes

import (
	"context"

	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

var _ TreeNodesModel = (*customTreeNodesModel)(nil)

type (
	// TreeNodesModel is an interface to be customized, add more methods here,
	// and implement the added methods in customTreeNodesModel.
	TreeNodesModel interface {
		treeNodesModel
		withSession(session sqlx.Session) TreeNodesModel
		// FindByIds 批量查询地理位置树节点详细信息
		FindByIds(ctx context.Context, ids []int64) ([]*TreeNodes, error)
	}

	customTreeNodesModel struct {
		*defaultTreeNodesModel
	}
)

// NewTreeNodesModel returns a model for the database table.
func NewTreeNodesModel(conn sqlx.SqlConn) TreeNodesModel {
	return &customTreeNodesModel{
		defaultTreeNodesModel: newTreeNodesModel(conn),
	}
}

func (m *customTreeNodesModel) withSession(session sqlx.Session) TreeNodesModel {
	return NewTreeNodesModel(sqlx.NewSqlConnFromSession(session))
}
