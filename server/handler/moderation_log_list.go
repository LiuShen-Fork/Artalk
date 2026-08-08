package handler

import (
	"github.com/artalkjs/artalk/v2/internal/core"
	"github.com/artalkjs/artalk/v2/internal/entity"
	"github.com/artalkjs/artalk/v2/server/common"
	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

type ParamsModerationLogList struct {
	SiteName string `query:"site_name" json:"site_name" validate:"optional"`
	Checker  string `query:"checker" json:"checker" validate:"optional"`
	Status   string `query:"status" json:"status" validate:"optional"`
	Limit    int    `query:"limit" json:"limit" validate:"optional"`
	Offset   int    `query:"offset" json:"offset" validate:"optional"`
}

type ModerationLogItem struct {
	entity.CookedModerationLog
	CommentContent string `json:"comment_content"`
	CommentPending bool   `json:"comment_pending"`
	UserName       string `json:"user_name"`
	UserEmail      string `json:"user_email"`
}

type ResponseModerationLogList struct {
	Total int64               `json:"count"`
	Logs  []ModerationLogItem `json:"logs"`
}

func ModerationLogList(app *core.App, router fiber.Router) {
	router.Get("/moderation/logs", common.AdminGuard(app, func(c *fiber.Ctx) error {
		var p ParamsModerationLogList
		if isOK, resp := common.ParamsDecode(c, &p); !isOK {
			return resp
		}

		q := app.Dao().DB().Model(&entity.ModerationLog{}).Order("created_at DESC")
		if p.SiteName != "" {
			q = q.Where("site_name = ?", p.SiteName)
		}
		if p.Checker != "" {
			q = q.Where("checker = ?", p.Checker)
		}
		if p.Status != "" {
			q = q.Where("status = ?", p.Status)
		}

		var total int64
		q.Count(&total)

		logs := loadModerationLogItems(app, q.Scopes(Paginate(p.Offset, p.Limit)))
		return common.RespData(c, ResponseModerationLogList{Total: total, Logs: logs})
	}))
}

func loadModerationLogItems(app *core.App, q *gorm.DB) []ModerationLogItem {
	var logs []entity.ModerationLog
	q.Find(&logs)

	items := make([]ModerationLogItem, 0, len(logs))
	for _, logRow := range logs {
		item := ModerationLogItem{CookedModerationLog: cookModerationLog(logRow)}
		comment := app.Dao().FindComment(logRow.CommentID)
		if !comment.IsEmpty() {
			item.CommentContent = comment.Content
			item.CommentPending = comment.IsPending
			user := app.Dao().FetchUserForComment(&comment)
			item.UserName = user.Name
			item.UserEmail = user.Email
		}
		items = append(items, item)
	}
	return items
}

func cookModerationLog(logRow entity.ModerationLog) entity.CookedModerationLog {
	return entity.CookedModerationLog{
		ID:        logRow.ID,
		CommentID: logRow.CommentID,
		SiteName:  logRow.SiteName,
		PageKey:   logRow.PageKey,
		UserID:    logRow.UserID,
		Checker:   logRow.Checker,
		Status:    logRow.Status,
		Action:    logRow.Action,
		Message:   logRow.Message,
		Date:      logRow.CreatedAt.Format("2006-01-02 15:04:05"),
	}
}
