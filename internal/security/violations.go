package security

import (
	"errors"
	"sync"
	"time"
)

var errInvalidBanTarget = errors.New("invalid ip or cidr target")

// ViolationTracker 滑动窗口内的违规次数统计，用于触发自动封禁
type ViolationTracker struct {
	mu sync.Mutex
	m  map[string][]time.Time
}

func NewViolationTracker() *ViolationTracker {
	return &ViolationTracker{m: map[string][]time.Time{}}
}

// Add 记一次违规，返回窗口内（含本次）的违规次数
func (t *ViolationTracker) Add(ip string, now time.Time, window time.Duration) int {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.sweepLocked(now)

	list := append(t.m[ip], now)
	list = pruneBefore(list, now.Add(-window))
	t.m[ip] = list
	return len(list)
}

// Count 窗口内的违规次数
func (t *ViolationTracker) Count(ip string, window time.Duration, now time.Time) int {
	t.mu.Lock()
	defer t.mu.Unlock()

	return len(pruneBefore(t.m[ip], now.Add(-window)))
}

// Reset 清空某 IP 的违规记录（封禁后会调用）
func (t *ViolationTracker) Reset(ip string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	delete(t.m, ip)
}

// Len 正在跟踪的 IP 数量
func (t *ViolationTracker) Len() int {
	t.mu.Lock()
	defer t.mu.Unlock()
	return len(t.m)
}

func (t *ViolationTracker) sweepLocked(now time.Time) {
	for ip, list := range t.m {
		if len(list) == 0 || now.Sub(list[len(list)-1]) > 24*time.Hour {
			delete(t.m, ip)
		}
	}
}

func pruneBefore(list []time.Time, before time.Time) []time.Time {
	idx := 0
	for idx < len(list) && list[idx].Before(before) {
		idx++
	}
	if idx == 0 {
		return list
	}
	return append([]time.Time{}, list[idx:]...)
}
