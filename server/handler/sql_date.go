package handler

import (
	"strings"

	"gorm.io/gorm"
)

// dayGroupExpr 按天分组的 SQL 表达式（兼容不同数据库）
func dayGroupExpr(db *gorm.DB) string {
	switch db.Dialector.Name() {
	case "sqlite":
		// created_at is stored in UTC; group by the server's own calendar day so
		// "today" matches what the dashboard shows the operator.
		return "strftime('%Y-%m-%d', created_at, 'localtime')"
	case "postgres":
		return "to_char(created_at, 'YYYY-MM-DD')"
	default: // mysql 等
		return "DATE(created_at)"
	}
}

// normalizeDayString 部分驱动会把日期表达式解析成 time.Time，
// 经 database/sql 转成字符串后带上时间部分（如 2026-11-17T00:00:00Z），这里统一成 YYYY-MM-DD
func normalizeDayString(s string) string {
	s = strings.TrimSpace(s)
	if len(s) >= 10 && s[4] == '-' && s[7] == '-' {
		return s[:10]
	}
	return s
}
