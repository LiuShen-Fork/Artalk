// Package security 提供限频、IP 封禁与验证码挑战的 HTTP 中间件
package security

import (
	"strconv"
	"strings"

	"github.com/artalkjs/artalk/v2/internal/captcha"
	"github.com/artalkjs/artalk/v2/internal/config"
	"github.com/artalkjs/artalk/v2/internal/core"
	"github.com/artalkjs/artalk/v2/internal/i18n"
	"github.com/artalkjs/artalk/v2/server/common"
	"github.com/gofiber/fiber/v2"
)

// Guard 安全防护中间件
//
// 职责：
//  1. 命中 IP 黑名单 / 封禁记录时直接拒绝；
//  2. 对「写操作」按分钟 / 小时 / 天三个窗口计数，超限后按配置要求验证码、拒绝或封禁。
//
// 读请求（GET / HEAD / OPTIONS）不参与计数，避免正常的列表轮询触发限频；
// 管理员与白名单 IP 完全放行；无法确定来源 IP 的请求不计入限制。
func Guard(app *core.App) fiber.Handler {
	return func(c *fiber.Ctx) error {
		conf := app.Conf()
		if conf == nil || !conf.Security.Enabled {
			return c.Next()
		}

		switch c.Method() {
		case fiber.MethodGet, fiber.MethodHead, fiber.MethodOptions:
			return c.Next()
		}

		if isExcludedPath(conf.Security.ExcludedPaths(), c.Path()) {
			return c.Next()
		}

		// 管理员不受安全策略限制
		if common.CheckIsAdminReq(app, c) {
			return c.Next()
		}

		svc, err := core.AppService[*core.SecurityService](app)
		if err != nil || svc == nil {
			return c.Next() // 安全服务不可用时不阻断业务
		}

		input := newCheckInput(c)
		if input.IP == "" {
			// 无法确定来源（反向代理未传 IP），不做限制，避免误伤全部访客
			return c.Next()
		}

		if svc.IsWhitelisted(input.IP) {
			return c.Next()
		}

		if ban, hit := svc.BanMatch(input.IP); hit {
			svc.RecordBanHit(input, ban)
			return respondBanned(c)
		}

		decision := svc.Check(input)
		if decision.Allowed {
			return c.Next()
		}

		switch decision.Action {
		case config.SecurityActionReject:
			return respondRejected(c, decision.RetryAfter)
		case config.SecurityActionBan:
			return respondBanned(c)
		default:
			return respondCaptcha(app, c)
		}
	}
}

func newCheckInput(c *fiber.Ctx) core.SecurityCheckInput {
	return core.SecurityCheckInput{
		IP:       common.ClientIP(c),
		Method:   c.Method(),
		Path:     c.Path(),
		UA:       c.Get("User-Agent"),
		SiteName: strings.TrimSpace(c.Query("site_name")),
	}
}

// isExcludedPath 路径是否为排除项（按后缀匹配，便于忽略 `/api/v2` 前缀）
func isExcludedPath(excluded []string, path string) bool {
	for _, item := range excluded {
		item = strings.TrimSpace(item)
		if item == "" {
			continue
		}
		if strings.HasSuffix(path, item) || strings.HasPrefix(path, item) {
			return true
		}
	}
	return false
}

// respondCaptcha 与 LimiterGuard 保持一致的响应结构，官方前端无需改动即可弹验证码
func respondCaptcha(app *core.App, c *fiber.Ctx) error {
	cap := common.NewCaptchaChecker(app, c)

	respData := common.Map{"need_captcha": true}
	switch cap.Type() {
	case captcha.Image:
		img, _ := cap.Get()
		respData["img_data"] = string(img)
	case captcha.IFrame:
		respData["iframe"] = true
	}

	return common.RespError(c, 403, i18n.T("Captcha required"), respData)
}

func respondRejected(c *fiber.Ctx, retryAfter int) error {
	if retryAfter < 1 {
		retryAfter = 1
	}
	c.Set(fiber.HeaderRetryAfter, strconv.Itoa(retryAfter))

	return common.RespError(c, fiber.StatusTooManyRequests, i18n.T("Too many requests, please try again later"), common.Map{
		"rate_limited": true,
		"retry_after":  retryAfter,
	})
}

func respondBanned(c *fiber.Ctx) error {
	return common.RespError(c, 403, i18n.T("Access denied"), common.Map{"ip_banned": true})
}
