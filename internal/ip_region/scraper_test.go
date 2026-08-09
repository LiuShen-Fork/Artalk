package ip_region

import (
	"testing"

	"github.com/artalkjs/artalk/v2/internal/config"
)

func TestIpScraper(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"  127.0.0.1  ", "127.0.0.1"},
		{"192.168.1.1, 10.0.0.1", "192.168.1.1"},
		{"  2001:0db8:85a3:0000:0000:8a2e:0370:7334  ", "2001:0db8:85a3:0000:0000:8a2e:0370:7334"},
		{"", ""},
		{", 172.16.0.1", ""},
	}

	for _, test := range tests {
		result := ipScraper(test.input)
		if result != test.expected {
			t.Errorf("For input '%s', expected '%s' but got '%s'", test.input, test.expected, result)
		}
	}
}

func TestSelectDBPathForIP(t *testing.T) {
	withV6Database := IPRegionConf{
		IPRegionConf: config.IPRegionConf{
			DBPath:   "ipv4.xdb",
			DBPathV6: "ipv6.xdb",
		},
	}
	withoutV6Database := IPRegionConf{
		IPRegionConf: config.IPRegionConf{
			DBPath: "ipv4.xdb",
		},
	}

	tests := []struct {
		name string
		ip   string
		conf IPRegionConf
		want string
	}{
		{name: "IPv4", ip: "203.0.113.10", conf: withV6Database, want: "ipv4.xdb"},
		{name: "IPv6", ip: "2001:db8::10", conf: withV6Database, want: "ipv6.xdb"},
		{name: "IPv6 without dedicated database", ip: "2001:db8::10", conf: withoutV6Database, want: "ipv4.xdb"},
		{name: "invalid IP", ip: "not-an-ip", conf: withV6Database, want: "ipv4.xdb"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := selectDBPathForIP(test.ip, test.conf); got != test.want {
				t.Fatalf("selectDBPathForIP(%q) = %q, want %q", test.ip, got, test.want)
			}
		})
	}
}

func TestRegionScraper(t *testing.T) {
	tests := []struct {
		raw       string
		precision config.IPRegionPrecision
		expected  string
	}{
		{"0|0|0|0|0", config.IPRegionCountry, ""},
		{"中国|0|上海市|上海市|电信", config.IPRegionProvince, "上海"},
		{"中国|0|上海市|上海市|电信", config.IPRegionCity, "上海"},
		{"中国|0|重庆市|重庆市|电信", config.IPRegionCity, "重庆"},
		{"中国|0|浙江省|杭州市|电信", config.IPRegionProvince, "浙江"},
		{"中国|0|北京市|北京市|电信", config.IPRegionCity, "北京"},
		{"中国|0|四川省|泸州市|电信", config.IPRegionCity, "四川泸州"},
		{"中国|0|广东省|深圳市|电信", config.IPRegionCity, "广东深圳"},
		{"中国|0|湖南省|岳阳市|电信", config.IPRegionCity, "湖南岳阳"},
		{"中国|0|陕西省|西安市|电信", config.IPRegionCity, "陕西西安"},
		{"中国|0|河北省|石家庄市|电信", config.IPRegionCity, "河北石家庄"},
		{"中国|0|江苏省|南京市|电信", config.IPRegionCity, "江苏南京"},
		{"美国|0|0|0|微软云", config.IPRegionCity, "美国"},
	}

	for _, test := range tests {
		result := regionScraper(test.raw, test.precision)
		if result != test.expected {
			t.Errorf("For input '%s' and precision '%s', expected '%s' but got '%s'", test.raw, test.precision, test.expected, result)
		}
	}
}
