package handler

import (
	"strings"
	"time"

	"github.com/artalkjs/artalk/v2/internal/core"
	"github.com/artalkjs/artalk/v2/internal/entity"
	"github.com/artalkjs/artalk/v2/internal/i18n"
	"github.com/artalkjs/artalk/v2/server/common"
	"github.com/gofiber/fiber/v2"
)

type RequestUserPasswordUpdate struct {
	CurrentPassword string `json:"current_password" validate:"optional"`
	NewPassword     string `json:"new_password" validate:"required"`
	Code            string `json:"code" validate:"optional"`
}

type ResponseUserPasswordUpdate struct {
	Token string            `json:"token"`
	User  entity.CookedUser `json:"user"`
}

// @Id           UpdatePassword
// @Summary      Change the current user's password
// @Description  Requires the current password, or an email verification code
//
//	when the account has no password yet.
//
// @Tags         Auth
// @Security     ApiKeyAuth
// @Param        data  body  RequestUserPasswordUpdate  true  "The password data"
// @Success      200  {object}  ResponseUserPasswordUpdate
// @Failure      400  {object}  Map{msg=string}
// @Failure      401  {object}  Map{msg=string}
// @Router       /user/password  [post]
func UserPasswordUpdate(app *core.App, router fiber.Router) {
	router.Post("/user/password", common.LoginGuard(app, func(c *fiber.Ctx, user entity.User) error {
		var p RequestUserPasswordUpdate
		if ok, resp := common.ParamsDecode(c, &p); !ok {
			return resp
		}

		p.Code = strings.TrimSpace(p.Code)
		if len(p.NewPassword) < 8 {
			return common.RespError(c, 400, "Password must be at least 8 characters")
		}

		hasPassword := strings.TrimSpace(user.Password) != ""
		switch {
		case hasPassword:
			if p.CurrentPassword == "" {
				return common.RespError(c, 400, "Current password is required")
			}
			if !user.CheckPassword(p.CurrentPassword) {
				return common.RespError(c, 401, i18n.T("Password is incorrect"))
			}
		default:
			if !app.Conf().Auth.Email.Enabled {
				return common.RespError(c, 400, "Email verification is not enabled")
			}
			if p.Code == "" {
				return common.RespError(c, 400, "Email verification code is required")
			}
			if ok, resp := CheckEmailVerifyCode(app, c, user.Email, p.Code); !ok {
				return resp
			}
		}

		if err := user.SetPasswordEncrypt(p.NewPassword); err != nil {
			return common.RespError(c, 500, "Failed to update password")
		}
		// JWT iat has second precision. Move the validity boundary to the next
		// second and mint the replacement token at exactly that second, so even
		// a token issued moments before this request is revoked.
		validFrom := time.Now().Truncate(time.Second).Add(time.Second)
		user.TokenValidFrom.Scan(validFrom)
		if err := app.Dao().UpdateUser(&user); err != nil {
			return common.RespError(c, 500, "Failed to update password")
		}

		token, err := common.LoginGetUserTokenAt(user, app.Conf().AppKey, app.Conf().LoginTimeout, validFrom)
		if err != nil {
			return common.RespError(c, 500, "Failed to create session")
		}

		return common.RespData(c, ResponseUserPasswordUpdate{
			Token: token,
			User:  app.Dao().CookUser(&user),
		})
	}))
}
