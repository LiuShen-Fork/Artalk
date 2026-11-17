package security

import (
	"net"
	"strings"
	"sync"
	"time"

	"github.com/artalkjs/artalk/v2/internal/entity"
)

// IPMatcher 一组 IP / CIDR 的快速匹配器
type IPMatcher struct {
	exact map[string]struct{}
	nets  []*net.IPNet
}

func NewIPMatcher() *IPMatcher {
	return &IPMatcher{exact: map[string]struct{}{}}
}

// NewIPMatcherFromList 由配置中的 IP 列表构建匹配器（白名单等）
func NewIPMatcherFromList(list []string) *IPMatcher {
	m := NewIPMatcher()
	m.ReplaceAll(list)
	return m
}

// Add 解析并加入一个 IP 或 CIDR，返回是否成功
func (m *IPMatcher) Add(raw string) bool {
	target, network, isCIDR, err := ParseBanTarget(raw)
	if err != nil {
		return false
	}
	if isCIDR {
		m.nets = append(m.nets, network)
		return true
	}
	m.exact[target] = struct{}{}
	return true
}

// ReplaceAll 用新的列表整体替换匹配器内容
func (m *IPMatcher) ReplaceAll(list []string) {
	m.exact = map[string]struct{}{}
	m.nets = nil
	for _, item := range list {
		m.Add(item)
	}
}

func (m *IPMatcher) Contains(ipStr string) bool {
	ip := net.ParseIP(strings.TrimSpace(ipStr))
	if ip == nil {
		return false
	}
	if _, ok := m.exact[ip.String()]; ok {
		return true
	}
	for _, n := range m.nets {
		if n.Contains(ip) {
			return true
		}
	}
	return false
}

func (m *IPMatcher) Len() int {
	if m == nil {
		return 0
	}
	return len(m.exact) + len(m.nets)
}

// ParseBanTarget 解析封禁目标：单个 IP 或 CIDR 网段
//
// 返回规范化的 IP 字符串、网段（仅 CIDR 时非空）以及是否为 CIDR。
func ParseBanTarget(raw string) (target string, network *net.IPNet, isCIDR bool, err error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", nil, false, errInvalidBanTarget
	}

	if strings.Contains(raw, "/") {
		_, network, err = net.ParseCIDR(raw)
		if err != nil {
			return "", nil, false, errInvalidBanTarget
		}
		return network.String(), network, true, nil
	}

	ip := net.ParseIP(raw)
	if ip == nil {
		return "", nil, false, errInvalidBanTarget
	}
	return ip.String(), nil, false, nil
}

// banRange 一条网段封禁
type banRange struct {
	network *net.IPNet
	ban     entity.IPBan
}

// BanStore 生效中的封禁记录索引（内存态，由 SecurityService 从数据库同步）
type BanStore struct {
	mu     sync.RWMutex
	exact  map[string]entity.IPBan
	ranges []banRange
}

func NewBanStore() *BanStore {
	return &BanStore{exact: map[string]entity.IPBan{}}
}

// ReplaceAll 用最新的封禁记录整体替换索引，已过期或格式非法的记录会被忽略
func (s *BanStore) ReplaceAll(bans []entity.IPBan, now time.Time) {
	exact := make(map[string]entity.IPBan, len(bans))
	ranges := make([]banRange, 0, len(bans))

	for _, ban := range bans {
		if !ban.IsActive(now) {
			continue
		}
		target, network, isCIDR, err := ParseBanTarget(ban.IP)
		if err != nil {
			continue
		}
		ban.IP = target
		ban.IsCIDR = isCIDR
		if isCIDR {
			ranges = append(ranges, banRange{network: network, ban: ban})
			continue
		}
		exact[target] = ban
	}

	s.mu.Lock()
	s.exact = exact
	s.ranges = ranges
	s.mu.Unlock()
}

// Match 判断 IP 是否被命中封禁
func (s *BanStore) Match(ipStr string, now time.Time) (entity.IPBan, bool) {
	ip := net.ParseIP(strings.TrimSpace(ipStr))
	if ip == nil {
		return entity.IPBan{}, false
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	if ban, ok := s.exact[ip.String()]; ok && ban.IsActive(now) {
		return ban, true
	}
	for _, r := range s.ranges {
		if r.network.Contains(ip) && r.ban.IsActive(now) {
			return r.ban, true
		}
	}
	return entity.IPBan{}, false
}

// Len 生效中的封禁条目数量
func (s *BanStore) Len() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.exact) + len(s.ranges)
}
