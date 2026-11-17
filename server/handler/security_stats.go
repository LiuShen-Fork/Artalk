package handler

import (
	"github.com/artalkjs/artalk/v2/internal/core"
	"github.com/artalkjs/artalk/v2/server/common"
	"github.com/gofiber/fiber/v2"
)

// SecurityStatsGet 实时安全统计（当前窗口计数、生效中的封禁）
//
// @Id           SecurityStatsGet
// @Summary      获取实时安全统计
// @Tags         Security
// @Security     ApiKeyAuth
// @Produce      json
// @Param        ip  query  string  false  "附加查询某个 IP 的状态"
// @Success      200  {object}  core.SecurityStats
// @Router       /security/stats [get]
func SecurityStatsGet(app *core.App, router fiber.Router) {
	router.Get("/security/stats", common.AdminGuard(app, func(c *fiber.Ctx) error {
		var p struct {
			IP string `query:"ip" json:"ip" validate:"optional"`
		}
		if isOK, resp := common.ParamsDecode(c, &p); !isOK {
			return resp
		}

		securityService, err := core.AppService[*core.SecurityService](app)
		if err != nil || securityService == nil {
			return common.RespError(c, fiber.StatusServiceUnavailable, "security service is not available")
		}

		return common.RespData(c, securityService.Stats(p.IP))
	}))
}
