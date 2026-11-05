package handler

import (
	"errors"
	"strings"

	"github.com/artalkjs/artalk/v2/internal/auth"
	"github.com/artalkjs/artalk/v2/internal/auth/gothic_fiber"
	"github.com/artalkjs/artalk/v2/internal/core"
	"github.com/artalkjs/artalk/v2/internal/log"
	"github.com/artalkjs/artalk/v2/server/common"
	"github.com/gofiber/fiber/v2"
	"github.com/markbates/goth"
)

func SocialLoginGuard(app *core.App, handler fiber.Handler) fiber.Handler {
	return func(c *fiber.Ctx) error {
		if !app.Conf().Auth.Enabled {
			return common.RespError(c, 404, "Auth Api disabled")
		}

		return handler(c)
	}
}

type ResponseConfAuthProviders struct {
	CallbackOrigin string                  `json:"callback_origin,omitempty"`
	Providers      []auth.AuthProviderInfo `json:"providers" validator:"required"`
	Anonymous      bool                    `json:"anonymous" validator:"required"`
}

// @Id            GetSocialLoginProviders
// @Summary       Get Social Login Providers
// @Description   Get social login providers
// @Tags          System
// @Produce       json
// @Success       200  {object}  ResponseConfAuthProviders
// @Success       404  {object}  Map{msg=string}
// @Router        /conf/auth/providers  [get]
func AuthSocialLogin(app *core.App, router fiber.Router) {
	// Load providers
	var providers []goth.Provider

	loadProviders := func() {
		providers = auth.GetProviders(app.Conf())
		goth.ClearProviders()
		goth.UseProviders(providers...)
	}

	loadProviders()
	app.OnConfUpdated().Add(func(e *core.ConfUpdatedEvent) error {
		loadProviders()
		return nil
	})

	// Endpoints
	router.Get("/conf/auth/providers", SocialLoginGuard(app, func(c *fiber.Ctx) error {
		return common.RespData(c, ResponseConfAuthProviders{
			CallbackOrigin: auth.CallbackOrigin(app.Conf()),
			Providers:      auth.GetProviderInfo(app.Conf(), providers),
			Anonymous:      app.Conf().Auth.Anonymous,
		})
	}))

	router.Get("/auth/:provider", SocialLoginGuard(app, func(c *fiber.Ctx) error {
		return gothic_fiber.BeginAuthHandler(c)
	}))

	router.Get("/auth/:provider/callback", SocialLoginGuard(app, func(c *fiber.Ctx) error {
		provider, err := gothic_fiber.GetProviderName(c)
		if err != nil {
			log.Error("[SocialLogin] ", err)
			return common.RespError(c, 500, "Field to get provider name")
		}

		// A signed state token marks an explicit "link this provider to my
		// current account" request. It must be resolved before the ordinary
		// login path, which would otherwise create or select a user from the
		// provider email.
		bindClaims, isBind, bindStateErr := parseIdentityBindState(app, c.Query("state"))
		bindFail := func(message string) error {
			return auth.ResponseBindCallbackPage(c, provider, false, message)
		}
		if isBind {
			if bindStateErr != nil {
				return bindFail(bindStateErr.Error())
			}
			if !strings.EqualFold(bindClaims.Provider, provider) {
				return bindFail("The provider does not match this bind request")
			}
			if app.Dao().FindUserByID(bindClaims.UserID).IsEmpty() {
				return bindFail("The account no longer exists")
			}
		}

		// Fetch user
		gothUser, err := gothic_fiber.CompleteUserAuth(c)
		if err != nil {
			log.Error("[SocialLogin] ", err)
			if isBind {
				return auth.ResponseBindCallbackPage(c, provider, false, "Authentication was cancelled or failed")
			}
			return common.RespError(c, 500, "Field to complete user auth")
		}

		// Convert to social user
		socialUser, err := auth.GetSocialUser(gothUser, app.Conf().Auth.Gitea.SiteURL)
		if err != nil {
			if isBind {
				return auth.ResponseBindCallbackPage(c, provider, false, "The provider did not return a usable account")
			}
			return common.RespError(c, 500, err.Error())
		}
		log.Debug("[SocialLogin] ", socialUser)

		if isBind {
			err := BindSocialIdentity(app, bindClaims.UserID, provider, socialUser)
			if errors.Is(err, ErrIdentityBoundToOtherUser) {
				return bindFail("This provider account is already linked to another Artalk account")
			}
			if errors.Is(err, ErrProviderAlreadyBound) {
				return bindFail("This Artalk account already has a different binding for this provider")
			}
			if err != nil {
				log.Error("[SocialLogin] bind identity error: ", err)
				return bindFail("Failed to link this provider account")
			}

			return auth.ResponseBindCallbackPage(c, provider, true, "This provider is now linked to your account")
		}

		// Find auth identity
		authIdentity := app.Dao().FindAuthIdentityByRemoteUID(provider, socialUser.RemoteUID)

		// No auth identity record, register user
		if authIdentity.IsEmpty() {
			authIdentity, err = auth.RegisterSocialUser(app.Dao(), socialUser)
			if err != nil {
				log.Error("[SocialLogin] ", err)
				return common.RespError(c, 500, "Failed to register user")
			}
		}
		if authIdentity.UserID == 0 {
			return common.RespError(c, 500, "Auth Identity user_id invalid")
		}

		// Find user perform login
		user := app.Dao().FindUserByID(authIdentity.UserID)
		if user.IsEmpty() {
			return common.RespError(c, 500, "Failed to find user")
		}

		// Get user token
		jwtToken, err := common.LoginGetUserToken(user, app.Conf().AppKey, app.Conf().LoginTimeout)
		if err != nil {
			return common.RespError(c, 500, err.Error())
		}

		// Render response
		return auth.ResponseCallbackPage(c, jwtToken)
	}))
}
