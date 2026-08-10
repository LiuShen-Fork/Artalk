package handler

import (
	"sync"
	"time"

	"github.com/artalkjs/artalk/v2/internal/config"
)

var uploadRateLimiter = newUploadRateLimiter()

type uploadRateLimiterStore struct {
	mu   sync.Mutex
	hits map[string]uploadRateLimitHit
}

type uploadRateLimitHit struct {
	count       int
	windowStart time.Time
}

func newUploadRateLimiter() *uploadRateLimiterStore {
	return &uploadRateLimiterStore{
		hits: map[string]uploadRateLimitHit{},
	}
}

func (l *uploadRateLimiterStore) Allow(ip string, conf config.UploadRateLimitConf) bool {
	if !conf.Enabled || conf.IPLimit <= 0 || conf.WindowSeconds <= 0 {
		return true
	}
	if ip == "" {
		ip = "unknown"
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	now := time.Now()
	window := time.Duration(conf.WindowSeconds) * time.Second
	hit := l.hits[ip]
	if hit.windowStart.IsZero() || now.Sub(hit.windowStart) >= window {
		l.hits[ip] = uploadRateLimitHit{count: 1, windowStart: now}
		return true
	}
	if hit.count >= conf.IPLimit {
		return false
	}

	hit.count++
	l.hits[ip] = hit
	return true
}
