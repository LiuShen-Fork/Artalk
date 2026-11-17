package handler

import (
	"strings"
	"time"

	"github.com/artalkjs/artalk/v2/internal/core"
	"github.com/artalkjs/artalk/v2/internal/entity"
	"github.com/artalkjs/artalk/v2/internal/security"
	"github.com/artalkjs/artalk/v2/server/common"
	"github.com/gofiber/fiber/v2"
)

type ParamsSecurityBanList struct {
	IP     string `query:"ip" json:"ip" validate:"optional"`
	Active string `query:"active" json:"active" validate:"optional"` // true / false，空 = 全部
	Limit  int    `query:"limit" json:"limit" validate:"optional"`
	Offset int    `query:"offset" json:"offset" validate:"optional"`
}

type ResponseSecurityBanList struct {
	Total int64                `json:"count"`
	Bans  []entity.CookedIPBan `json:"bans"`
}

type ParamsSecurityBanCreate struct {
	IP       string `json:"ip" validate:"required"`
	Reason   string `json:"reason" validate:"optional"`
	Note     string `json:"note" validate:"optional"`
	Duration int    `json:"duration" validate:"optional"` // 秒；<= 0 表示永久
}

// SecurityBanList 获取 IP 封禁列表
//
// 包含自动封禁、手动封禁以及配置文件中黑名单的条目（配置条目 id 为 0）。
//
// @Id           SecurityBanList
// @Summary      获取 IP 封禁列表
// @Tags         Security
// @Security     ApiKeyAuth
// @Produce      json
// @Success      200  {object}  ResponseSecurityBanList
// @Router       /security/bans [get]
func SecurityBanList(app *core.App, router fiber.Router) {
	router.Get("/security/bans", common.AdminGuard(app, func(c *fiber.Ctx) error {
		var p ParamsSecurityBanList
		if isOK, resp := common.ParamsDecode(c, &p); !isOK {
			return resp
		}

		now := time.Now()
		q := app.Dao().DB().Model(&entity.IPBan{})
		if ip := strings.TrimSpace(p.IP); ip != "" {
			q = q.Where("ip = ?", ip)
		}
		if p.Active == "true" {
			q = q.Where("expire_at IS NULL OR expire_at > ?", now)
		} else if p.Active == "false" {
			q = q.Where("expire_at IS NOT NULL AND expire_at <= ?", now)
		}

		var total int64
		q.Count(&total)

		var bans []entity.IPBan
		q.Order("created_at DESC").Scopes(Paginate(p.Offset, p.Limit)).Find(&bans)

		items := make([]entity.CookedIPBan, 0, len(bans))
		for _, ban := range bans {
			items = append(items, cookIPBan(app, ban, now))
		}

		// 配置文件中的黑名单并不是数据库记录，这里作为只读条目一并返回，方便回看
		if p.Active != "false" {
			for _, target := range app.Conf().Security.Blacklist {
				if p.IP != "" && !strings.EqualFold(p.IP, strings.TrimSpace(target)) {
					continue
				}
				items = append(items, cookIPBan(app, entity.IPBan{
					IP:     strings.TrimSpace(target),
					Reason: "config blacklist",
					Source: entity.IPBanSourceConfig,
				}, now))
			}
		}

		return common.RespData(c, ResponseSecurityBanList{Total: total, Bans: items})
	}))
}

// SecurityBanCreate 手动封禁 IP
//
// @Id           SecurityBanCreate
// @Summary      封禁 IP 或网段
// @Tags         Security
// @Security     ApiKeyAuth
// @Accept       json
// @Produce      json
// @Param        data  body  ParamsSecurityBanCreate  true  "封禁信息"
// @Success      200  {object}  entity.CookedIPBan
// @Router       /security/bans [post]
func SecurityBanCreate(app *core.App, router fiber.Router) {
	router.Post("/security/bans", common.AdminGuard(app, func(c *fiber.Ctx) error {
		var p ParamsSecurityBanCreate
		if isOK, resp := common.ParamsDecode(c, &p); !isOK {
			return resp
		}

		target, _, _, err := security.ParseBanTarget(strings.TrimSpace(p.IP))
		if err != nil {
			return common.RespError(c, fiber.StatusBadRequest, errInvalidIPTarget.Error())
		}

		securityService, err := core.AppService[*core.SecurityService](app)
		if err != nil || securityService == nil {
			return common.RespError(c, fiber.StatusServiceUnavailable, "security service is not available")
		}

		duration := p.Duration
		if duration <= 0 {
			duration = -1 // 永久
		}

		reason := strings.TrimSpace(p.Reason)
		if reason == "" {
			reason = "manual ban"
		}

		operatorID := uint(0)
		if user, userErr := common.GetUserByReq(app, c); userErr == nil && !user.IsEmpty() {
			operatorID = user.ID
		}

		ban, err := securityService.BanIP(target, reason, strings.TrimSpace(p.Note), duration, operatorID, entity.IPBanSourceManual)
		if err != nil {
			return common.RespError(c, fiber.StatusInternalServerError, "create ip ban failed: "+err.Error())
		}

		return common.RespData(c, cookIPBan(app, ban, time.Now()))
	}))
}

// SecurityBanDelete 解除封禁
//
// @Id           SecurityBanDelete
// @Summary      解除 IP 封禁
// @Tags         Security
// @Security     ApiKeyAuth
// @Produce      json
// @Success      200  {object}  common.ResponseSuccess
// @Router       /security/bans/{id} [delete]
func SecurityBanDelete(app *core.App, router fiber.Router) {
	router.Delete("/security/bans/:id", common.AdminGuard(app, func(c *fiber.Ctx) error {
		id, err := c.ParamsInt("id")
		if err != nil || id < 1 {
			return common.RespError(c, fiber.StatusBadRequest, "invalid ip ban ID (config blacklist entries cannot be deleted here)")
		}

		securityService, err := core.AppService[*core.SecurityService](app)
		if err != nil || securityService == nil {
			return common.RespError(c, fiber.StatusServiceUnavailable, "security service is not available")
		}
		if err := securityService.Unban(uint(id)); err != nil {
			return common.RespError(c, fiber.StatusNotFound, err.Error())
		}

		return common.RespSuccess(c)
	}))
}

func cookIPBan(app *core.App, ban entity.IPBan, now time.Time) entity.CookedIPBan {
	item := entity.CookedIPBan{
		ID:        ban.ID,
		IP:        ban.IP,
		IsCIDR:    strings.Contains(ban.IP, "/"),
		Reason:    ban.Reason,
		Note:      ban.Note,
		Source:    ban.Source,
		CreatedBy: ban.CreatedBy,
		Active:    ban.IsActive(now),
		HitCount:  ban.HitCount,
	}
	if ban.ExpireAt != nil {
		item.ExpireAt = ban.ExpireAt.Format("2006-01-02 15:04:05")
	}
	if ban.LastHitAt != nil {
		item.LastHitAt = ban.LastHitAt.Format("2006-01-02 15:04:05")
	}
	if !ban.CreatedAt.IsZero() {
		item.CreatedAt = ban.CreatedAt.Format("2006-01-02 15:04:05")
	}
	if ipRegion, err := core.AppService[*core.IPRegionService](app); err == nil && ipRegion != nil && !item.IsCIDR {
		item.IPRegion = ipRegion.Query(ban.IP)
	}
	return item
}
