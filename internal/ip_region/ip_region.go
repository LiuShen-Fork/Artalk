package ip_region

import (
	"net"
	"strings"

	"github.com/artalkjs/artalk/v2/internal/config"
	"github.com/artalkjs/artalk/v2/internal/log"
)

type IPRegion struct {
	conf IPRegionConf
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
	if strings.TrimSpace(ip) == "" {
		return ""
	}

	ip = ipScraper(ip)
	dbPath := selectDBPathForIP(ip, ipRegion.conf)
	region, err := search(ip, dbPath, ipRegion.conf.CacheEnabled)
	if err != nil {
		if !strings.HasPrefix(err.Error(), "invalid ip address") {
			log.Warn("[IP2Region] ", err)
		}
		return ""
	}

	return regionScraper(region, config.IPRegionPrecision(ipRegion.conf.Precision))
}

func selectDBPathForIP(ip string, conf IPRegionConf) string {
	parsedIP := net.ParseIP(ip)
	if parsedIP != nil && parsedIP.To4() == nil && strings.TrimSpace(conf.DBPathV6) != "" {
		return conf.DBPathV6
	}

	return conf.DBPath
}
