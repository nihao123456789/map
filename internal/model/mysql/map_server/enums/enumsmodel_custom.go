package enums

import (
	"context"
	"fmt"

	"github.com/zeromicro/go-zero/core/stores/sqlx"
	"map-server/pkg/slices"
)

type EnumsModelCustom interface {
	// FindOneByCategoryAndValue 根据分类和值查找单条字典项记录
	FindOneByCategoryAndValue(ctx context.Context, category, value string) (*Enums, error)
	// FindByCategoryAndItemIds 根据分类和ID列表批量查找字典项记录
	FindByCategoryAndItemIds(ctx context.Context, category string, itemIds []string) ([]*Enums, error)
	// FindByCategories 根据分类列表批量查找所有相关的字典项记录
	FindByCategories(ctx context.Context, categories []string) ([]*Enums, error)
}

func (m *customEnumsModel) FindOneByCategoryAndValue(ctx context.Context, category, value string) (*Enums, error) {
	query := fmt.Sprintf("select %s from %s where `category` = ? and `value` = ? limit 1", enumsRows, m.table)
	var resp Enums
	err := m.conn.QueryRowCtx(ctx, &resp, query, category, value)
	switch err {
	case nil:
		return &resp, nil
	case sqlx.ErrNotFound:
		return nil, ErrNotFound
	default:
		return nil, err
	}
}

func (m *customEnumsModel) FindByCategoryAndItemIds(ctx context.Context, category string, itemIds []string) ([]*Enums, error) {
	if len(itemIds) == 0 {
		return nil, nil
	}
	placeholders, inArgs := slices.BuildInArgs(itemIds)
	args := make([]interface{}, 0, len(itemIds)+1)
	args = append(args, category)
	args = append(args, inArgs...)
	query := fmt.Sprintf("select %s from %s where `category` = ? and `item_id` in (%s)", enumsRows, m.table, placeholders)
	var resp []*Enums
	err := m.conn.QueryRowsCtx(ctx, &resp, query, args...)
	return resp, err
}

func (m *customEnumsModel) FindByCategories(ctx context.Context, categories []string) ([]*Enums, error) {
	if len(categories) == 0 {
		query := fmt.Sprintf("select %s from %s", enumsRows, m.table)
		var resp []*Enums
		err := m.conn.QueryRowsCtx(ctx, &resp, query)
		return resp, err
	}
	placeholders, args := slices.BuildInArgs(categories)
	query := fmt.Sprintf("select %s from %s where `category` in (%s)", enumsRows, m.table, placeholders)
	var resp []*Enums
	err := m.conn.QueryRowsCtx(ctx, &resp, query, args...)
	return resp, err
}
