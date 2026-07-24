// Package times 提供常用的时间与日期格式化辅助工具函数。
package times

import (
	"database/sql"
	"time"
)

// FormatDateTime 辅助格式化 time.Time 为 CST 格式字符串。
func FormatDateTime(t time.Time) string {
	shanghaiZone := time.FixedZone("CST", 8*3600)
	return t.In(shanghaiZone).Format("2006-01-02T15:04:05.000-07:00")
}

// FormatTime 辅助格式化 NullTime 为 CST 格式字符串。
func FormatTime(nt sql.NullTime) string {
	if nt.Valid {
		return FormatDateTime(nt.Time)
	}
	return ""
}
