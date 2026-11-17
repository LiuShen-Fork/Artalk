package handler

import (
	"time"

	"github.com/artalkjs/artalk/v2/internal/core"
	"github.com/artalkjs/artalk/v2/internal/entity"
	"github.com/artalkjs/artalk/v2/server/common"
	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

// 默认展示近半年的活跃度（热力图宽度约 26 周）
const (
	userActivityDefaultDays  = 182
	userActivityMaxDays      = 365
	userActivityDefaultLimit = 10
	userActivityMaxLimit     = 50
)

type ParamsUserActivity struct {
	Days  int `query:"days" json:"days" validate:"optional"`   // 活跃度统计的天数
	Limit int `query:"limit" json:"limit" validate:"optional"` // 最近发言的条数
}

type UserActivityDay struct {
	Date  string `json:"date"`  // YYYY-MM-DD
	Count int64  `json:"count"` // 当天发表数
}

type ResponseUserActivity struct {
	Count     int64                  `json:"count"`      // 评论总数
	Pending   int64                  `json:"pending"`    // 待审核数量
	Votes     int64                  `json:"votes"`      // 收到的赞同总数
	FirstDate string                 `json:"first_date"` // 首次发言日期
	ByDay     []UserActivityDay      `json:"by_day"`     // 每日发言数（近 N 天，仅含有发言的日期）
	Recent    []entity.CookedComment `json:"recent"`     // 最近发言
}

// UserActivity 当前用户的发言统计
//
// 控制台的「我的概览」用它绘制热力图与最近发言列表。登录用户按 user_id 统计，
// 未登录但持有访客凭据（X-Artalk-Guest）时按 guest_token_hash 统计，
// 两者都没有时返回空数据而不是报错，页面照常渲染。
//
// @Id           UserActivity
// @Summary      获取当前用户的发言统计与最近发言
// @Tags         User
// @Security     ApiKeyAuth
// @Param        options  query  ParamsUserActivity  false  "选项"
// @Produce      json
// @Success      200  {object}  ResponseUserActivity
// @Router       /user/activity [get]
func UserActivity(app *core.App, router fiber.Router) {
	router.Get("/user/activity", func(c *fiber.Ctx) error {
		var p ParamsUserActivity
		if isOK, resp := common.ParamsDecode(c, &p); !isOK {
			return resp
		}

		days := clampInt(p.Days, userActivityDefaultDays, 1, userActivityMaxDays)
		limit := clampInt(p.Limit, userActivityDefaultLimit, 1, userActivityMaxLimit)

		result := ResponseUserActivity{
			ByDay:  []UserActivityDay{},
			Recent: []entity.CookedComment{},
		}

		scope, hasScope := userActivityScope(app, c)
		if !hasScope {
			return common.RespData(c, result)
		}

		db := app.Dao().DB()
		base := func() *gorm.DB {
			return db.Model(&entity.Comment{}).Where(scope.column+" = ?", scope.value)
		}

		base().Count(&result.Count)
		base().Where("is_pending = ?", true).Count(&result.Pending)

		var votes struct{ Total int64 }
		base().Select("COALESCE(SUM(vote_up), 0) as total").Scan(&votes)
		result.Votes = votes.Total

		var first struct{ CreatedAt *time.Time }
		base().Select("MIN(created_at) as created_at").Scan(&first)
		if first.CreatedAt != nil {
			result.FirstDate = first.CreatedAt.Format("2006-01-02")
		}

		// 热力图：按天聚合（数据库各异的日期表达式统一在 dayGroupExpr 里处理）
		since := time.Now().AddDate(0, 0, -(days - 1))
		since = time.Date(since.Year(), since.Month(), since.Day(), 0, 0, 0, 0, since.Location())
		base().
			Select(dayGroupExpr(db) + " as date, count(*) as count").
			Where("created_at >= ?", since).
			Group(dayGroupExpr(db)).Order("date ASC").Scan(&result.ByDay)
		for i := range result.ByDay {
			result.ByDay[i].Date = normalizeDayString(result.ByDay[i].Date)
		}

		// 最近发言
		var comments []*entity.Comment
		base().Order("created_at DESC").Limit(limit).Find(&comments)
		result.Recent = app.Dao().CookAllComments(comments)

		return common.RespData(c, result)
	})
}

// userCommentScope 请求身份对应的评论归属条件
type userCommentScope struct {
	column string
	value  any
}

// userActivityScope 把请求身份转换成 WHERE 条件，无法识别身份时 hasScope 为 false
func userActivityScope(app *core.App, c *fiber.Ctx) (scope userCommentScope, hasScope bool) {
	user, _ := common.GetUserByReq(app, c)
	if !user.IsEmpty() {
		return userCommentScope{"user_id", user.ID}, true
	}
	if hash := common.GuestTokenHash(c); hash != "" {
		return userCommentScope{"guest_token_hash", hash}, true
	}
	return userCommentScope{}, false
}

func clampInt(value, fallback, min, max int) int {
	if value <= 0 {
		return fallback
	}
	if value < min {
		return min
	}
	if value > max {
		return max
	}
	return value
}
