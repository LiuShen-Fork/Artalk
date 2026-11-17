package config

// 安全防护配置
//
// 频率限制、自动封禁、IP 黑名单/白名单。所有配额为 `0` 时表示不限制，
// 因此 `security.enabled: true` 但未填写任何配额时不会拦截任何请求。
type SecurityConf struct {
	Enabled bool `koanf:"enabled" json:"enabled"` // 安全防护总开关

	// 超出频率限制后的动作: "captcha" | "reject" | "ban"
	// captcha: 要求通过验证码 (与 captcha 配置联动)
	// reject: 直接拒绝 (HTTP 429)
	// ban: 自动封禁该 IP
	Action string `koanf:"action" json:"action"`

	RateLimit SecurityRateLimitConf `koanf:"rate_limit" json:"rate_limit"` // 频率限制
	AutoBan   SecurityAutoBanConf   `koanf:"auto_ban" json:"auto_ban"`     // 自动封禁

	Blacklist []string `koanf:"blacklist" json:"blacklist"` // IP 黑名单 (支持 CIDR)
	Whitelist []string `koanf:"whitelist" json:"whitelist"` // IP 白名单 (支持 CIDR，优先于黑名单)

	// 事件记录保留天数，0 表示永久保留
	EventRetentionDays int `koanf:"event_retention_days" json:"event_retention_days"`

	// 不参与计数的路径前缀 (默认已忽略 /captcha)
	ExcludePaths []string `koanf:"exclude_paths" json:"exclude_paths"`
}

type SecurityRateLimitConf struct {
	IP     SecurityQuotaConf `koanf:"ip" json:"ip"`         // 单 IP 限频
	Global SecurityQuotaConf `koanf:"global" json:"global"` // 全局限频
}

// 时间窗口配额，`0` 表示该窗口不限制
type SecurityQuotaConf struct {
	Minute int `koanf:"minute" json:"minute"` // 每分钟允许次数
	Hour   int `koanf:"hour" json:"hour"`     // 每小时允许次数
	Day    int `koanf:"day" json:"day"`       // 每天允许次数
}

func (q SecurityQuotaConf) IsUnlimited() bool {
	return q.Minute <= 0 && q.Hour <= 0 && q.Day <= 0
}

type SecurityAutoBanConf struct {
	Enabled bool `koanf:"enabled" json:"enabled"`

	// 在 Window 秒内触发 Threshold 次超限后自动封禁
	Threshold int `koanf:"threshold" json:"threshold"` // 0 = 默认 5
	Window    int `koanf:"window" json:"window"`       // 单位秒，0 = 默认 3600
	Duration  int `koanf:"duration" json:"duration"`   // 封禁时长（秒），-1 = 永久，0 = 默认 86400
}

const (
	SecurityActionCaptcha = "captcha"
	SecurityActionReject  = "reject"
	SecurityActionBan     = "ban"

	DefaultSecurityAutoBanThreshold = 5
	DefaultSecurityAutoBanWindow    = 3600
	DefaultSecurityAutoBanDuration  = 86400

	DefaultSecurityEventRetentionDays = 90
)

// 不参与安全计数与封禁拦截的路径（验证码校验本身必须放行，否则无法自救）
var defaultSecurityExcludePaths = []string{"/captcha/verify"}

// 实际的排除路径列表：默认值 + 用户配置
func (c *SecurityConf) ExcludedPaths() []string {
	paths := make([]string, 0, len(defaultSecurityExcludePaths)+len(c.ExcludePaths))
	paths = append(paths, defaultSecurityExcludePaths...)
	paths = append(paths, c.ExcludePaths...)
	return paths
}

// 触发限频后的动作，未配置时默认要求验证码
func (c *SecurityConf) ActionType() string {
	switch c.Action {
	case SecurityActionReject, SecurityActionBan, SecurityActionCaptcha:
		return c.Action
	default:
		return SecurityActionCaptcha
	}
}

// 补齐自动封禁的默认值，避免 `threshold: 0` 变成「立刻封禁」
func (a *SecurityAutoBanConf) Normalized() SecurityAutoBanConf {
	out := *a
	if out.Threshold <= 0 {
		out.Threshold = DefaultSecurityAutoBanThreshold
	}
	if out.Window <= 0 {
		out.Window = DefaultSecurityAutoBanWindow
	}
	if out.Duration == 0 {
		out.Duration = DefaultSecurityAutoBanDuration
	}
	return out
}
