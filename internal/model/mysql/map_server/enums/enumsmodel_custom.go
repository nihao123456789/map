package enums

import (
	"context"
	"fmt"
	"strings"

	"github.com/zeromicro/go-zero/core/stores/sqlx"
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
	placeholders := make([]string, len(itemIds))
	args := make([]interface{}, 0, len(itemIds)+1)
	args = append(args, category)
	for i, id := range itemIds {
		placeholders[i] = "?"
		args = append(args, id)
	}
	query := fmt.Sprintf("select %s from %s where `category` = ? and `item_id` in (%s)", enumsRows, m.table, strings.Join(placeholders, ","))
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
	placeholders := make([]string, len(categories))
	args := make([]interface{}, len(categories))
	for i, cat := range categories {
		placeholders[i] = "?"
		args[i] = cat
	}
	query := fmt.Sprintf("select %s from %s where `category` in (%s)", enumsRows, m.table, strings.Join(placeholders, ","))
	var resp []*Enums
	err := m.conn.QueryRowsCtx(ctx, &resp, query, args...)
	return resp, err
}
