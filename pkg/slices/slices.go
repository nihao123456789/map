// Package slices 提供针对 Go 切片的常用辅助函数与泛型工具。
package slices

// Unique 对传入的切片进行去重处理，返回一个保留原始顺序的新切片。
func Unique[T comparable](slice []T) []T {
	if len(slice) == 0 {
		return slice
	}
	res := make([]T, 0, len(slice))
	seen := make(map[T]struct{}, len(slice))
	for _, val := range slice {
		if _, exists := seen[val]; !exists {
			seen[val] = struct{}{}
			res = append(res, val)
		}
	}
	return res
}
