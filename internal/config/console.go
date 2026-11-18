package config

import "strings"

// ConsoleConf 控制台的品牌信息与自定义样式、脚本
//
// 控制台展示什么名字、什么图标、登录页写一句什么介绍，以及管理员想注入的
// CSS / JS，都放在这里。都是部署级配置，不含任何用户数据。
type ConsoleConf struct {
	Name        string `koanf:"name" json:"name"`               // 控制台名称，留空显示 Artalk
	Icon        string `koanf:"icon" json:"icon"`               // 站点图标 URL，留空使用字母徽标
	Description string `koanf:"description" json:"description"` // 站点描述，显示在登录页
	CSS         string `koanf:"css" json:"css"`                 // 控制台自定义样式
	JS          string `koanf:"js" json:"js"`                   // 控制台自定义脚本
}

const (
	// MaxConsoleCodeSize 单个代码片段的长度上限（字节）
	// 配置项不是代码仓库，超过这个量级应该改前端而不是塞配置。
	MaxConsoleCodeSize = 256 * 1024
	// MaxConsoleNameRunes 控制台名称的长度上限
	MaxConsoleNameRunes = 40
	// MaxConsoleDescriptionRunes 站点描述的长度上限
	MaxConsoleDescriptionRunes = 300
	// MaxConsoleIconBytes 图标地址的长度上限
	MaxConsoleIconBytes = 2048
)

// Normalized 去掉首尾空白，避免保存一堆空行
func (c ConsoleConf) Normalized() ConsoleConf {
	return ConsoleConf{
		Name:        strings.TrimSpace(c.Name),
		Icon:        strings.TrimSpace(c.Icon),
		Description: strings.TrimSpace(c.Description),
		CSS:         strings.TrimSpace(c.CSS),
		JS:          strings.TrimSpace(c.JS),
	}
}
