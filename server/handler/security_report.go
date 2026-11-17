package handler

import (
	"strings"
	"time"

	"github.com/artalkjs/artalk/v2/internal/core"
	"github.com/artalkjs/artalk/v2/internal/entity"
	"github.com/artalkjs/artalk/v2/server/common"
	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

type SecurityKindCount struct {
	Kind  string `json:"kind"`
	Count int64  `json:"count"`
}

type SecurityActionCount struct {
	Action string `json:"action"`
	Count  int64  `json:"count"`
}

type SecurityDayCount struct {
	Date  string `json:"date"`
	Count int64  `json:"count"`
}

type SecurityIPCount struct {
	IP       string `json:"ip"`
	IPRegion string `json:"ip_region"`
	Count    int64  `json:"count"`
}

type SecurityPathCount struct {
	Path  string `json:"path"`
	Count int64  `json:"count"`
}

type ResponseSecurityReport struct {
	RangeDays int                   `json:"range_days"`
	Total     int64                 `json:"total"`
	Blocked   int64                 `json:"blocked"`
	ByKind    []SecurityKindCount   `json:"by_kind"`
	ByAction  []SecurityActionCount `json:"by_action"`
	ByDay     []SecurityDayCount    `json:"by_day"`
	TopIPs    []SecurityIPCount     `json:"top_ips"`
	TopPaths  []SecurityPathCount   `json:"top_paths"`
}

// SecurityReport 安全攻击报表
//
// @Id           SecurityReport
// @Summary      获取安全事件统计报表
// @Tags         Security
// @Security     ApiKeyAuth
// @Produce      json
// @Param        days  query  int  false  "统计最近 N 天 (默认 7)"
// @Success      200  {object}  ResponseSecurityReport
// @Router       /security/report [get]
func SecurityReport(app *core.App, router fiber.Router) {
	router.Get("/security/report", common.AdminGuard(app, func(c *fiber.Ctx) error {
		var p struct {
			Days int `query:"days" json:"days" validate:"optional"`
		}
		if isOK, resp := common.ParamsDecode(c, &p); !isOK {
			return resp
		}

		days := p.Days
		if days <= 0 {
			days = 7
		}
		if days > 365 {
			days = 365
		}

		db := app.Dao().DB()
		since := time.Now().AddDate(0, 0, -days)
		base := db.Model(&entity.SecurityEvent{}).Where("created_at >= ?", since)

		var total int64
		base.Count(&total)

		var blocked int64
		db.Model(&entity.SecurityEvent{}).
			Where("created_at >= ? AND blocked = ?", since, true).
			Count(&blocked)

		byKind := []SecurityKindCount{}
		db.Model(&entity.SecurityEvent{}).
			Select("kind as kind, count(*) as count").
			Where("created_at >= ?", since).
			Group("kind").Order("count DESC").Scan(&byKind)

		byAction := []SecurityActionCount{}
		db.Model(&entity.SecurityEvent{}).
			Select("action as action, count(*) as count").
			Where("created_at >= ?", since).
			Group("action").Order("count DESC").Scan(&byAction)

		byDay := []SecurityDayCount{}
		db.Model(&entity.SecurityEvent{}).
			Select(securityDateGroupExpr(db)+" as date, count(*) as count").
			Where("created_at >= ?", since).
			Group(securityDateGroupExpr(db)).Order("date ASC").Scan(&byDay)
		for i := range byDay {
			byDay[i].Date = normalizeDayString(byDay[i].Date)
		}

		topIPs := []SecurityIPCount{}
		db.Model(&entity.SecurityEvent{}).
			Select("ip as ip, count(*) as count").
			Where("created_at >= ?", since).
			Group("ip").Order("count DESC").Limit(20).Scan(&topIPs)

		topPaths := []SecurityPathCount{}
		db.Model(&entity.SecurityEvent{}).
			Select("path as path, count(*) as count").
			Where("created_at >= ?", since).
			Group("path").Order("count DESC").Limit(20).Scan(&topPaths)

		if ipRegion, err := core.AppService[*core.IPRegionService](app); err == nil && ipRegion != nil {
			for i := range topIPs {
				topIPs[i].IPRegion = ipRegion.Query(topIPs[i].IP)
			}
		}

		return common.RespData(c, ResponseSecurityReport{
			RangeDays: days,
			Total:     total,
			Blocked:   blocked,
			ByKind:    byKind,
			ByAction:  byAction,
			ByDay:     byDay,
			TopIPs:    topIPs,
			TopPaths:  topPaths,
		})
	}))
}

// securityDateGroupExpr 按天分组的 SQL 表达式（兼容不同数据库）
func securityDateGroupExpr(db *gorm.DB) string {
	switch db.Dialector.Name() {
	case "sqlite":
		return "strftime('%Y-%m-%d', created_at)"
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
