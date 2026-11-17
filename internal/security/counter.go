package security

import (
	"sync"
	"time"

	"github.com/artalkjs/artalk/v2/internal/config"
)

// 限频作用范围
const (
	ScopeIP     = "ip"
	ScopeGlobal = "global"
)

// 限频时间窗口
const (
	WindowMinute = "minute"
	WindowHour   = "hour"
	WindowDay    = "day"
)

// Trip 描述某个被超限的窗口
type Trip struct {
	Scope  string
	Window string
	Count  int
	Limit  int
}

// RetryAfter 返回距离该窗口重置的秒数，至少 1 秒
func (t Trip) RetryAfter(now time.Time) int {
	var next time.Time
	switch t.Window {
	case WindowMinute:
		next = now.Truncate(time.Minute).Add(time.Minute)
	case WindowHour:
		next = now.Truncate(time.Hour).Add(time.Hour)
	case WindowDay:
		next = time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location()).AddDate(0, 0, 1)
	default:
		return 1
	}
	sec := int(next.Sub(now).Seconds())
	if sec < 1 {
		return 1
	}
	return sec
}

// Counter 多窗口（分 / 时 / 天）固定窗口计数器
//
// 固定窗口的实现允许在窗口边界处出现最多两倍于配额的突发请求，
// 换来 O(1) 的读写与极低的内存占用，对评论区的刷屏防护已经足够。
type Counter struct {
	mu        sync.Mutex
	ipStats   map[string]*windowStat
	global    windowStat
	lastSweep time.Time
}

type windowStat struct {
	lastSeen time.Time

	minuteStart time.Time
	hourStart   time.Time
	dayStart    time.Time

	minute int
	hour   int
	day    int
}

func newWindowStat(now time.Time) *windowStat {
	st := &windowStat{lastSeen: now}
	st.resetWindows(now)
	return st
}

func (s *windowStat) roll(now time.Time) {
	if !s.minuteStart.Equal(now.Truncate(time.Minute)) {
		s.minuteStart = now.Truncate(time.Minute)
		s.minute = 0
	}
	if !s.hourStart.Equal(now.Truncate(time.Hour)) {
		s.hourStart = now.Truncate(time.Hour)
		s.hour = 0
	}
	dayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	if !s.dayStart.Equal(dayStart) {
		s.dayStart = dayStart
		s.day = 0
	}
}

func (s *windowStat) resetWindows(now time.Time) {
	s.minuteStart = now.Truncate(time.Minute)
	s.hourStart = now.Truncate(time.Hour)
	s.dayStart = time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	s.minute, s.hour, s.day = 0, 0, 0
}

func (s *windowStat) inc() {
	s.minute++
	s.hour++
	s.day++
}

func (s *windowStat) exceeded(scope string, q config.SecurityQuotaConf) (Trip, bool) {
	switch {
	case q.Minute > 0 && s.minute > q.Minute:
		return Trip{Scope: scope, Window: WindowMinute, Count: s.minute, Limit: q.Minute}, true
	case q.Hour > 0 && s.hour > q.Hour:
		return Trip{Scope: scope, Window: WindowHour, Count: s.hour, Limit: q.Hour}, true
	case q.Day > 0 && s.day > q.Day:
		return Trip{Scope: scope, Window: WindowDay, Count: s.day, Limit: q.Day}, true
	}
	return Trip{}, false
}

// Snapshot 某个窗口的当前计数
type Snapshot struct {
	Minute int `json:"minute"`
	Hour   int `json:"hour"`
	Day    int `json:"day"`
}

func (s *windowStat) snapshot() Snapshot {
	return Snapshot{Minute: s.minute, Hour: s.hour, Day: s.day}
}

func NewCounter() *Counter {
	return &Counter{ipStats: map[string]*windowStat{}}
}

// Allow 计入一次操作，并判断是否超出配额
//
// 计数先自增再比较：配额为 30 时，第 31 次请求会被判定为超限。
func (c *Counter) Allow(ip string, now time.Time, ipQuota config.SecurityQuotaConf, globalQuota config.SecurityQuotaConf) (Trip, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.sweepLocked(now)

	st := c.ipStats[ip]
	if st == nil {
		st = newWindowStat(now)
		c.ipStats[ip] = st
	}
	st.roll(now)
	st.inc()
	st.lastSeen = now

	c.global.roll(now)
	c.global.inc()

	if trip, ok := st.exceeded(ScopeIP, ipQuota); ok {
		return trip, false
	}
	if trip, ok := c.global.exceeded(ScopeGlobal, globalQuota); ok {
		return trip, false
	}
	return Trip{}, true
}

// Snapshot 返回某 IP 当前的计数，不改变计数状态
func (c *Counter) Snapshot(ip string, now time.Time) Snapshot {
	c.mu.Lock()
	defer c.mu.Unlock()

	st := c.ipStats[ip]
	if st == nil {
		return Snapshot{}
	}
	st.roll(now)
	return st.snapshot()
}

// GlobalSnapshot 返回全局计数
func (c *Counter) GlobalSnapshot(now time.Time) Snapshot {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.global.roll(now)
	return c.global.snapshot()
}

// ResetIP 清空单个 IP 的计数（验证码通过后调用）
func (c *Counter) ResetIP(ip string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.ipStats, ip)
}

// Reset 清空所有计数（配置变更后调用）
func (c *Counter) Reset() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.ipStats = map[string]*windowStat{}
	c.global = *newWindowStat(time.Now())
}

// Len 返回正在被统计的 IP 数量
func (c *Counter) Len() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return len(c.ipStats)
}

// 清理一天以上没有请求的 IP，避免内存无限增长
func (c *Counter) sweepLocked(now time.Time) {
	if now.Sub(c.lastSweep) < time.Minute {
		return
	}
	c.lastSweep = now
	for ip, st := range c.ipStats {
		if now.Sub(st.lastSeen) > 24*time.Hour {
			delete(c.ipStats, ip)
		}
	}
}
