// Package consts 集中定义和管理整个项目中的全局常量、业务配置参数以及缓存 Key 构造方法。
package consts

import "fmt"

// GetEnumValCacheKey 生成数据字典项具体属性值（value）反查对应的缓存键名。
// 格式为：enum_val:{category}:{value}
func GetEnumValCacheKey(category string, value string) string {
	return fmt.Sprintf("enum_val:%s:%s", category, value)
}

// GetEnumsCacheKey 生成特定分类下全量数据字典列表的缓存键名。
// 格式为：enums:{category}
func GetEnumsCacheKey(category string) string {
	return fmt.Sprintf("enums:%s", category)
}
