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

type ParamsSecurityEventList struct {
	IP     string `query:"ip" json:"ip" validate:"optional"`
	Kind   string `query:"kind" json:"kind" validate:"optional"`
	Scope  string `query:"scope" json:"scope" validate:"optional"`
	Days   int    `query:"days" json:"days" validate:"optional"` // 最近 N 天，0 = 不限
	Limit  int    `query:"limit" json:"limit" validate:"optional"`
	Offset int    `query:"offset" json:"offset" validate:"optional"`
}

type ResponseSecurityEventList struct {
	Total  int64                       `json:"count"`
	Events []entity.CookedSecurityEvent `json:"events"`
}

// SecurityEventList 查询安全事件记录
//
// @Id           SecurityEventList
// @Summary      获取安全事件记录
// @Tags         Security
// @Security     ApiKeyAuth
// @Produce      json
// @Success      200  {object}  ResponseSecurityEventList
// @Router       /security/events [get]
func SecurityEventList(app *core.App, router fiber.Router) {
	router.Get("/security/events", common.AdminGuard(app, func(c *fiber.Ctx) error {
		var p ParamsSecurityEventList
		if isOK, resp := common.ParamsDecode(c, &p); !isOK {
			return resp
		}

		q := securityEventQuery(app, p)
		var total int64
		q.Count(&total)

		var events []entity.SecurityEvent
		q.Order("created_at DESC").Scopes(Paginate(p.Offset, p.Limit)).Find(&events)

		items := make([]entity.CookedSecurityEvent, 0, len(events))
		for _, event := range events {
			items = append(items, cookSecurityEvent(app, event))
		}

		return common.RespData(c, ResponseSecurityEventList{Total: total, Events: items})
	}))
}

func securityEventQuery(app *core.App, p ParamsSecurityEventList) *gorm.DB {
	q := app.Dao().DB().Model(&entity.SecurityEvent{})

	if ip := strings.TrimSpace(p.IP); ip != "" {
		q = q.Where("ip = ?", ip)
	}
	if kind := strings.TrimSpace(p.Kind); kind != "" {
		q = q.Where("kind = ?", kind)
	}
	if scope := strings.TrimSpace(p.Scope); scope != "" {
		q = q.Where("scope = ?", scope)
	}
	if p.Days > 0 {
		q = q.Where("created_at >= ?", time.Now().AddDate(0, 0, -p.Days))
	}
	return q
}

func cookSecurityEvent(app *core.App, event entity.SecurityEvent) entity.CookedSecurityEvent {
	item := entity.CookedSecurityEvent{
		ID:       event.ID,
		IP:       event.IP,
		Kind:     event.Kind,
		Scope:    event.Scope,
		Window:   event.Window,
		Action:   event.Action,
		Path:     event.Path,
		Method:   event.Method,
		SiteName: event.SiteName,
		UserID:   event.UserID,
		UserName: event.UserName,
		UA:       event.UA,
		Detail:   event.Detail,
		Blocked:  event.Blocked,
		Date:     event.CreatedAt.Format("2006-01-02 15:04:05"),
	}
	if ipRegion, err := core.AppService[*core.IPRegionService](app); err == nil && ipRegion != nil {
		item.IPRegion = ipRegion.Query(event.IP)
	}
	return item
}

// SecurityEventDelete 删除一条安全事件记录
//
// @Id           SecurityEventDelete
// @Summary      删除安全事件记录
// @Tags         Security
// @Security     ApiKeyAuth
// @Produce      json
// @Success      200  {object}  common.ResponseSuccess
// @Router       /security/events/{id} [delete]
func SecurityEventDelete(app *core.App, router fiber.Router) {
	router.Delete("/security/events/:id", common.AdminGuard(app, func(c *fiber.Ctx) error {
		id, err := c.ParamsInt("id")
		if err != nil || id < 1 {
			return common.RespError(c, fiber.StatusBadRequest, "invalid security event ID")
		}

		result := app.Dao().DB().Unscoped().Delete(&entity.SecurityEvent{}, id)
		if result.Error != nil {
			return common.RespError(c, fiber.StatusInternalServerError, "security event deletion failed")
		}
		if result.RowsAffected == 0 {
			return common.RespError(c, fiber.StatusNotFound, "security event not found")
		}
		return common.RespSuccess(c)
	}))
}

// SecurityEventClear 清空安全事件记录
//
// @Id           SecurityEventClear
// @Summary      清空安全事件记录
// @Tags         Security
// @Security     ApiKeyAuth
// @Produce      json
// @Success      200  {object}  common.ResponseSuccess
// @Router       /security/events [delete]
func SecurityEventClear(app *core.App, router fiber.Router) {
	router.Delete("/security/events", common.AdminGuard(app, func(c *fiber.Ctx) error {
		var p struct {
			IP   string `query:"ip" json:"ip" validate:"optional"`
			Kind string `query:"kind" json:"kind" validate:"optional"`
			Days int    `query:"days" json:"days" validate:"optional"`
		}
		if isOK, resp := common.ParamsDecode(c, &p); !isOK {
			return resp
		}

		q := securityEventQuery(app, ParamsSecurityEventList{IP: p.IP, Kind: p.Kind, Days: p.Days})
		if err := q.Unscoped().Where("1 = 1").Delete(&entity.SecurityEvent{}).Error; err != nil {
			return common.RespError(c, fiber.StatusInternalServerError, "security event cleanup failed")
		}
		return common.RespSuccess(c)
	}))
}
