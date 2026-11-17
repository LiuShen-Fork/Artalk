package core

import (
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/artalkjs/artalk/v2/internal/config"
	"github.com/artalkjs/artalk/v2/internal/entity"
	"github.com/artalkjs/artalk/v2/internal/log"
	"github.com/artalkjs/artalk/v2/internal/security"
	"gorm.io/gorm"
)

var _ Service = (*SecurityService)(nil)

const (
	// 同一 IP 同一类事件的最短落库间隔，避免攻击时把事件表刷爆
	securityEventThrottle = time.Minute

	// 过期封禁记录在数据库中的保留时间，过期后自动清理
	securityBanKeepExpired = 7 * 24 * time.Hour
)

// SecurityService 负责限频计数、IP 封禁与安全事件记录
type SecurityService struct {
	app *App

	counter    *security.Counter
	bans       *security.BanStore
	whitelist  *security.IPMatcher
	violations *security.ViolationTracker

	mu        sync.Mutex
	lastEvent map[string]time.Time

	stopCh   chan struct{}
	stopOnce sync.Once
}

func NewSecurityService(app *App) *SecurityService {
	return &SecurityService{
		app:        app,
		counter:    security.NewCounter(),
		bans:       security.NewBanStore(),
		whitelist:  security.NewIPMatcher(),
		violations: security.NewViolationTracker(),
		lastEvent:  map[string]time.Time{},
	}
}

func (s *SecurityService) Init() error {
	s.stopOnce = sync.Once{}
	s.stopCh = make(chan struct{})

	s.ReloadBans()
	s.prune(time.Now())

	go s.loop()

	return nil
}

func (s *SecurityService) Dispose() error {
	s.stopOnce.Do(func() {
		if s.stopCh != nil {
			close(s.stopCh)
		}
	})
	return nil
}

func (s *SecurityService) loop() {
	ticker := time.NewTicker(time.Hour)
	defer ticker.Stop()

	for {
		select {
		case <-s.stopCh:
			return
		case now := <-ticker.C:
			s.ReloadBans()
			s.prune(now)
		}
	}
}

// -------------------------------------------------------------------
//  查询
// -------------------------------------------------------------------

// IsWhitelisted 白名单中的 IP 不计数、不拦截
func (s *SecurityService) IsWhitelisted(ip string) bool {
	if s == nil {
		return false
	}
	return s.whitelist.Contains(ip)
}

// BanMatch 判断 IP 是否命中封禁
func (s *SecurityService) BanMatch(ip string) (entity.IPBan, bool) {
	if s == nil {
		return entity.IPBan{}, false
	}
	return s.bans.Match(ip, time.Now())
}

// Snapshot 某 IP 当前的窗口计数
func (s *SecurityService) Snapshot(ip string) security.Snapshot {
	if s == nil {
		return security.Snapshot{}
	}
	return s.counter.Snapshot(ip, time.Now())
}

// Stats 实时统计信息
func (s *SecurityService) Stats(ip string) SecurityStats {
	if s == nil {
		return SecurityStats{}
	}
	now := time.Now()
	stats := SecurityStats{
		TrackedIPs: s.counter.Len(),
		ActiveBans: s.bans.Len(),
		Whitelist:  s.whitelist.Len(),
		Global:     s.counter.GlobalSnapshot(now),
		Violations: s.violations.Len(),
	}
	if ip != "" {
		stats.IP = &SecurityIPStats{
			IP:           ip,
			Snapshot:     s.counter.Snapshot(ip, now),
			Whitelisted:  s.whitelist.Contains(ip),
			ViolationNum: s.violations.Count(ip, time.Hour, now),
		}
		if ban, ok := s.bans.Match(ip, now); ok {
			stats.IP.Ban = &ban
		}
	}
	return stats
}

type SecurityStats struct {
	TrackedIPs int                `json:"tracked_ips"` // 正在统计的 IP 数
	ActiveBans int                `json:"active_bans"` // 生效中的封禁数
	Whitelist  int                `json:"whitelist"`   // 白名单条目数
	Violations int                `json:"violations"`  // 正在跟踪违规次数的 IP 数
	Global     security.Snapshot  `json:"global"`
	IP         *SecurityIPStats   `json:"ip,omitempty"`
}

type SecurityIPStats struct {
	IP           string            `json:"ip"`
	Snapshot     security.Snapshot `json:"snapshot"`
	Whitelisted  bool              `json:"whitelisted"`
	ViolationNum int               `json:"violations_last_hour"`
	Ban          *entity.IPBan     `json:"ban,omitempty"`
}

// -------------------------------------------------------------------
//  检查
// -------------------------------------------------------------------

type SecurityCheckInput struct {
	IP       string
	Method   string
	Path     string
	UA       string
	SiteName string
	UserID   uint
	UserName string
}

type SecurityDecision struct {
	Allowed    bool
	Action     string // captcha | reject | ban
	Trip       security.Trip
	RetryAfter int
	Ban        entity.IPBan
}

// Check 计入一次操作并给出处置结果
//
// 只有被判定为超限的请求才会记录安全事件；正常请求不落库。
func (s *SecurityService) Check(input SecurityCheckInput) SecurityDecision {
	allowed := SecurityDecision{Allowed: true}
	if s == nil || input.IP == "" || s.app == nil {
		return allowed
	}

	conf := s.app.Conf()
	if conf == nil || !conf.Security.Enabled {
		return allowed
	}

	now := time.Now()
	trip, ok := s.counter.Allow(input.IP, now, conf.Security.RateLimit.IP, conf.Security.RateLimit.Global)
	if ok {
		return allowed
	}

	action := conf.Security.ActionType()
	if action == config.SecurityActionBan && trip.Scope != security.ScopeIP {
		// 全局超限无法归咎于单个 IP，退化为直接拒绝
		action = config.SecurityActionReject
	}

	decision := SecurityDecision{Action: action, Trip: trip, RetryAfter: trip.RetryAfter(now)}

	switch action {
	case config.SecurityActionBan:
		ban, err := s.ban(input.IP, fmt.Sprintf("rate limit exceeded (%s/%s: %d > %d)", trip.Scope, trip.Window, trip.Count, trip.Limit), "", entity.IPBanSourceAuto, 0, 0, now)
		if err != nil {
			log.Errorf("[Security] auto ban %s failed: %v", input.IP, err)
			decision.Action = config.SecurityActionReject
		} else {
			decision.Ban = ban
		}
	default:
		s.maybeAutoBan(input.IP, now)
	}

	s.recordEvent(entity.SecurityEvent{
		IP:       input.IP,
		Kind:     string(entity.SecurityEventKindRateLimit),
		Scope:    trip.Scope,
		Window:   trip.Window,
		Action:   decision.Action,
		Path:     input.Path,
		Method:   input.Method,
		SiteName: input.SiteName,
		UserID:   input.UserID,
		UserName: input.UserName,
		UA:       input.UA,
		Detail:   mustMarshalDetail(map[string]any{"count": trip.Count, "limit": trip.Limit}),
		Blocked:  true,
	}, now)

	return decision
}

// MarkVerified 验证码通过：清空该 IP 的计数，让它重新开始
func (s *SecurityService) MarkVerified(ip string) {
	if s == nil || ip == "" {
		return
	}
	s.counter.ResetIP(ip)
	s.violations.Reset(ip)
}

// RecordCaptchaFailure 记录一次验证码失败，必要时自动封禁；返回是否已封禁
func (s *SecurityService) RecordCaptchaFailure(input SecurityCheckInput) bool {
	if s == nil || input.IP == "" || s.app == nil {
		return false
	}
	conf := s.app.Conf()
	if conf == nil || !conf.Security.Enabled {
		return false
	}

	now := time.Now()
	s.recordEvent(entity.SecurityEvent{
		IP:       input.IP,
		Kind:     string(entity.SecurityEventKindCaptchaFailed),
		Action:   conf.Security.ActionType(),
		Path:     input.Path,
		Method:   input.Method,
		SiteName: input.SiteName,
		UserID:   input.UserID,
		UserName: input.UserName,
		UA:       input.UA,
		Blocked:  true,
	}, now)

	return s.maybeAutoBan(input.IP, now)
}

// RecordBanHit 记录一次「已封禁 IP 再次访问」，命中次数做节流更新
func (s *SecurityService) RecordBanHit(input SecurityCheckInput, ban entity.IPBan) {
	if s == nil || input.IP == "" || s.app == nil || s.app.dao == nil {
		return
	}

	now := time.Now()
	if !s.shouldRecord("ban-hit|"+input.IP, now) {
		return
	}

	if ban.ID > 0 {
		// 用原生表达式累加，避免并发场景下的读改写丢失
		err := s.app.dao.DB().Model(&entity.IPBan{}).Where("id = ?", ban.ID).
			Updates(map[string]any{"hit_count": gorm.Expr("hit_count + 1"), "last_hit_at": now}).Error
		if err != nil {
			log.Errorf("[Security] update ban hit count failed: %v", err)
		}
	}

	s.recordEvent(entity.SecurityEvent{
		IP:       input.IP,
		Kind:     string(entity.SecurityEventKindBanned),
		Action:   config.SecurityActionReject,
		Path:     input.Path,
		Method:   input.Method,
		SiteName: input.SiteName,
		UserID:   input.UserID,
		UserName: input.UserName,
		UA:       input.UA,
		Detail:   mustMarshalDetail(map[string]any{"ban_id": ban.ID, "ban_reason": ban.Reason, "ban_expire_at": ban.ExpireAt}),
		Blocked:  true,
	}, now)
}

// -------------------------------------------------------------------
//  封禁管理
// -------------------------------------------------------------------

// BanIP 手动封禁，duration 为秒（-1 永久，0 使用自动封禁的默认时长，仅自动封禁时生效）
func (s *SecurityService) BanIP(ip, reason, note string, duration int, operatorID uint, source string) (entity.IPBan, error) {
	if _, _, _, err := security.ParseBanTarget(ip); err != nil {
		return entity.IPBan{}, err
	}
	return s.ban(ip, reason, note, source, duration, operatorID, time.Now())
}

// Unban 解除封禁
func (s *SecurityService) Unban(id uint) error {
	if s == nil || s.app == nil || s.app.dao == nil {
		return errors.New("security service is not available")
	}
	result := s.app.dao.DB().Unscoped().Delete(&entity.IPBan{}, id)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return errors.New("ip ban not found")
	}
	s.ReloadBans()
	return nil
}

// ReloadBans 重新加载数据库封禁记录与配置中的黑白名单
func (s *SecurityService) ReloadBans() {
	if s == nil || s.app == nil {
		return
	}
	conf := s.app.Conf()
	if conf == nil {
		return
	}

	now := time.Now()
	rows := []entity.IPBan{}
	if s.app.dao != nil {
		if err := s.app.dao.DB().Where("expire_at IS NULL OR expire_at > ?", now).Find(&rows).Error; err != nil {
			log.Errorf("[Security] load ip bans failed: %v", err)
		}
	}
	for _, target := range conf.Security.Blacklist {
		rows = append(rows, entity.IPBan{IP: target, Source: entity.IPBanSourceConfig, Reason: "config blacklist"})
	}

	s.bans.ReplaceAll(rows, now)
	s.whitelist.ReplaceAll(conf.Security.Whitelist)
}

// ApplyConfig 配置变更后调用：重载名单并清空计数
func (s *SecurityService) ApplyConfig() {
	if s == nil {
		return
	}
	s.counter.Reset()
	s.violations = security.NewViolationTracker()
	s.ReloadBans()
}

func (s *SecurityService) ban(ip, reason, note, source string, duration int, operatorID uint, now time.Time) (entity.IPBan, error) {
	if s == nil || s.app == nil || s.app.dao == nil {
		return entity.IPBan{}, errors.New("security service is not available")
	}
	if source == "" {
		source = entity.IPBanSourceManual
	}

	autoBan := s.app.Conf().Security.AutoBan.Normalized()
	if duration == 0 {
		duration = autoBan.Duration
	}

	ban := entity.IPBan{IP: ip, Reason: reason, Note: note, Source: source, CreatedBy: operatorID}
	if duration > 0 {
		expireAt := now.Add(time.Duration(duration) * time.Second)
		ban.ExpireAt = &expireAt
	}

	var existing entity.IPBan
	err := s.app.dao.DB().Where("ip = ?", ip).First(&existing).Error
	switch {
	case err == nil:
		existing.Reason = reason
		existing.Note = note
		existing.Source = source
		existing.ExpireAt = ban.ExpireAt
		if operatorID > 0 {
			existing.CreatedBy = operatorID
		}
		if saveErr := s.app.dao.DB().Unscoped().Save(&existing).Error; saveErr != nil {
			return entity.IPBan{}, saveErr
		}
		ban = existing
	case errors.Is(err, gorm.ErrRecordNotFound):
		if createErr := s.app.dao.DB().Create(&ban).Error; createErr != nil {
			return entity.IPBan{}, createErr
		}
	default:
		return entity.IPBan{}, err
	}

	s.violations.Reset(ip)
	s.ReloadBans()

	return ban, nil
}

func (s *SecurityService) maybeAutoBan(ip string, now time.Time) bool {
	conf := s.app.Conf()
	if conf == nil || !conf.Security.AutoBan.Enabled {
		return false
	}

	autoBan := conf.Security.AutoBan.Normalized()
	count := s.violations.Add(ip, now, time.Duration(autoBan.Window)*time.Second)
	if count < autoBan.Threshold {
		return false
	}

	reason := fmt.Sprintf("%d violations within %ds", count, autoBan.Window)
	ban, err := s.ban(ip, reason, "", entity.IPBanSourceAuto, autoBan.Duration, 0, now)
	if err != nil {
		log.Errorf("[Security] auto ban %s failed: %v", ip, err)
		return false
	}

	s.recordEvent(entity.SecurityEvent{
		IP:      ip,
		Kind:    string(entity.SecurityEventKindAutoBan),
		Action:  config.SecurityActionBan,
		Detail:  mustMarshalDetail(map[string]any{"violations": count, "window": autoBan.Window, "duration": autoBan.Duration, "ban_id": ban.ID}),
		Blocked: true,
	}, now)

	return true
}

// -------------------------------------------------------------------
//  内部工具
// -------------------------------------------------------------------

func (s *SecurityService) recordEvent(evt entity.SecurityEvent, now time.Time) {
	if s == nil || s.app == nil || s.app.dao == nil {
		return
	}

	key := evt.Kind + "|" + evt.IP + "|" + evt.Scope + "|" + evt.Window
	if !s.shouldRecord(key, now) {
		return
	}

	if err := s.app.dao.DB().Create(&evt).Error; err != nil {
		log.Errorf("[Security] record security event failed: %v", err)
	}
}

func (s *SecurityService) shouldRecord(key string, now time.Time) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	if last, ok := s.lastEvent[key]; ok && now.Sub(last) < securityEventThrottle {
		return false
	}
	if len(s.lastEvent) > 10000 {
		for k, t := range s.lastEvent {
			if now.Sub(t) >= securityEventThrottle {
				delete(s.lastEvent, k)
			}
		}
	}
	s.lastEvent[key] = now
	return true
}

func (s *SecurityService) prune(now time.Time) {
	if s == nil || s.app == nil || s.app.dao == nil {
		return
	}

	conf := s.app.Conf()
	if conf == nil {
		return
	}

	if days := conf.Security.EventRetentionDays; days > 0 {
		cutoff := now.AddDate(0, 0, -days)
		if err := s.app.dao.DB().Unscoped().Where("created_at < ?", cutoff).Delete(&entity.SecurityEvent{}).Error; err != nil {
			log.Errorf("[Security] prune security events failed: %v", err)
		}
	}

	if err := s.app.dao.DB().Unscoped().Where("expire_at IS NOT NULL AND expire_at < ?", now.Add(-securityBanKeepExpired)).
		Delete(&entity.IPBan{}).Error; err != nil {
		log.Errorf("[Security] prune ip bans failed: %v", err)
	}
}

func mustMarshalDetail(v any) string {
	b, err := json.Marshal(v)
	if err != nil {
		return ""
	}
	return string(b)
}
