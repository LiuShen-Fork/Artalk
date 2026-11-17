package config

import "strings"

// CustomConf 管理后台的自定义样式与脚本
//
// 站点管理员希望微调控制台外观时（换配色、隐藏某个按钮、嵌入统计脚本），
// 不必重新构建前端：把 CSS / JS 写进配置，控制台启动时会注入到页面中。
type CustomConf struct {
	CSS string `koanf:"css" json:"css"` // 控制台自定义样式
	JS  string `koanf:"js" json:"js"`   // 控制台自定义脚本
}

// MaxCustomCodeSize 单个片段的长度上限（字节）
// 配置项不是代码仓库，超过这个量级应该改前端而不是塞配置。
const MaxCustomCodeSize = 256 * 1024

// Normalized 去掉首尾空白，避免保存一堆空行
func (c CustomConf) Normalized() CustomConf {
	return CustomConf{
		CSS: strings.TrimSpace(c.CSS),
		JS:  strings.TrimSpace(c.JS),
	}
}

// IsEmpty 是否没有任何自定义代码
func (c CustomConf) IsEmpty() bool {
	return c.CSS == "" && c.JS == ""
}
