package security

import (
	"fmt"
	"testing"
	"time"

	"github.com/artalkjs/artalk/v2/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func quota(minute, hour, day int) config.SecurityQuotaConf {
	return config.SecurityQuotaConf{Minute: minute, Hour: hour, Day: day}
}

func TestCounterAllowsUpToQuota(t *testing.T) {
	c := NewCounter()
	now := time.Date(2026, 11, 17, 10, 0, 0, 0, time.UTC)
	ipQuota := quota(3, 0, 0)

	for i := 0; i < 3; i++ {
		trip, ok := c.Allow("1.2.3.4", now, ipQuota, config.SecurityQuotaConf{})
		require.True(t, ok, "request #%d should be allowed", i+1)
		require.Equal(t, Trip{}, trip)
	}

	trip, ok := c.Allow("1.2.3.4", now, ipQuota, config.SecurityQuotaConf{})
	assert.False(t, ok)
	assert.Equal(t, Trip{Scope: ScopeIP, Window: WindowMinute, Count: 4, Limit: 3}, trip)
	assert.Equal(t, 60, trip.RetryAfter(now))

	// 其他 IP 不受影响
	_, ok = c.Allow("5.6.7.8", now, ipQuota, config.SecurityQuotaConf{})
	assert.True(t, ok)
}

func TestCounterUnlimitedWhenQuotaIsZero(t *testing.T) {
	c := NewCounter()
	now := time.Date(2026, 11, 17, 10, 0, 0, 0, time.UTC)

	for i := 0; i < 500; i++ {
		_, ok := c.Allow("1.2.3.4", now, config.SecurityQuotaConf{}, config.SecurityQuotaConf{})
		require.True(t, ok)
	}
}

func TestCounterRollsOverWindows(t *testing.T) {
	c := NewCounter()
	start := time.Date(2026, 11, 17, 10, 0, 0, 0, time.UTC)
	ipQuota := quota(1, 2, 0)

	_, ok := c.Allow("1.2.3.4", start, ipQuota, config.SecurityQuotaConf{})
	require.True(t, ok)

	_, ok = c.Allow("1.2.3.4", start, ipQuota, config.SecurityQuotaConf{})
	require.False(t, ok, "same minute should already be over the limit")

	// 同一小时内下一分钟：分钟窗口重置，小时计数继续累加并超限
	nextMinute := start.Add(70 * time.Second)
	trip, ok := c.Allow("1.2.3.4", nextMinute, ipQuota, config.SecurityQuotaConf{})
	require.False(t, ok)
	assert.Equal(t, Trip{Scope: ScopeIP, Window: WindowHour, Count: 3, Limit: 2}, trip)
	assert.Equal(t, 3530, trip.RetryAfter(nextMinute))
	assert.Equal(t, Snapshot{Minute: 1, Hour: 3, Day: 3}, c.Snapshot("1.2.3.4", nextMinute))

	// 进入下一个小时：小时计数也重置
	_, ok = c.Allow("1.2.3.4", start.Add(time.Hour), ipQuota, config.SecurityQuotaConf{})
	assert.True(t, ok)
}

func TestCounterMinuteWindowResetsEachMinute(t *testing.T) {
	c := NewCounter()
	start := time.Date(2026, 11, 17, 10, 0, 0, 0, time.UTC)
	ipQuota := quota(1, 0, 0)

	_, ok := c.Allow("1.2.3.4", start, ipQuota, config.SecurityQuotaConf{})
	require.True(t, ok)
	_, ok = c.Allow("1.2.3.4", start.Add(30*time.Second), ipQuota, config.SecurityQuotaConf{})
	require.False(t, ok, "still the same minute")

	_, ok = c.Allow("1.2.3.4", start.Add(time.Minute), ipQuota, config.SecurityQuotaConf{})
	assert.True(t, ok, "the minute window has rolled over")
}

func TestCounterDayWindowResetsAtMidnight(t *testing.T) {
	c := NewCounter()
	before := time.Date(2026, 11, 17, 23, 59, 0, 0, time.UTC)
	after := time.Date(2026, 11, 18, 0, 1, 0, 0, time.UTC)
	dayQuota := quota(0, 0, 1)

	_, ok := c.Allow("1.2.3.4", before, dayQuota, config.SecurityQuotaConf{})
	require.True(t, ok)

	trip, ok := c.Allow("1.2.3.4", before, dayQuota, config.SecurityQuotaConf{})
	require.False(t, ok)
	assert.Equal(t, WindowDay, trip.Window)
	assert.Equal(t, 60, trip.RetryAfter(before))

	_, ok = c.Allow("1.2.3.4", after, dayQuota, config.SecurityQuotaConf{})
	assert.True(t, ok, "a new day should start from zero again")
}

func TestCounterChecksIPBeforeGlobal(t *testing.T) {
	c := NewCounter()
	now := time.Date(2026, 11, 17, 10, 0, 0, 0, time.UTC)
	globalQuota := quota(1, 0, 0)

	_, ok := c.Allow("1.2.3.4", now, config.SecurityQuotaConf{}, globalQuota)
	require.True(t, ok)

	trip, ok := c.Allow("5.6.7.8", now, config.SecurityQuotaConf{}, globalQuota)
	require.False(t, ok)
	assert.Equal(t, ScopeGlobal, trip.Scope)
	assert.Equal(t, 1, trip.Limit)
}

func TestCounterResetIP(t *testing.T) {
	c := NewCounter()
	now := time.Date(2026, 11, 17, 10, 0, 0, 0, time.UTC)
	ipQuota := quota(1, 0, 0)

	_, ok := c.Allow("1.2.3.4", now, ipQuota, config.SecurityQuotaConf{})
	require.True(t, ok)
	_, ok = c.Allow("1.2.3.4", now, ipQuota, config.SecurityQuotaConf{})
	require.False(t, ok)

	c.ResetIP("1.2.3.4")
	assert.Equal(t, 0, c.Len())

	_, ok = c.Allow("1.2.3.4", now, ipQuota, config.SecurityQuotaConf{})
	assert.True(t, ok)
}

func TestCounterSweepsIdleIPs(t *testing.T) {
	c := NewCounter()
	start := time.Date(2026, 11, 17, 10, 0, 0, 0, time.UTC)

	for i := 0; i < 50; i++ {
		c.Allow(fmt.Sprintf("10.0.0.%d", i), start, config.SecurityQuotaConf{}, config.SecurityQuotaConf{})
	}
	require.Greater(t, c.Len(), 1)

	// 25 小时后，空闲 IP 会被清理
	c.Allow("192.168.1.1", start.Add(25*time.Hour), config.SecurityQuotaConf{}, config.SecurityQuotaConf{})
	assert.Equal(t, 1, c.Len())
}

func TestTripRetryAfterNeverZero(t *testing.T) {
	now := time.Date(2026, 11, 17, 10, 0, 59, int(999*time.Millisecond), time.UTC)
	assert.Equal(t, 1, (Trip{Window: WindowMinute}).RetryAfter(now))
	assert.Equal(t, 1, (Trip{Window: "unknown"}).RetryAfter(now))
	assert.Equal(t, 3600, (Trip{Window: WindowHour}).RetryAfter(time.Date(2026, 11, 17, 10, 0, 0, 0, time.UTC)))
}
