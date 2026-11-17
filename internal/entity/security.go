package entity

import (
	"time"

	"gorm.io/gorm"
)

// 安全事件的类型
type SecurityEventKind string

const (
	SecurityEventKindRateLimit     SecurityEventKind = "rate_limit"     // 超出频率限制
	SecurityEventKindCaptchaFailed SecurityEventKind = "captcha_failed" // 验证码校验失败
	SecurityEventKindBanned        SecurityEventKind = "banned"         // 命中黑名单 / 封禁访问被拦截
	SecurityEventKindAutoBan       SecurityEventKind = "auto_ban"       // 自动封禁
)

// 限频作用的范围
const (
	SecurityScopeIP     = "ip"
	SecurityScopeGlobal = "global"
)

// IP 封禁的来源
const (
	IPBanSourceAuto   = "auto"   // 超限自动封禁
	IPBanSourceManual = "manual" // 管理员手动封禁
	IPBanSourceConfig = "config" // 配置文件中的黑名单
)

// SecurityEvent 安全事件记录（限频、验证码失败、封禁命中）
//
// 仅记录「异常」请求，用于攻击报表和封禁取证；正常请求不落库。
type SecurityEvent struct {
	gorm.Model

	IP       string `gorm:"index;size:64" json:"ip"`         // 客户端 IP
	Kind     string `gorm:"index;size:32" json:"kind"`       // 事件类型
	Scope    string `gorm:"size:16" json:"scope"`            // ip | global
	Window   string `gorm:"size:16" json:"window"`           // minute | hour | day
	Action   string `gorm:"size:16" json:"action"`           // captcha | reject | ban
	Path     string `gorm:"size:255" json:"path"`            // 请求路径
	Method   string `gorm:"size:8" json:"method"`            // 请求方法
	SiteName string `gorm:"index;size:255" json:"site_name"` // 站点
	UserID   uint   `gorm:"index" json:"user_id"`
	UserName string `gorm:"size:255" json:"user_name"`
	UA       string `json:"ua"`
	Detail   string `gorm:"type:text" json:"detail"`  // 附加信息（当前计数等，JSON 文本）
	Blocked  bool   `json:"blocked"`                  // 本次请求是否被拦截
}

type CookedSecurityEvent struct {
	ID       uint   `json:"id"`
	IP       string `json:"ip"`
	IPRegion string `json:"ip_region"`
	Kind     string `json:"kind"`
	Scope    string `json:"scope"`
	Window   string `json:"window"`
	Action   string `json:"action"`
	Path     string `json:"path"`
	Method   string `json:"method"`
	SiteName string `json:"site_name"`
	UserID   uint   `json:"user_id"`
	UserName string `json:"user_name"`
	UA       string `json:"ua"`
	Detail   string `json:"detail"`
	Blocked  bool   `json:"blocked"`
	Date     string `json:"date"`
}

// IPBan IP 封禁记录（自动封禁与手动封禁共用一张表）
type IPBan struct {
	gorm.Model

	IP     string `gorm:"uniqueIndex;size:64" json:"ip"` // 单个 IP 或 CIDR 网段
	IsCIDR bool   `json:"is_cidr"`                       // 是否为 CIDR 网段
	Reason string `gorm:"size:255" json:"reason"`        // 封禁原因
	Note   string `json:"note"`                          // 备注

	Source    string `gorm:"size:16" json:"source"` // auto | manual | config
	CreatedBy uint   `json:"created_by"`            // 手动封禁的操作人

	ExpireAt  *time.Time `gorm:"index" json:"expire_at"` // nil = 永久封禁
	HitCount  uint64     `json:"hit_count"`              // 命中次数
	LastHitAt *time.Time `json:"last_hit_at"`            // 最近一次命中时间
}

func (b *IPBan) IsActive(now time.Time) bool {
	return b.ExpireAt == nil || b.ExpireAt.After(now)
}

type CookedIPBan struct {
	ID        uint   `json:"id"`
	IP        string `json:"ip"`
	IPRegion  string `json:"ip_region"`
	IsCIDR    bool   `json:"is_cidr"`
	Reason    string `json:"reason"`
	Note      string `json:"note"`
	Source    string `json:"source"`
	CreatedBy uint   `json:"created_by"`
	Active    bool   `json:"active"`
	ExpireAt  string `json:"expire_at"`
	HitCount  uint64 `json:"hit_count"`
	LastHitAt string `json:"last_hit_at"`
	CreatedAt string `json:"created_at"`
}
