package handler

import (
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/artalkjs/artalk/v2/internal/core"
	"github.com/artalkjs/artalk/v2/internal/entity"
	"github.com/artalkjs/artalk/v2/server/common"
	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

type CookedUserIdentity struct {
	ID          uint      `json:"id"`
	Provider    string    `json:"provider"`
	CreatedAt   time.Time `json:"created_at"`
	ConfirmedAt time.Time `json:"confirmed_at"`
}

type ResponseUserIdentities struct {
	Identities []CookedUserIdentity `json:"identities"`
}

// @Id           GetUserIdentities
// @Summary      List linked login identities
// @Description  List the OAuth identities linked to the current account
// @Tags         User
// @Security     ApiKeyAuth
// @Success      200  {object}  ResponseUserIdentities
// @Failure      401  {object}  Map{msg=string}
// @Router       /user/identities  [get]
func UserIdentityList(app *core.App, router fiber.Router) {
	router.Get("/user/identities", common.LoginGuard(app, func(c *fiber.Ctx, user entity.User) error {
		var identities []entity.AuthIdentity
		if err := app.Dao().DB().Where("user_id = ?", user.ID).Order("provider ASC, id ASC").Find(&identities).Error; err != nil {
			return common.RespError(c, 500, "Failed to list identities")
		}

		result := make([]CookedUserIdentity, 0, len(identities))
		for _, identity := range identities {
			item := CookedUserIdentity{
				ID:        identity.ID,
				Provider:  identity.Provider,
				CreatedAt: identity.CreatedAt,
			}
			if identity.ConfirmedAt != nil {
				item.ConfirmedAt = *identity.ConfirmedAt
			}
			result = append(result, item)
		}

		return common.RespData(c, ResponseUserIdentities{Identities: result})
	}))
}

type ResponseUserIdentityBindURL struct {
	Path string `json:"path"`
}

// @Id           BeginUserIdentityBind
// @Summary      Start an OAuth identity binding
// @Description  Returns a short-lived OAuth URL that links the provider account to the current user
// @Tags         User
// @Security     ApiKeyAuth
// @Success      200  {object}  ResponseUserIdentityBindURL
// @Failure      400  {object}  Map{msg=string}
// @Failure      401  {object}  Map{msg=string}
// @Router       /user/identities/:provider/bind  [post]
func UserIdentityBind(app *core.App, router fiber.Router) {
	router.Post("/user/identities/:provider/bind", common.LoginGuard(app, func(c *fiber.Ctx, user entity.User) error {
		if !app.Conf().Auth.Enabled {
			return common.RespError(c, 404, "Auth Api disabled")
		}

		provider := strings.ToLower(strings.TrimSpace(c.Params("provider")))
		if !isEnabledOAuthProvider(app, provider) {
			return common.RespError(c, 404, "Provider is not enabled")
		}

		state, err := newIdentityBindState(app, user.ID, provider)
		if err != nil {
			return common.RespError(c, 500, "Failed to create bind request")
		}

		return common.RespData(c, ResponseUserIdentityBindURL{
			Path: fmt.Sprintf("/auth/%s?state=%s", provider, url.QueryEscape(state)),
		})
	}))
}

// @Id           RemoveUserIdentity
// @Summary      Unlink a login identity
// @Description  Unlink one OAuth identity from the current account. The last
//
//	remaining login method cannot be removed.
//
// @Tags         User
// @Security     ApiKeyAuth
// @Param        id  path  int  true  "The identity ID"
// @Success      200  {object}  Map{msg=string}
// @Failure      400  {object}  Map{msg=string}
// @Failure      401  {object}  Map{msg=string}
// @Failure      404  {object}  Map{msg=string}
// @Router       /user/identities/{id}  [delete]
func UserIdentityUnbind(app *core.App, router fiber.Router) {
	router.Delete("/user/identities/:id", common.LoginGuard(app, func(c *fiber.Ctx, user entity.User) error {
		id, err := c.ParamsInt("id")
		if err != nil || id <= 0 {
			return common.RespError(c, 400, "Invalid identity ID")
		}

		var identity entity.AuthIdentity
		err = app.Dao().DB().Where("id = ? AND user_id = ?", uint(id), user.ID).First(&identity).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return common.RespError(c, 404, "Identity not found")
		}
		if err != nil {
			return common.RespError(c, 500, "Failed to find identity")
		}

		var identityCount int64
		if err := app.Dao().DB().Model(&entity.AuthIdentity{}).Where("user_id = ?", user.ID).Count(&identityCount).Error; err != nil {
			return common.RespError(c, 500, "Failed to check login methods")
		}
		if strings.TrimSpace(user.Password) == "" && identityCount <= 1 {
			return common.RespError(c, 400, "Cannot remove the last login method")
		}

		if err := app.Dao().DelAuthIdentity(&identity); err != nil {
			return common.RespError(c, 500, "Failed to remove identity")
		}

		return common.RespData(c, Map{"msg": "Identity removed"})
	}))
}
