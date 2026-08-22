package handler

import (
	"github.com/artalkjs/artalk/v2/internal/core"
	"github.com/artalkjs/artalk/v2/internal/entity"
	"github.com/artalkjs/artalk/v2/server/common"
	"github.com/gofiber/fiber/v2"
)

type ParamsAIAssistantLogList struct {
	SiteName string `query:"site_name" json:"site_name" validate:"optional"`
	Status   string `query:"status" json:"status" validate:"optional"`
	Limit    int    `query:"limit" json:"limit" validate:"optional"`
	Offset   int    `query:"offset" json:"offset" validate:"optional"`
}

type ResponseAIAssistantLogList struct {
	Total int64                   `json:"count"`
	Logs  []entity.AIAssistantLog `json:"logs"`
}

func AIAssistantLogList(app *core.App, router fiber.Router) {
	router.Get("/ai-assistant/logs", common.AdminGuard(app, func(c *fiber.Ctx) error {
		var p ParamsAIAssistantLogList
		if isOK, resp := common.ParamsDecode(c, &p); !isOK {
			return resp
		}
		q := app.Dao().DB().Model(&entity.AIAssistantLog{}).Order("created_at DESC")
		if p.SiteName != "" {
			q = q.Where("site_name = ?", p.SiteName)
		}
		if p.Status != "" {
			q = q.Where("status = ?", p.Status)
		}
		var total int64
		q.Count(&total)
		var logs []entity.AIAssistantLog
		q.Scopes(Paginate(p.Offset, p.Limit)).Find(&logs)
		return common.RespData(c, ResponseAIAssistantLogList{Total: total, Logs: logs})
	}))
}

func AIAssistantLogDelete(app *core.App, router fiber.Router) {
	router.Delete("/ai-assistant/logs/:id", common.AdminGuard(app, func(c *fiber.Ctx) error {
		id, err := c.ParamsInt("id")
		if err != nil || id < 1 {
			return common.RespError(c, fiber.StatusBadRequest, "invalid AI assistant log ID")
		}
		result := app.Dao().DB().Delete(&entity.AIAssistantLog{}, id)
		if result.Error != nil {
			return common.RespError(c, fiber.StatusInternalServerError, "AI assistant log deletion failed")
		}
		if result.RowsAffected == 0 {
			return common.RespError(c, fiber.StatusNotFound, "AI assistant log not found")
		}
		return common.RespSuccess(c)
	}))
}

func AIAssistantLogClear(app *core.App, router fiber.Router) {
	router.Delete("/ai-assistant/logs", common.AdminGuard(app, func(c *fiber.Ctx) error {
		var p struct {
			SiteName string `query:"site_name" json:"site_name" validate:"optional"`
		}
		if isOK, resp := common.ParamsDecode(c, &p); !isOK {
			return resp
		}
		q := app.Dao().DB()
		if p.SiteName != "" {
			q = q.Where("site_name = ?", p.SiteName)
		}
		if err := q.Delete(&entity.AIAssistantLog{}).Error; err != nil {
			return common.RespError(c, fiber.StatusInternalServerError, "AI assistant log cleanup failed")
		}
		return common.RespSuccess(c)
	}))
}
