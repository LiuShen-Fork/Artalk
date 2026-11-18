package handler

import (
	"errors"
	"strings"
	"unicode/utf8"

	"github.com/artalkjs/artalk/v2/internal/config"
	"github.com/artalkjs/artalk/v2/internal/core"
	"github.com/artalkjs/artalk/v2/internal/log"
	"github.com/artalkjs/artalk/v2/server/common"
	"github.com/gofiber/fiber/v2"
)

var (
	errConsoleCodeTooLarge = errors.New("自定义代码超出长度上限")
	errConsoleNotUTF8      = errors.New("内容必须是 UTF-8 文本")
	errConsoleNameTooLong  = errors.New("控制台名称过长")
	errConsoleDescTooLong  = errors.New("站点描述过长")
	errConsoleIconInvalid  = errors.New("图标地址必须是 http(s) 链接或以 / 开头的站内路径")
	errConsoleUnavailable  = errors.New("配置来自环境变量或内存，无法写入控制台设置")
)

type ResponseConsole struct {
	Name        string `json:"name"`
	Icon        string `json:"icon"`
	Description string `json:"description"`
	CSS         string `json:"css"`
	JS          string `json:"js"`
}

type ParamsConsoleSave struct {
	Name        *string `json:"name" validate:"optional"`
	Icon        *string `json:"icon" validate:"optional"`
	Description *string `json:"description" validate:"optional"`
	CSS         *string `json:"css" validate:"optional"`
	JS          *string `json:"js" validate:"optional"`
}

// ConsoleGet 读取控制台的品牌信息与自定义 CSS / JS
//
// 由控制台在启动时拉取：名称、图标、描述决定登录页与侧栏显示什么，CSS / JS 会被
// 注入页面。内容由站长维护，不含任何用户数据，因此无需鉴权——登录页也要用同一份
// 品牌信息，才能保证换肤前后风格一致。
//
// @Id           ConsoleGet
// @Summary      获取控制台站点信息与自定义样式脚本
// @Tags         Console
// @Produce      json
// @Success      200  {object}  ResponseConsole
// @Router       /sidebar/console [get]
func ConsoleGet(app *core.App, router fiber.Router) {
	router.Get("/sidebar/console", func(c *fiber.Ctx) error {
		return common.RespData(c, buildConsoleResponse(app))
	})
}

// ConsoleSave 保存控制台的品牌信息与自定义 CSS / JS
//
// 只改动 `console` 段，配置文件中的注释与其他配置保持不变，保存后立即生效（不重启服务）。
// 脚本会在控制台页面内执行，请只填写自己信任的代码。
//
// @Id           ConsoleSave
// @Summary      保存控制台站点信息与自定义样式脚本
// @Tags         Console
// @Security     ApiKeyAuth
// @Accept       json
// @Produce      json
// @Param        data  body  ParamsConsoleSave  true  "站点信息与自定义代码"
// @Success      200  {object}  ResponseConsole
// @Router       /sidebar/console [put]
func ConsoleSave(app *core.App, router fiber.Router) {
	router.Put("/sidebar/console", common.AdminGuard(app, func(c *fiber.Ctx) error {
		var p ParamsConsoleSave
		if isOK, resp := common.ParamsDecode(c, &p); !isOK {
			return resp
		}

		updates, err := consoleUpdates(app, &p)
		if err != nil {
			return common.RespError(c, fiber.StatusBadRequest, err.Error())
		}

		if len(updates) > 0 {
			newConf, err := writeConfigFile(app, updates)
			if err != nil {
				return common.RespError(c, fiber.StatusInternalServerError, errConsoleUnavailable.Error()+": "+err.Error())
			}
			app.SetConf(newConf)
			log.Info("[Console] settings updated")
		}

		return common.RespData(c, buildConsoleResponse(app))
	}))
}

func buildConsoleResponse(app *core.App) ResponseConsole {
	console := app.Conf().Console.Normalized()
	return ResponseConsole{
		Name:        console.Name,
		Icon:        console.Icon,
		Description: console.Description,
		CSS:         console.CSS,
		JS:          console.JS,
	}
}

func consoleUpdates(app *core.App, p *ParamsConsoleSave) (map[string]any, error) {
	updates := map[string]any{}
	current := app.Conf().Console.Normalized()
	fields := []struct {
		key   string
		value *string
		limit int
	}{
		{"name", p.Name, config.MaxConsoleNameRunes},
		{"icon", p.Icon, config.MaxConsoleIconBytes},
		{"description", p.Description, config.MaxConsoleDescriptionRunes},
		{"css", p.CSS, config.MaxConsoleCodeSize},
		{"js", p.JS, config.MaxConsoleCodeSize},
	}

	for _, field := range fields {
		if field.value == nil {
			continue
		}
		value := strings.TrimSpace(*field.value)
		if !utf8.ValidString(value) {
			return nil, errConsoleNotUTF8
		}
		switch field.key {
		case "name":
			if utf8.RuneCountInString(value) > field.limit {
				return nil, errConsoleNameTooLong
			}
		case "description":
			if utf8.RuneCountInString(value) > field.limit {
				return nil, errConsoleDescTooLong
			}
		case "icon":
			if len(value) > field.limit || !validConsoleIcon(value) {
				return nil, errConsoleIconInvalid
			}
		default:
			if len(value) > field.limit {
				return nil, errConsoleCodeTooLarge
			}
		}
		updates["console."+field.key] = value
	}

	// 内容没有变化时不写文件，避免无意义的磁盘写入和配置热重载
	for _, key := range []string{"name", "icon", "description", "css", "js"} {
		updateKey := "console." + key
		if v, ok := updates[updateKey]; !ok || v == consoleField(current, key) {
			delete(updates, updateKey)
		}
	}
	return updates, nil
}

func consoleField(conf config.ConsoleConf, key string) string {
	switch key {
	case "name":
		return conf.Name
	case "icon":
		return conf.Icon
	case "description":
		return conf.Description
	case "css":
		return conf.CSS
	default:
		return conf.JS
	}
}

// validConsoleIcon 只接受 http(s) 链接或站内绝对路径：图标会写进 <img src>，
// 排除 data:/javascript: 之类的值，也省得页面里出现来历不明的外链。
func validConsoleIcon(value string) bool {
	if value == "" {
		return true
	}
	if strings.HasPrefix(value, "/") {
		return !strings.HasPrefix(value, "//")
	}
	lower := strings.ToLower(value)
	return strings.HasPrefix(lower, "http://") || strings.HasPrefix(lower, "https://")
}
