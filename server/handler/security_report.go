package handler

import (
	"time"

	"github.com/artalkjs/artalk/v2/internal/core"
	"github.com/artalkjs/artalk/v2/internal/entity"
	"github.com/artalkjs/artalk/v2/server/common"
	"github.com/gofiber/fiber/v2"
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
		dayCounts, err := countByLocalDay(db.Model(&entity.SecurityEvent{}), since)
		if err != nil {
			return common.RespError(c, fiber.StatusInternalServerError, "Failed to count security events by day: "+err.Error())
		}
		for _, day := range dayCounts {
			byDay = append(byDay, SecurityDayCount{Date: day.Date, Count: day.Count})
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
