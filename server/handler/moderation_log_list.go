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
	CommentContent   string `json:"comment_content"`
	CommentAvailable bool   `json:"comment_available"`
	CommentRid       uint   `json:"comment_rid"`
	CommentUA        string `json:"comment_ua"`
	CommentIP        string `json:"comment_ip"`
	CommentPending   bool   `json:"comment_pending"`
	CommentCollapsed bool   `json:"comment_collapsed"`
	CommentPinned    bool   `json:"comment_pinned"`
	UserName         string `json:"user_name"`
	UserEmail        string `json:"user_email"`
	UserLink         string `json:"user_link"`
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

		q := moderationLogAttentionQuery(app.Dao().DB().Model(&entity.ModerationLog{})).Order("created_at DESC")
		if p.SiteName != "" {
			q = q.Where("site_name = ?", p.SiteName)
		}
		if p.Checker != "" {
			q = q.Where("checker = ?", p.Checker)
		}
		if p.Status == "replace" {
			q = q.Where("action = ?", entity.ModerationLogActionReplace)
		} else if p.Status != "" {
			q = q.Where("status = ?", p.Status)
		}

		var total int64
		q.Count(&total)

		logs := loadModerationLogItems(app, q.Scopes(Paginate(p.Offset, p.Limit)))
		return common.RespData(c, ResponseModerationLogList{Total: total, Logs: logs})
	}))
}

func moderationLogAttentionQuery(q *gorm.DB) *gorm.DB {
	return q.Where(
		"(status IN ? OR action = ?)",
		[]string{string(entity.ModerationLogStatusBlock), string(entity.ModerationLogStatusError)},
		entity.ModerationLogActionReplace,
	)
}

func ModerationLogDelete(app *core.App, router fiber.Router) {
	router.Delete("/moderation/logs/:id", common.AdminGuard(app, func(c *fiber.Ctx) error {
		id, err := c.ParamsInt("id")
		if err != nil || id < 1 {
			return common.RespError(c, fiber.StatusBadRequest, "invalid moderation log ID")
		}

		result := app.Dao().DB().Delete(&entity.ModerationLog{}, id)
		if result.Error != nil {
			return common.RespError(c, fiber.StatusInternalServerError, "moderation log deletion failed")
		}
		if result.RowsAffected == 0 {
			return common.RespError(c, fiber.StatusNotFound, "moderation log not found")
		}
		return common.RespSuccess(c)
	}))
}

func ModerationLogClear(app *core.App, router fiber.Router) {
	router.Delete("/moderation/logs", common.AdminGuard(app, func(c *fiber.Ctx) error {
		var p struct {
			SiteName string `query:"site_name" json:"site_name" validate:"optional"`
		}
		if isOK, resp := common.ParamsDecode(c, &p); !isOK {
			return resp
		}

		q := app.Dao().DB()
		if p.SiteName != "" {
			q = q.Where("site_name = ?", p.SiteName)
		} else {
			q = q.Where("1 = 1")
		}
		if err := q.Delete(&entity.ModerationLog{}).Error; err != nil {
			return common.RespError(c, fiber.StatusInternalServerError, "moderation log cleanup failed")
		}
		return common.RespSuccess(c)
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
			item.CommentAvailable = true
			item.CommentContent = comment.Content
			item.CommentRid = comment.Rid
			item.CommentUA = comment.UA
			item.CommentIP = comment.IP
			item.CommentPending = comment.IsPending
			item.CommentCollapsed = comment.IsCollapsed
			item.CommentPinned = comment.IsPinned
			user := app.Dao().FetchUserForComment(&comment)
			item.UserName = user.Name
			item.UserEmail = user.Email
			item.UserLink = user.Link
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
