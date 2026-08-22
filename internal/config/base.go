package config

//go:generate go run ./meta/gen --format go --locale zh-CN --pkg config -o ./cache.go

import (
	"fmt"
	"os"
	"path"
	"strconv"
	"strings"
	"unicode"

	"github.com/adrg/xdg"
	"github.com/artalkjs/artalk/v2/internal/config/env_provider"
	"github.com/artalkjs/artalk/v2/internal/log"
	"github.com/artalkjs/artalk/v2/internal/utils"
	"github.com/knadh/koanf"
	"github.com/knadh/koanf/parsers/yaml"
	"github.com/knadh/koanf/providers/file"
)

func New() *Config {
	conf := &Config{}

	return conf
}

// 从文件中创建配置实例
func NewFromFile(cfgFile string) (*Config, error) {
	kf := koanf.New(".")

	// load yaml config
	if err := kf.Load(file.Provider(cfgFile), yaml.Parser()); err != nil {
		return nil, fmt.Errorf("config file read error: %w", err)
	}

	// load environment variables and merge into the loaded config
	const envPrefix = "ATK_"
	if err := kf.Load(env_provider.Provider(envPrefix, EnvPathMapCache), nil); err != nil {
		return nil, fmt.Errorf("config environment variable parse error: %w", err)
	}

	// create new config instance
	conf := &Config{
		cfgFile: cfgFile,
	}

	// use koanf parser to decode config file to instance
	if err := kf.Unmarshal("", conf); err != nil {
		return nil, fmt.Errorf("config file parse error: %w", err)
	}

	// patch config
	{
		conf.historyPatch()
		conf.normalPatch()
		conf.i18nPatch()
		conf.ipRegionPatch()
	}

	return conf, nil
}

func (conf *Config) GetCfgFileLoaded() string {
	return conf.cfgFile
}

// 配置修补
func (conf *Config) normalPatch() {
	// 检查 app_key 是否设置
	if strings.TrimSpace(conf.AppKey) == "" {
		conf.AppKey = utils.RandomString(16)
		log.Warn("config `app_key` is not set, now it is random value")
	}

	// 检查时区
	conf.TimeZone = strings.TrimSpace(conf.TimeZone)
	if conf.TimeZone == "" {
		conf.TimeZone = "Local"
		log.Warn("config `timezone` is not set, now it is: " + strconv.Quote(conf.TimeZone))
	}

	// 默认站点配置
	conf.SiteDefault = strings.TrimSpace(conf.SiteDefault)
	if conf.SiteDefault == "" {
		conf.SiteDefault = "Default Site"
		log.Warn("config `site_default` is not set, now it is: " + strconv.Quote(conf.SiteDefault))
	}

	// 缓存配置
	if conf.Cache.Type == "" {
		// 默认使用内建缓存
		conf.Cache.Type = CacheTypeBuiltin
	}

	// 配置文件 alias 处理
	if conf.Captcha.ActionLimit == 0 {
		conf.Captcha.Always = true
	}

	// 管理员邮件通知配置继承
	if conf.AdminNotify.Email.MailSubject == "" {
		if conf.AdminNotify.NotifySubject != "" {
			conf.AdminNotify.Email.MailSubject = conf.AdminNotify.NotifySubject
		} else if conf.Email.MailSubject != "" {
			conf.AdminNotify.Email.MailSubject = conf.Email.MailSubject
		}
	}

	// 默认待审模式下开启管理员通知嘈杂模式，保证管理员能看到待审核文章
	if conf.Moderator.PendingDefault {
		conf.AdminNotify.NoiseMode = true
	}

	// AI 审核兼容旧配置，避免启用后因缺少新字段而直接请求失败
	if conf.Moderator.AI.Enabled {
		if strings.TrimSpace(string(conf.Moderator.AI.APIType)) == "" {
			conf.Moderator.AI.APIType = AIAPITypeResponses
		}
		if strings.TrimSpace(conf.Moderator.AI.BaseURL) == "" {
			conf.Moderator.AI.BaseURL = "https://api.openai.com/v1"
		}
		if strings.TrimSpace(conf.Moderator.AI.Prompt) == "" {
			conf.Moderator.AI.Prompt = DefaultAIModerationPrompt
		}
		if conf.Moderator.AI.DisableThinking == nil {
			disableThinking := true
			conf.Moderator.AI.DisableThinking = &disableThinking
		}
	}

	// AI comment assistant defaults. These also apply when it is enabled via
	// environment variables without a complete YAML section.
	if strings.TrimSpace(conf.AIAssistant.Name) == "" {
		conf.AIAssistant.Name = "清羽酱"
	}
	if strings.TrimSpace(conf.AIAssistant.Email) == "" {
		conf.AIAssistant.Email = "ai-assistant@example.com"
	}
	if strings.TrimSpace(string(conf.AIAssistant.APIType)) == "" {
		conf.AIAssistant.APIType = AIAPITypeResponses
	}
	if strings.TrimSpace(conf.AIAssistant.BaseURL) == "" {
		conf.AIAssistant.BaseURL = "https://api.openai.com/v1"
	}
	if conf.AIAssistant.MaxTokens == 0 {
		conf.AIAssistant.MaxTokens = 512
	}
	if conf.AIAssistant.MaxReplyChars == 0 {
		conf.AIAssistant.MaxReplyChars = 300
	}
	if conf.AIAssistant.MaxContextComments == 0 {
		conf.AIAssistant.MaxContextComments = 12
	}
	if conf.AIAssistant.MaxPageChars == 0 {
		conf.AIAssistant.MaxPageChars = 12000
	}
	if conf.AIAssistant.TimeoutSeconds == 0 {
		conf.AIAssistant.TimeoutSeconds = 30
	}
	if conf.AIAssistant.DisableThinking == nil {
		disableThinking := true
		conf.AIAssistant.DisableThinking = &disableThinking
	}

	// 默认将验证码类型设置为 image
	if strings.TrimSpace(string(conf.Captcha.CaptchaType)) == "" {
		conf.Captcha.CaptchaType = TypeImage
	}

	// 图片上传存放路径默认设置
	if conf.ImgUpload.Path == "" {
		conf.ImgUpload.Path = "./data/artalk-img/"
		log.Warn("[Image Upload] img_upload.path is not configured, using the default value: " + strconv.Quote(conf.ImgUpload.Path))
	}

	// HTTP 配置默认值
	if conf.HTTP.BodyLimit <= 0 {
		conf.HTTP.BodyLimit = 100
	}
	if conf.HTTP.ProxyHeader == nil {
		log.Warn("config `http.proxy_header` is not set, not it is: \"X-Forwarded-For\". If you are not using a reverse proxy or CDN, please set it to blank for preventing IP spoofing.")
		// The reason it doesn't default to empty here is that the default value in the old version
		// was "X-Forwarded-For", and there is no new field in the conf file for this (the parse result would be a nil pointer).
		//
		// Therefore, to avoid BREAKING CHANGES, it remains unchanged with a warning.
		// New user with a new conf file generated by the latest version will have this field set to empty.
		defaultProxyHeader := "X-Forwarded-For"
		conf.HTTP.ProxyHeader = &defaultProxyHeader
	} else {
		*conf.HTTP.ProxyHeader = strings.TrimSpace(*conf.HTTP.ProxyHeader)
	}

	// 社交登录配置
	if conf.Auth.Enabled && strings.TrimSpace(conf.Auth.Callback) == "" {
		callbackURL := "http://localhost:23366/api/v2/auth/:provider/callback"
		log.Warn("[SocialLogin] config `auth.callback` is not set, now it is: ", strconv.Quote(callbackURL))
		conf.Auth.Callback = callbackURL
	}
}

// 多语言配置修补
func (conf *Config) i18nPatch() {
	if conf.Locale == "" {
		conf.Locale = "zh-CN"

		// Keep old configuration files without an explicit locale in Simplified Chinese.
		if confRaw, err := os.ReadFile(conf.GetCfgFileLoaded()); err == nil {
			containsHan := false
			for _, runeValue := range string(confRaw) {
				if unicode.Is(unicode.Han, runeValue) {
					containsHan = true
					break
				}
			}
			if containsHan {
				conf.Locale = "zh-CN"
			}
		}

		log.Warn("config `locale` is not set, now it is: " + strconv.Quote(conf.Locale))
	} else if conf.Locale == "zh" {
		conf.Locale = "zh-CN"
	}

	// Case Convert
	// follow Unicode BCP 47
	// @see https://www.techonthenet.com/js/language_tags.php
	parts := strings.Split(conf.Locale, "-")
	if len(parts) == 2 {
		conf.Locale = strings.ToLower(parts[0]) + "-" + strings.ToUpper(parts[1])
	} else if len(parts) == 1 {
		conf.Locale = strings.ToLower(parts[0])
	}

	// Temporary convert `en-*` to `en`
	if parts[0] == "en" {
		conf.Locale = "en"
	}
}

// IP属地功能配置修补
func (conf *Config) ipRegionPatch() {
	if !conf.IPRegion.Enabled {
		return
	}

	// IP 属地默认数据文件
	if conf.IPRegion.DBPath == "" {
		conf.IPRegion.DBPath = "./data/ip2region.xdb"
	}

	// 检测配置文件是否存在
	if !utils.CheckFileExist(conf.IPRegion.DBPath) {
		log.Warn("未找到 IP 数据库文件：" + strconv.Quote(conf.IPRegion.DBPath) + "，IP 属地功能已禁用，" +
			"参考链接：https://artalk.js.org/guide/frontend/ip-region.html")
		conf.IPRegion.Enabled = false
	}

	// IPv6 数据库为可选配置；未配置时保持原有单库查询逻辑。
	if strings.TrimSpace(conf.IPRegion.DBPathV6) != "" && !utils.CheckFileExist(conf.IPRegion.DBPathV6) {
		log.Warn("未找到 IPv6 IP 数据库文件：" + strconv.Quote(conf.IPRegion.DBPathV6) + "，IPv6 IP 属地解析已禁用")
		conf.IPRegion.DBPathV6 = ""
	}

	// 默认精确到省
	if conf.IPRegion.Precision == "" {
		conf.IPRegion.Precision = string(IPRegionProvince)
	}
}

// 配置修补 for 历史版本
func (conf *Config) historyPatch() {
	if conf.Captcha.ActionTimeout != 0 {
		log.Warn("The config option `captcha.action_timeout` is deprecated, please use `captcha.action_reset` instead")
		if conf.Captcha.ActionReset == 0 {
			conf.Captcha.ActionReset = conf.Captcha.ActionTimeout
		}
	}
	if len(conf.AllowOrigins) != 0 {
		log.Warn("The config option `allow_origins` is deprecated, please use `trusted_domains` instead")
		if len(conf.TrustedDomains) == 0 {
			conf.TrustedDomains = conf.AllowOrigins
		}
	}

	// @version < 2.2.0
	if conf.Notify != nil {
		log.Warn("The config option `notify` is deprecated, please use `admin_notify` instead")
		conf.AdminNotify = *conf.Notify
	}
	if conf.AdminNotify.Email == nil {
		conf.AdminNotify.Email = &AdminEmailConf{
			Enabled: true, // 默认开启管理员邮件通知
		}
	}
	if conf.Email.MailSubjectToAdmin != "" {
		log.Warn("The config option `email.mail_subject_to_admin` is deprecated, please use `admin_notify.email.mail_subject` instead")
		conf.AdminNotify.Email.MailSubject = conf.Email.MailSubjectToAdmin
	}
}

// Try to find the configuration file
//
// The order of the search is:
//  1. Current directory (./artalk.yml)
//  2. Current subdirectory (./data/artalk.yml)
//  3. XDG_CONFIG_HOME (~/.config/artalk.yml)
//  4. /etc/artalk.yml (for linux packing version)
func RetrieveConfigFile() string {
	lookupDirs := []string{"", "data", xdg.ConfigHome, "/etc"}
	paths := []string{}
	for _, dir := range lookupDirs {
		for _, name := range CONF_DEFAULT_FILENAMES {
			if dir == "" || dir == "data" {
				paths = append(paths, path.Join(dir, name))
			} else {
				paths = append(paths, path.Join(dir, "artalk", name))
			}
		}
	}
	for _, path := range paths {
		if utils.CheckFileExist(path) {
			return path
		}
	}
	return ""
}

// Try to find the data directory (work directory)
//
// The order of the search is:
//  1. XDG_DATA_HOME (~/.local/share)
//  2. /var/lib (for linux packing version)
func RetrieveDataDir() string {
	lookupDirs := []string{xdg.DataHome, "/var/lib"}
	for _, dir := range lookupDirs {
		dir = path.Join(dir, "artalk")
		if utils.CheckDirExist(dir) {
			return dir
		}
	}
	return ""
}
