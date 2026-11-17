package handler

import (
	"strings"

	"github.com/artalkjs/artalk/v2/internal/config"
	"github.com/artalkjs/artalk/v2/internal/core"
	"github.com/artalkjs/artalk/v2/internal/log"
	"github.com/artalkjs/artalk/v2/internal/security"
	"github.com/artalkjs/artalk/v2/server/common"
	"github.com/gofiber/fiber/v2"
)

// -------------------------------------------------------------------
//  数据结构
// -------------------------------------------------------------------

type SecurityGeetestConfig struct {
	CaptchaID  string `json:"captcha_id"`
	CaptchaKey string `json:"captcha_key"`
}

type SecurityCaptchaConfig struct {
	Enabled     bool                  `json:"enabled"`
	Always      bool                  `json:"always"`
	CaptchaType string                `json:"captcha_type"`
	ActionReset int                   `json:"action_reset"`
	ActionLimit int                   `json:"action_limit"`
	Geetest     SecurityGeetestConfig `json:"geetest"`
}

type SecurityRateLimitConfig struct {
	IP     config.SecurityQuotaConf `json:"ip"`
	Global config.SecurityQuotaConf `json:"global"`
}

type SecurityAutoBanConfig struct {
	Enabled   bool `json:"enabled"`
	Threshold int  `json:"threshold"`
	Window    int  `json:"window"`
	Duration  int  `json:"duration"`
}

type SecurityConfigData struct {
	Enabled            bool                    `json:"enabled"`
	Action             string                  `json:"action"`
	RateLimit          SecurityRateLimitConfig `json:"rate_limit"`
	AutoBan            SecurityAutoBanConfig   `json:"auto_ban"`
	Blacklist          []string                `json:"blacklist"`
	Whitelist          []string                `json:"whitelist"`
	EventRetentionDays int                     `json:"event_retention_days"`
	ExcludePaths       []string                `json:"exclude_paths"`
}

type SecurityRuntimeData struct {
	CaptchaReady bool              `json:"captcha_ready"` // 验证码动作是否可用（依赖 captcha 启用）
	ActionGroups []string          `json:"action_groups"`
	TrackedIPs   int               `json:"tracked_ips"`
	ActiveBans   int               `json:"active_bans"`
	Violators    int               `json:"tracked_violators"`
	Global       security.Snapshot `json:"global"`
}

type ResponseSecurityConfig struct {
	Security SecurityConfigData  `json:"security"`
	Captcha  SecurityCaptchaConfig `json:"captcha"`
	Runtime  SecurityRuntimeData `json:"runtime"`
}

type ParamsSecurityConfigSave struct {
	Enabled            *bool                    `json:"enabled" validate:"optional"`
	Action             *string                  `json:"action" validate:"optional"`
	RateLimit          *SecurityRateLimitConfig `json:"rate_limit" validate:"optional"`
	AutoBan            *SecurityAutoBanConfig   `json:"auto_ban" validate:"optional"`
	Blacklist          *[]string                `json:"blacklist" validate:"optional"`
	Whitelist          *[]string                `json:"whitelist" validate:"optional"`
	EventRetentionDays *int                     `json:"event_retention_days" validate:"optional"`
	Captcha            *ParamsSecurityCaptcha   `json:"captcha" validate:"optional"`
}

type ParamsSecurityCaptcha struct {
	Enabled     *bool                  `json:"enabled" validate:"optional"`
	Always      *bool                  `json:"always" validate:"optional"`
	CaptchaType *string                `json:"captcha_type" validate:"optional"`
	ActionReset *int                   `json:"action_reset" validate:"optional"`
	ActionLimit *int                   `json:"action_limit" validate:"optional"`
	Geetest     *SecurityGeetestConfig `json:"geetest" validate:"optional"`
}

var validCaptchaTypes = map[string]bool{
	string(config.TypeImage):     true,
	string(config.TypeTurnstile): true,
	string(config.TypeReCaptcha): true,
	string(config.TypeHCaptcha):  true,
	string(config.TypeGeetest):   true,
	string(config.TypeCapJS):     true,
}

// -------------------------------------------------------------------
//  接口
// -------------------------------------------------------------------

// SecurityConfigGet 读取当前的安全防护与验证码配置
//
// @Id           SecurityConfigGet
// @Summary      获取安全防护配置
// @Tags         Security
// @Security     ApiKeyAuth
// @Produce      json
// @Success      200  {object}  ResponseSecurityConfig
// @Router       /security/config [get]
func SecurityConfigGet(app *core.App, router fiber.Router) {
	router.Get("/security/config", common.AdminGuard(app, func(c *fiber.Ctx) error {
		return common.RespData(c, buildSecurityConfigResponse(app))
	}))
}

// SecurityConfigSave 保存安全防护配置
//
// 只改动 `security` 与 `captcha` 相关字段，配置文件中的注释与其他配置保持不变，
// 保存后立即生效（不重启服务）。
//
// @Id           SecurityConfigSave
// @Summary      保存安全防护配置
// @Tags         Security
// @Security     ApiKeyAuth
// @Accept       json
// @Produce      json
// @Param        data  body  ParamsSecurityConfigSave  true  "配置"
// @Success      200  {object}  ResponseSecurityConfig
// @Router       /security/config [put]
func SecurityConfigSave(app *core.App, router fiber.Router) {
	router.Put("/security/config", common.AdminGuard(app, func(c *fiber.Ctx) error {
		var p ParamsSecurityConfigSave
		if isOK, resp := common.ParamsDecode(c, &p); !isOK {
			return resp
		}

		updates, err := securityConfigUpdates(&p)
		if err != nil {
			return common.RespError(c, fiber.StatusBadRequest, err.Error())
		}

		if len(updates) > 0 {
			newConf, err := writeConfigFile(app, updates)
			if err != nil {
				return common.RespError(c, fiber.StatusInternalServerError, "save security config failed: "+err.Error())
			}
			app.SetConf(newConf)
			log.Info("[Security] config updated")
		}

		// 重载黑白名单并清空计数，使新配置立刻生效
		if securityService, svcErr := core.AppService[*core.SecurityService](app); svcErr == nil && securityService != nil {
			securityService.ApplyConfig()
		}

		return common.RespData(c, buildSecurityConfigResponse(app))
	}))
}

func buildSecurityConfigResponse(app *core.App) ResponseSecurityConfig {
	conf := app.Conf()

	data := SecurityConfigData{
		Enabled:            conf.Security.Enabled,
		Action:             conf.Security.ActionType(),
		EventRetentionDays: conf.Security.EventRetentionDays,
		Blacklist:          nonNilStrings(conf.Security.Blacklist),
		Whitelist:          nonNilStrings(conf.Security.Whitelist),
		ExcludePaths:       nonNilStrings(conf.Security.ExcludePaths),
		RateLimit: SecurityRateLimitConfig{
			IP:     conf.Security.RateLimit.IP,
			Global: conf.Security.RateLimit.Global,
		},
		AutoBan: SecurityAutoBanConfig{
			Enabled:   conf.Security.AutoBan.Enabled,
			Threshold: conf.Security.AutoBan.Threshold,
			Window:    conf.Security.AutoBan.Window,
			Duration:  conf.Security.AutoBan.Duration,
		},
	}

	captcha := SecurityCaptchaConfig{
		Enabled:     conf.Captcha.Enabled,
		Always:      conf.Captcha.Always,
		CaptchaType: string(conf.Captcha.CaptchaType),
		ActionReset: conf.Captcha.ActionReset,
		ActionLimit: conf.Captcha.ActionLimit,
		Geetest: SecurityGeetestConfig{
			CaptchaID:  conf.Captcha.Geetest.CaptchaID,
			CaptchaKey: conf.Captcha.Geetest.CaptchaKey,
		},
	}

	runtime := SecurityRuntimeData{
		CaptchaReady: conf.Captcha.Enabled,
		ActionGroups: []string{config.SecurityActionCaptcha, config.SecurityActionReject, config.SecurityActionBan},
	}
	if securityService, err := core.AppService[*core.SecurityService](app); err == nil && securityService != nil {
		stats := securityService.Stats("")
		runtime.TrackedIPs = stats.TrackedIPs
		runtime.ActiveBans = stats.ActiveBans
		runtime.Violators = stats.Violations
		runtime.Global = stats.Global
	}

	return ResponseSecurityConfig{Security: data, Captcha: captcha, Runtime: runtime}
}

func securityConfigUpdates(p *ParamsSecurityConfigSave) (map[string]any, error) {
	updates := map[string]any{}

	if p.Enabled != nil {
		updates["security.enabled"] = *p.Enabled
	}
	if p.Action != nil {
		action := strings.TrimSpace(*p.Action)
		switch action {
		case config.SecurityActionCaptcha, config.SecurityActionReject, config.SecurityActionBan, "":
			updates["security.action"] = action
		default:
			return nil, errInvalidSecurityAction
		}
	}
	if p.RateLimit != nil {
		quotaUpdates(updates, "security.rate_limit.ip", p.RateLimit.IP)
		quotaUpdates(updates, "security.rate_limit.global", p.RateLimit.Global)
	}
	if p.AutoBan != nil {
		if p.AutoBan.Threshold < 0 || p.AutoBan.Window < 0 || p.AutoBan.Duration < -1 {
			return nil, errInvalidAutoBan
		}
		updates["security.auto_ban.enabled"] = p.AutoBan.Enabled
		updates["security.auto_ban.threshold"] = p.AutoBan.Threshold
		updates["security.auto_ban.window"] = p.AutoBan.Window
		updates["security.auto_ban.duration"] = p.AutoBan.Duration
	}
	if p.Blacklist != nil {
		list, err := normalizeIPList(*p.Blacklist)
		if err != nil {
			return nil, err
		}
		updates["security.blacklist"] = list
	}
	if p.Whitelist != nil {
		list, err := normalizeIPList(*p.Whitelist)
		if err != nil {
			return nil, err
		}
		updates["security.whitelist"] = list
	}
	if p.EventRetentionDays != nil {
		if *p.EventRetentionDays < 0 {
			return nil, errInvalidRetention
		}
		updates["security.event_retention_days"] = *p.EventRetentionDays
	}

	if p.Captcha != nil {
		if p.Captcha.Enabled != nil {
			updates["captcha.enabled"] = *p.Captcha.Enabled
		}
		if p.Captcha.Always != nil {
			updates["captcha.always"] = *p.Captcha.Always
		}
		if p.Captcha.CaptchaType != nil {
			captchaType := strings.TrimSpace(*p.Captcha.CaptchaType)
			if !validCaptchaTypes[captchaType] {
				return nil, errInvalidCaptchaType
			}
			updates["captcha.captcha_type"] = captchaType
		}
		if p.Captcha.ActionReset != nil {
			if *p.Captcha.ActionReset < 0 {
				return nil, errInvalidCaptchaLimit
			}
			updates["captcha.action_reset"] = *p.Captcha.ActionReset
		}
		if p.Captcha.ActionLimit != nil {
			if *p.Captcha.ActionLimit < 0 {
				return nil, errInvalidCaptchaLimit
			}
			updates["captcha.action_limit"] = *p.Captcha.ActionLimit
		}
		if p.Captcha.Geetest != nil {
			updates["captcha.geetest.captcha_id"] = strings.TrimSpace(p.Captcha.Geetest.CaptchaID)
			updates["captcha.geetest.captcha_key"] = strings.TrimSpace(p.Captcha.Geetest.CaptchaKey)
		}
	}

	return updates, nil
}

func quotaUpdates(updates map[string]any, prefix string, quota config.SecurityQuotaConf) {
	// 0 表示该窗口不限制，属于合法值
	updates[prefix+".minute"] = maxInt(quota.Minute, 0)
	updates[prefix+".hour"] = maxInt(quota.Hour, 0)
	updates[prefix+".day"] = maxInt(quota.Day, 0)
}

func normalizeIPList(list []string) ([]string, error) {
	out := make([]string, 0, len(list))
	for _, item := range list {
		item = strings.TrimSpace(item)
		if item == "" {
			continue
		}
		target, _, _, err := security.ParseBanTarget(item)
		if err != nil {
			return nil, errInvalidIPTarget
		}
		out = append(out, target)
	}
	return out, nil
}

func nonNilStrings(list []string) []string {
	if list == nil {
		return []string{}
	}
	return list
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
