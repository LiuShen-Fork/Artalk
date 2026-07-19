package auth

import (
	"strings"
	"testing"

	"github.com/artalkjs/artalk/v2/internal/auth/generic"
	"github.com/artalkjs/artalk/v2/internal/config"
	"github.com/artalkjs/artalk/v2/internal/utils"
	"github.com/markbates/goth"
)

func genericAuthConfig() *config.Config {
	conf := &config.Config{}
	conf.Auth.Callback = "https://artalk.example.com/api/v2/auth/{provider}/callback"
	conf.Auth.Generic = config.AuthGenericOAuthConf{
		Enabled:      true,
		Label:        "Company Login",
		ClientID:     "client-id",
		ClientSecret: "client-secret",
		AuthorizeURL: "https://id.example.com/oauth/authorize",
		TokenURL:     "https://id.example.com/oauth/token",
		UserInfoURL:  "https://id.example.com/api/user",
		Scopes:       []string{"profile", "email"},
	}
	return conf
}

func TestGetProvidersIncludesGenericOAuth(t *testing.T) {
	providers := GetProviders(genericAuthConfig())
	if len(providers) != 1 {
		t.Fatalf("provider count = %d, want 1", len(providers))
	}
	if providers[0].Name() != generic.ProviderName {
		t.Fatalf("provider name = %q", providers[0].Name())
	}

	session, err := providers[0].BeginAuth("state-value")
	if err != nil {
		t.Fatal(err)
	}
	authURL, err := session.GetAuthURL()
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(authURL, "redirect_uri=https%3A%2F%2Fartalk.example.com%2Fapi%2Fv2%2Fauth%2Fgeneric%2Fcallback") {
		t.Fatalf("authorization URL has unexpected callback: %s", authURL)
	}
}

func TestGetProvidersSkipsInvalidGenericOAuth(t *testing.T) {
	conf := genericAuthConfig()
	conf.Auth.Generic.TokenURL = ""
	if providers := GetProviders(conf); len(providers) != 0 {
		t.Fatalf("provider count = %d, want 0", len(providers))
	}
}

func TestGetProviderInfoUsesGenericLabelAndIcon(t *testing.T) {
	conf := genericAuthConfig()
	providers := GetProviders(conf)
	info := GetProviderInfo(conf, providers)
	if len(info) != 1 {
		t.Fatalf("provider info count = %d", len(info))
	}
	if info[0].Name != generic.ProviderName || info[0].Label != "Company Login" {
		t.Fatalf("provider info = %#v", info[0])
	}
	if info[0].Path != "/api/v2/auth/generic" {
		t.Fatalf("provider path = %q", info[0].Path)
	}
	if !strings.HasPrefix(info[0].Icon, "data:image/svg+xml;base64,") {
		t.Fatalf("provider icon is missing")
	}
}

func TestGetProviderInfoUsesCustomEmailLabel(t *testing.T) {
	conf := &config.Config{}
	conf.Auth.Email.Enabled = true
	conf.Auth.Email.Label = "邮箱密码登录"

	info := GetProviderInfo(conf, nil)
	if len(info) != 1 {
		t.Fatalf("provider info count = %d", len(info))
	}
	if info[0].Name != "email" || info[0].Label != "邮箱密码登录" {
		t.Fatalf("provider info = %#v", info[0])
	}
}

func TestGetProviderInfoUsesDefaultEmailLabel(t *testing.T) {
	conf := &config.Config{}
	conf.Auth.Email.Enabled = true
	conf.Auth.Email.Label = "  "

	info := GetProviderInfo(conf, nil)
	if len(info) != 1 {
		t.Fatalf("provider info count = %d", len(info))
	}
	if info[0].Label != "Email" {
		t.Fatalf("email label = %q, want Email", info[0].Label)
	}
}

func TestGetSocialUserUsesEmailLocalPartAsName(t *testing.T) {
	socialUser := GetSocialUser(goth.User{
		Provider: generic.ProviderName,
		UserID:   "user-123",
		Email:    "alice@example.com",
	})
	if socialUser.Name != "alice" {
		t.Fatalf("Name = %q, want email local part", socialUser.Name)
	}
}

func TestGetSocialUserMapsGenericLinkAndSafeFallbackEmail(t *testing.T) {
	socialUser := GetSocialUser(goth.User{
		Provider: generic.ProviderName,
		UserID:   "tenant|user:123",
		Name:     "Alice",
		RawData: map[string]any{
			generic.RawDataUserLinkKey: "https://example.com/users/alice",
		},
	})
	if socialUser.Link != "https://example.com/users/alice" {
		t.Fatalf("Link = %q", socialUser.Link)
	}
	if !strings.HasPrefix(socialUser.Email, "oauth-") || !strings.HasSuffix(socialUser.Email, "@generic.invalid") {
		t.Fatalf("fallback email = %q", socialUser.Email)
	}
	if !utils.ValidateEmail(socialUser.Email) {
		t.Fatalf("fallback email is invalid: %q", socialUser.Email)
	}
}
