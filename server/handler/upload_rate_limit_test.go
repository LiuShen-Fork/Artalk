package handler

import (
	"testing"
	"time"

	"github.com/artalkjs/artalk/v2/internal/config"
	"github.com/stretchr/testify/assert"
)

func TestUploadRateLimiter(t *testing.T) {
	limiter := newUploadRateLimiter()
	conf := config.UploadRateLimitConf{Enabled: true, IPLimit: 2, WindowSeconds: 1}

	assert.True(t, limiter.Allow("127.0.0.1", conf))
	assert.True(t, limiter.Allow("127.0.0.1", conf))
	assert.False(t, limiter.Allow("127.0.0.1", conf))
	assert.True(t, limiter.Allow("127.0.0.2", conf))

	time.Sleep(1100 * time.Millisecond)
	assert.True(t, limiter.Allow("127.0.0.1", conf))
}
