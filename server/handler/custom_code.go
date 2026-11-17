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
	errCustomCodeTooLarge     = errors.New("自定义代码超出长度上限")
	errCustomCodeNotUTF8      = errors.New("自定义代码必须是 UTF-8 文本")
	errCustomCodeUnavailable  = errors.New("配置来自环境变量或内存，无法写入自定义代码")
)

type ResponseCustomCode struct {
	CSS string `json:"css"`
	JS  string `json:"js"`
}

type ParamsCustomCodeSave struct {
	CSS *string `json:"css" validate:"optional"`
	JS  *string `json:"js" validate:"optional"`
}

// CustomCodeGet 读取控制台的自定义 CSS / JS
//
// 由控制台在启动时拉取并注入页面。内容是管理员维护的静态文本，不含任何用户数据，
// 因此无需鉴权；登录页同样会应用它，保证换肤后登录界面风格一致。
//
// @Id           CustomCodeGet
// @Summary      获取控制台自定义样式与脚本
// @Tags         Custom
// @Produce      json
// @Success      200  {object}  ResponseCustomCode
// @Router       /sidebar/custom [get]
func CustomCodeGet(app *core.App, router fiber.Router) {
	router.Get("/sidebar/custom", func(c *fiber.Ctx) error {
		return common.RespData(c, buildCustomCodeResponse(app))
	})
}

// CustomCodeSave 保存控制台的自定义 CSS / JS
//
// 只改动 `custom` 段，配置文件中的注释与其他配置保持不变，保存后立即生效（不重启服务）。
// 脚本会在控制台页面内执行，请只填写自己信任的代码。
//
// @Id           CustomCodeSave
// @Summary      保存控制台自定义样式与脚本
// @Tags         Custom
// @Security     ApiKeyAuth
// @Accept       json
// @Produce      json
// @Param        data  body  ParamsCustomCodeSave  true  "自定义代码"
// @Success      200  {object}  ResponseCustomCode
// @Router       /sidebar/custom [put]
func CustomCodeSave(app *core.App, router fiber.Router) {
	router.Put("/sidebar/custom", common.AdminGuard(app, func(c *fiber.Ctx) error {
		var p ParamsCustomCodeSave
		if isOK, resp := common.ParamsDecode(c, &p); !isOK {
			return resp
		}

		updates, err := customCodeUpdates(app, &p)
		if err != nil {
			return common.RespError(c, fiber.StatusBadRequest, err.Error())
		}

		if len(updates) > 0 {
			newConf, err := writeConfigFile(app, updates)
			if err != nil {
				return common.RespError(c, fiber.StatusInternalServerError, errCustomCodeUnavailable.Error()+": "+err.Error())
			}
			app.SetConf(newConf)
			log.Info("[Custom] console code updated")
		}

		return common.RespData(c, buildCustomCodeResponse(app))
	}))
}

func buildCustomCodeResponse(app *core.App) ResponseCustomCode {
	custom := app.Conf().Custom.Normalized()
	return ResponseCustomCode{CSS: custom.CSS, JS: custom.JS}
}

func customCodeUpdates(app *core.App, p *ParamsCustomCodeSave) (map[string]any, error) {
	updates := map[string]any{}
	current := app.Conf().Custom

	for field, value := range map[string]*string{"css": p.CSS, "js": p.JS} {
		if value == nil {
			continue
		}
		snippet := *value
		if !utf8.ValidString(snippet) {
			return nil, errCustomCodeNotUTF8
		}
		if len(snippet) > config.MaxCustomCodeSize {
			return nil, errCustomCodeTooLarge
		}
		updates["custom."+field] = strings.TrimSpace(snippet)
	}

	// 内容没有变化时不写文件，避免无意义的磁盘写入和配置热重载
	if v, ok := updates["custom.css"]; ok && v == current.CSS {
		delete(updates, "custom.css")
	}
	if v, ok := updates["custom.js"]; ok && v == current.JS {
		delete(updates, "custom.js")
	}
	return updates, nil
}
