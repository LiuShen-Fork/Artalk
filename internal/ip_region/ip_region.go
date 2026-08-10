package ip_region

import (
	"errors"
	"fmt"
	"net"
	"strings"
	"sync"

	"github.com/artalkjs/artalk/v2/internal/config"
	"github.com/artalkjs/artalk/v2/internal/log"
	ipdb "github.com/ipipdotnet/ipdb-go"
)

type IPRegion struct {
	conf   IPRegionConf
	mu     sync.Mutex
	cached map[string]regionSearcher
}

type IPRegionConf struct {
	config.IPRegionConf
	CacheEnabled bool
}

func NewIPRegion(conf IPRegionConf) *IPRegion {
	return &IPRegion{
		conf: conf,
	}
}

func (ipRegion *IPRegion) IP2Region(ip string) string {
	ip = ipScraper(ip)
	if net.ParseIP(ip) == nil {
		return ""
	}

	region, err := ipRegion.search(ip)
	if err != nil {
		if !strings.HasPrefix(err.Error(), "invalid ip address") &&
			!errors.Is(err, ipdb.ErrNoSupportIPv4) && !errors.Is(err, ipdb.ErrNoSupportIPv6) &&
			!errors.Is(err, ipdb.ErrDataNotExists) {
			log.Warnf("[IP2Region] %q: %v", ipRegion.conf.DBPath, err)
		}
		return ""
	}

	return region.format(config.IPRegionPrecision(ipRegion.conf.Precision))
}

func (ipRegion *IPRegion) search(ip string) (result region, err error) {
	// Both upstream readers may panic on corrupt database offsets. Treat a
	// broken local database as a failed lookup, not a failed comment request.
	defer func() {
		if recovered := recover(); recovered != nil {
			result = region{}
			err = fmt.Errorf("invalid IP region database: %v", recovered)
		}
	}()
	dbPath := selectDBPathForIP(ip, ipRegion.conf)
	searcher, err := ipRegion.getSearcher(dbPath)
	if err != nil {
		return region{}, err
	}
	if !ipRegion.reuseSearcher(dbPath) {
		defer searcher.Close()
	}
	return searcher.Search(ip)
}

func (ipRegion *IPRegion) reuseSearcher(dbPath string) bool {
	// IPDB always loads the whole file into memory; retain it regardless of the
	// global cache setting. XDB keeps its existing memory/file mode selection.
	return ipRegion.conf.CacheEnabled || isIPDBPath(dbPath)
}

func (ipRegion *IPRegion) getSearcher(dbPath string) (regionSearcher, error) {
	if !ipRegion.reuseSearcher(dbPath) {
		return newSearcher(dbPath, false)
	}
	ipRegion.mu.Lock()
	defer ipRegion.mu.Unlock()
	if ipRegion.cached == nil {
		ipRegion.cached = make(map[string]regionSearcher)
	}
	if searcher, ok := ipRegion.cached[dbPath]; ok {
		return searcher, nil
	}
	searcher, err := newSearcher(dbPath, true)
	if err != nil {
		return nil, err // Retry on the next lookup if the file is repaired.
	}
	ipRegion.cached[dbPath] = searcher
	return searcher, nil
}

func selectDBPathForIP(ip string, conf IPRegionConf) string {
	parsedIP := net.ParseIP(ip)
	if parsedIP != nil && parsedIP.To4() == nil && strings.TrimSpace(conf.DBPathV6) != "" {
		return conf.DBPathV6
	}

	return conf.DBPath
}
