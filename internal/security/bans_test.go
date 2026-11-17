package security

import (
	"net"
	"testing"
	"time"

	"github.com/artalkjs/artalk/v2/internal/entity"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func netIP(t *testing.T, raw string) net.IP {
	ip := net.ParseIP(raw)
	require.NotNil(t, ip)
	return ip
}

func TestParseBanTarget(t *testing.T) {
	t.Run("plain ip", func(t *testing.T) {
		target, network, isCIDR, err := ParseBanTarget(" 203.0.113.7 ")
		require.NoError(t, err)
		assert.Equal(t, "203.0.113.7", target)
		assert.Nil(t, network)
		assert.False(t, isCIDR)
	})

	t.Run("ipv6", func(t *testing.T) {
		target, network, isCIDR, err := ParseBanTarget("2001:db8::1")
		require.NoError(t, err)
		assert.Equal(t, "2001:db8::1", target)
		assert.Nil(t, network)
		assert.False(t, isCIDR)
	})

	t.Run("cidr", func(t *testing.T) {
		target, network, isCIDR, err := ParseBanTarget("10.0.0.0/8")
		require.NoError(t, err)
		assert.Equal(t, "10.0.0.0/8", target)
		require.NotNil(t, network)
		assert.True(t, isCIDR)
		assert.True(t, network.Contains(netIP(t, "10.1.2.3")))
	})

	t.Run("invalid", func(t *testing.T) {
		for _, raw := range []string{"", "abc", "1.2.3.4/33", "999.0.0.1", "10.0.0.0/"} {
			_, _, _, err := ParseBanTarget(raw)
			assert.ErrorIs(t, err, errInvalidBanTarget, "target %q should be rejected", raw)
		}
	})
}

func TestIPMatcher(t *testing.T) {
	m := NewIPMatcher()

	assert.True(t, m.Add("203.0.113.7"))
	assert.True(t, m.Add("10.0.0.0/8"))
	assert.False(t, m.Add("not-an-ip"))
	assert.Equal(t, 2, m.Len())

	assert.True(t, m.Contains("203.0.113.7"))
	assert.False(t, m.Contains("203.0.113.8"))
	assert.True(t, m.Contains("10.9.8.7"), "cidr entries match their range")

	m.ReplaceAll([]string{"192.168.0.0/16", " "})
	assert.Equal(t, 1, m.Len())
	assert.True(t, m.Contains("192.168.5.5"))
	assert.False(t, m.Contains("203.0.113.7"), "old entries are dropped on replace")
}

func TestBanStore(t *testing.T) {
	now := time.Date(2026, 11, 17, 12, 0, 0, 0, time.UTC)
	future := now.Add(time.Hour)
	past := now.Add(-time.Hour)

	store := NewBanStore()
	store.ReplaceAll([]entity.IPBan{
		{IP: "203.0.113.7", Reason: "spam", Source: entity.IPBanSourceManual},
		{IP: "198.51.100.0/24", Reason: "range", Source: entity.IPBanSourceAuto},
		{IP: "192.0.2.1", Reason: "expired", ExpireAt: &past},
		{IP: "192.0.2.2", Reason: "still valid", ExpireAt: &future},
		{IP: "garbage", Reason: "invalid"},
	}, now)

	assert.Equal(t, 3, store.Len())

	ban, ok := store.Match("203.0.113.7", now)
	require.True(t, ok)
	assert.Equal(t, "spam", ban.Reason)

	ban, ok = store.Match("198.51.100.42", now)
	require.True(t, ok)
	assert.Equal(t, "range", ban.Reason)

	_, ok = store.Match("192.0.2.1", now)
	assert.False(t, ok, "expired bans are skipped")

	_, ok = store.Match("192.0.2.2", now)
	assert.True(t, ok)

	_, ok = store.Match("192.0.2.2", future.Add(time.Minute))
	assert.False(t, ok, "a ban stops matching once it expires")

	_, ok = store.Match("203.0.113.8", now)
	assert.False(t, ok)
}
