package handler

import (
	"time"

	"github.com/artalkjs/artalk/v2/internal/core"
	"github.com/artalkjs/artalk/v2/internal/entity"
	"github.com/artalkjs/artalk/v2/server/common"
	"github.com/gofiber/fiber/v2"
)

type ParamsDashboard struct {
	SiteName string `query:"site_name" json:"site_name" validate:"optional"`
}

type DashboardMetric struct {
	Total int64 `json:"total"`
	New90 int64 `json:"new_90d"`
	Today int64 `json:"today"`
}

type DashboardTrendPoint struct {
	Date     string `json:"date"`
	Comments int64  `json:"comments"`
	Users    int64  `json:"users"`
}

type DashboardModerationSummary struct {
	Pass  int64 `json:"pass"`
	Block int64 `json:"block"`
	Error int64 `json:"error"`
}

type DashboardTopPage struct {
	ID           uint   `json:"id"`
	Key          string `json:"key"`
	Title        string `json:"title"`
	SiteName     string `json:"site_name"`
	PV           int    `json:"pv"`
	CommentCount int64  `json:"comment_count"`
}

type ResponseDashboard struct {
	PV              DashboardMetric            `json:"pv"`
	Comments        DashboardMetric            `json:"comments"`
	Users           DashboardMetric            `json:"users"`
	PendingComments int64                      `json:"pending_comments"`
	Pages           int64                      `json:"pages"`
	Trend           []DashboardTrendPoint      `json:"trend"`
	Moderation      DashboardModerationSummary `json:"moderation"`
	TopPages        []DashboardTopPage         `json:"top_pages"`
	RecentReviews   []ModerationLogItem        `json:"recent_reviews"`
}

func Dashboard(app *core.App, router fiber.Router) {
	router.Get("/dashboard", common.AdminGuard(app, func(c *fiber.Ctx) error {
		var p ParamsDashboard
		if isOK, resp := common.ParamsDecode(c, &p); !isOK {
			return resp
		}

		now := time.Now()
		startToday := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
		start90 := startToday.AddDate(0, 0, -89)

		db := app.Dao().DB()
		pageQ := db.Model(&entity.Page{})
		commentQ := db.Model(&entity.Comment{})
		if p.SiteName != "" {
			pageQ = pageQ.Where("site_name = ?", p.SiteName)
			commentQ = commentQ.Where("site_name = ?", p.SiteName)
		}

		var totalPV int64
		pageQ.Select("COALESCE(SUM(pv), 0)").Scan(&totalPV)

		var pageCount int64
		pageQ.Count(&pageCount)

		var totalComments, newComments, todayComments, pendingComments int64
		commentQ.Count(&totalComments)
		commentQ.Where("created_at >= ?", start90).Count(&newComments)
		commentQ.Where("created_at >= ?", startToday).Count(&todayComments)
		commentQ.Where("is_pending = ?", true).Count(&pendingComments)

		userQ := db.Model(&entity.User{})
		var totalUsers, newUsers, todayUsers int64
		userQ.Count(&totalUsers)
		userQ.Where("created_at >= ?", start90).Count(&newUsers)
		userQ.Where("created_at >= ?", startToday).Count(&todayUsers)

		trend := buildDashboardTrend(app, p.SiteName, start90, startToday)
		moderationSummary, recentReviews := loadDashboardModeration(app, p.SiteName, start90)
		topPages := loadDashboardTopPages(app, p.SiteName)

		return common.RespData(c, ResponseDashboard{
			PV:              DashboardMetric{Total: totalPV},
			Comments:        DashboardMetric{Total: totalComments, New90: newComments, Today: todayComments},
			Users:           DashboardMetric{Total: totalUsers, New90: newUsers, Today: todayUsers},
			PendingComments: pendingComments,
			Pages:           pageCount,
			Trend:           trend,
			Moderation:      moderationSummary,
			TopPages:        topPages,
			RecentReviews:   recentReviews,
		})
	}))
}

func buildDashboardTrend(app *core.App, siteName string, start90 time.Time, startToday time.Time) []DashboardTrendPoint {
	trend := make([]DashboardTrendPoint, 0, 90)
	byDate := map[string]*DashboardTrendPoint{}
	for day := start90; !day.After(startToday); day = day.AddDate(0, 0, 1) {
		date := day.Format("2006-01-02")
		trend = append(trend, DashboardTrendPoint{Date: date})
		byDate[date] = &trend[len(trend)-1]
	}

	var comments []entity.Comment
	commentQ := app.Dao().DB().Model(&entity.Comment{}).Where("created_at >= ?", start90)
	if siteName != "" {
		commentQ = commentQ.Where("site_name = ?", siteName)
	}
	commentQ.Find(&comments)
	for _, comment := range comments {
		if point := byDate[comment.CreatedAt.Format("2006-01-02")]; point != nil {
			point.Comments++
		}
	}

	var users []entity.User
	app.Dao().DB().Model(&entity.User{}).Where("created_at >= ?", start90).Find(&users)
	for _, user := range users {
		if point := byDate[user.CreatedAt.Format("2006-01-02")]; point != nil {
			point.Users++
		}
	}

	return trend
}

func loadDashboardModeration(app *core.App, siteName string, start90 time.Time) (DashboardModerationSummary, []ModerationLogItem) {
	q := app.Dao().DB().Model(&entity.ModerationLog{}).Where("created_at >= ?", start90)
	if siteName != "" {
		q = q.Where("site_name = ?", siteName)
	}

	var summary DashboardModerationSummary
	var rows []struct {
		Status string
		Count  int64
	}
	q.Select("status, COUNT(*) AS count").Group("status").Scan(&rows)
	for _, row := range rows {
		switch row.Status {
		case string(entity.ModerationLogStatusPass):
			summary.Pass = row.Count
		case string(entity.ModerationLogStatusBlock):
			summary.Block = row.Count
		case string(entity.ModerationLogStatusError):
			summary.Error = row.Count
		}
	}

	items := loadModerationLogItems(app, q.Order("created_at DESC").Limit(8))
	return summary, items
}

func loadDashboardTopPages(app *core.App, siteName string) []DashboardTopPage {
	q := app.Dao().DB().Model(&entity.Page{})
	if siteName != "" {
		q = q.Where("site_name = ?", siteName)
	}

	var pages []entity.Page
	q.Order("pv DESC").Limit(8).Find(&pages)

	items := make([]DashboardTopPage, 0, len(pages))
	for _, page := range pages {
		commentQ := app.Dao().DB().Model(&entity.Comment{}).
			Where("page_key = ? AND site_name = ?", page.Key, page.SiteName)
		var commentCount int64
		commentQ.Count(&commentCount)
		items = append(items, DashboardTopPage{
			ID:           page.ID,
			Key:          page.Key,
			Title:        page.Title,
			SiteName:     page.SiteName,
			PV:           page.PV,
			CommentCount: commentCount,
		})
	}
	return items
}
