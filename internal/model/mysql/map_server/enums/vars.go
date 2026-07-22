package enums

import "github.com/zeromicro/go-zero/core/stores/sqlx"

// ErrNotFound 定义没有查询到记录时的通用错误变量。
var ErrNotFound = sqlx.ErrNotFound

// 全局可维护的常量定义
const (
	// CategoryConditions 代表箱况的枚举分类
	CategoryConditions = "conditions"

	// CategoryContainerCategory 代表箱型分类的枚举分类
	CategoryContainerCategory = "containerCategory"

	// CategoryEquipmentTypes 代表箱型规格的枚举分类
	CategoryEquipmentTypes = "equipmentTypes"
)
