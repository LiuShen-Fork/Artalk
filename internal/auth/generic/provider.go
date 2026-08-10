package generic

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/markbates/goth"
	"golang.org/x/oauth2"
)

const (
	ProviderName        = "generic"
	RawDataUserLinkKey  = "_artalk_user_link"
	defaultHTTPTimeout  = 20 * time.Second
	maxUserInfoBodySize = 4 << 20
)

// Options describes a standard OAuth 2.0 Authorization Code provider.
type Options struct {
	ClientID     string
	ClientSecret string
	CallbackURL  string
	AuthorizeURL string
	TokenURL     string
	UserInfoURL  string
	Scopes       []string
}

// Provider implements goth.Provider for a configurable OAuth 2.0 service.
type Provider struct {
	userInfoURL  string
	providerName string
	config       *oauth2.Config
	HTTPClient   *http.Client
}

var commonUserInfoContainers = []string{"", "user", "data", "data.user", "account", "data.account", "profile", "data.profile"}

var _ goth.Provider = (*Provider)(nil)

// New validates the supplied endpoints and creates a generic OAuth provider.
func New(options Options) (*Provider, error) {
	options.ClientID = strings.TrimSpace(options.ClientID)
	options.CallbackURL = strings.TrimSpace(options.CallbackURL)
	options.AuthorizeURL = strings.TrimSpace(options.AuthorizeURL)
	options.TokenURL = strings.TrimSpace(options.TokenURL)
	options.UserInfoURL = strings.TrimSpace(options.UserInfoURL)

	if options.ClientID == "" {
		return nil, errors.New("client ID is required")
	}
	if strings.TrimSpace(options.ClientSecret) == "" {
		return nil, errors.New("client secret is required")
	}
	for _, endpoint := range []struct {
		name string
		url  string
	}{
		{name: "callback URL", url: options.CallbackURL},
		{name: "authorize URL", url: options.AuthorizeURL},
		{name: "token URL", url: options.TokenURL},
		{name: "user info URL", url: options.UserInfoURL},
	} {
		if err := validateHTTPURL(endpoint.url); err != nil {
			return nil, fmt.Errorf("invalid %s: %w", endpoint.name, err)
		}
	}

	scopes := normalizeScopes(options.Scopes)
	provider := &Provider{
		userInfoURL:  options.UserInfoURL,
		providerName: ProviderName,
		HTTPClient:   &http.Client{Timeout: defaultHTTPTimeout},
	}
	provider.config = &oauth2.Config{
		ClientID:     options.ClientID,
		ClientSecret: options.ClientSecret,
		RedirectURL:  options.CallbackURL,
		Endpoint: oauth2.Endpoint{
			AuthURL:  options.AuthorizeURL,
			TokenURL: options.TokenURL,
		},
		Scopes: scopes,
	}

	return provider, nil
}

func validateHTTPURL(rawURL string) error {
	if rawURL == "" {
		return errors.New("value is required")
	}
	u, err := url.Parse(rawURL)
	if err != nil {
		return err
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return errors.New("scheme must be http or https")
	}
	if u.Host == "" {
		return errors.New("host is required")
	}
	return nil
}

func normalizeScopes(scopes []string) []string {
	result := make([]string, 0, len(scopes))
	seen := make(map[string]struct{}, len(scopes))
	for _, scope := range scopes {
		scope = strings.TrimSpace(scope)
		if scope == "" {
			continue
		}
		if _, ok := seen[scope]; ok {
			continue
		}
		seen[scope] = struct{}{}
		result = append(result, scope)
	}
	return result
}

func (p *Provider) Name() string {
	return p.providerName
}

func (p *Provider) SetName(name string) {
	p.providerName = name
}

func (p *Provider) Debug(bool) {}

func (p *Provider) Client() *http.Client {
	return goth.HTTPClientWithFallBack(p.HTTPClient)
}

func (p *Provider) BeginAuth(state string) (goth.Session, error) {
	return &Session{AuthURL: p.config.AuthCodeURL(state)}, nil
}

func (p *Provider) UnmarshalSession(data string) (goth.Session, error) {
	session := &Session{}
	err := json.NewDecoder(strings.NewReader(data)).Decode(session)
	return session, err
}

func (p *Provider) FetchUser(session goth.Session) (goth.User, error) {
	sess, ok := session.(*Session)
	if !ok {
		return goth.User{}, fmt.Errorf("%s received an invalid session type", p.providerName)
	}

	user := goth.User{
		Provider:     p.Name(),
		AccessToken:  sess.AccessToken,
		RefreshToken: sess.RefreshToken,
		ExpiresAt:    sess.ExpiresAt,
		IDToken:      sess.IDToken,
	}
	if sess.AccessToken == "" {
		return user, fmt.Errorf("%s cannot get user information without an access token", p.providerName)
	}

	req, err := http.NewRequest(http.MethodGet, p.userInfoURL, nil)
	if err != nil {
		return user, err
	}
	req.Header.Set("Accept", "application/json")
	(&oauth2.Token{AccessToken: sess.AccessToken, TokenType: sess.TokenType}).SetAuthHeader(req)

	response, err := p.Client().Do(req)
	if err != nil {
		if response != nil && response.Body != nil {
			response.Body.Close()
		}
		return user, err
	}
	defer response.Body.Close()

	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return user, fmt.Errorf("%s responded with status %d while fetching user information", p.providerName, response.StatusCode)
	}

	body, err := io.ReadAll(io.LimitReader(response.Body, maxUserInfoBodySize+1))
	if err != nil {
		return user, err
	}
	if len(body) > maxUserInfoBodySize {
		return user, fmt.Errorf("%s user information response exceeds %d bytes", p.providerName, maxUserInfoBodySize)
	}

	decoder := json.NewDecoder(bytes.NewReader(body))
	decoder.UseNumber()
	var rawData map[string]any
	if err := decoder.Decode(&rawData); err != nil {
		return user, fmt.Errorf("decode %s user information: %w", p.providerName, err)
	}
	if err := ensureJSONEOF(decoder); err != nil {
		return user, fmt.Errorf("decode %s user information: %w", p.providerName, err)
	}

	userID := commonUserInfoValue(rawData, "id", "sub", "user_id", "uid", "uuid")
	if userID == "" {
		return user, fmt.Errorf("%s user information does not contain a recognized user ID field", p.providerName)
	}

	user.UserID = userID
	user.Name = commonUserInfoValue(rawData, "name", "display_name", "full_name", "username", "login", "nickname", "nick_name", "preferred_username")
	user.Email = commonUserInfoValue(rawData, "email", "mail", "email_address")
	user.AvatarURL = commonUserInfoValue(rawData, "avatar_url", "avatar.url", "avatar", "picture", "profile_image_url_https", "profile_image_url")
	if link := commonUserInfoValue(rawData, "html_url", "profile_url", "web_url", "website", "url"); link != "" {
		rawData[RawDataUserLinkKey] = link
	}
	if user.Name == "" && user.Email == "" {
		user.Name = user.UserID
	}
	user.NickName = user.Name
	user.RawData = rawData

	return user, nil
}

func commonUserInfoValue(data map[string]any, fields ...string) string {
	for _, container := range commonUserInfoContainers {
		for _, field := range fields {
			path := field
			if container != "" {
				path = container + "." + field
			}
			if value := valueAtPath(data, path); value != "" {
				return value
			}
		}
	}
	return ""
}

func ensureJSONEOF(decoder *json.Decoder) error {
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		if err == nil {
			return errors.New("response contains multiple JSON values")
		}
		return err
	}
	return nil
}

func valueAtPath(data any, path string) string {
	path = strings.TrimSpace(path)
	if path == "" {
		return ""
	}

	current := data
	for _, part := range strings.Split(path, ".") {
		part = strings.TrimSpace(part)
		if part == "" {
			return ""
		}
		switch value := current.(type) {
		case map[string]any:
			var ok bool
			current, ok = value[part]
			if !ok {
				return ""
			}
		case []any:
			index, err := strconv.Atoi(part)
			if err != nil || index < 0 || index >= len(value) {
				return ""
			}
			current = value[index]
		default:
			return ""
		}
	}

	switch value := current.(type) {
	case string:
		return strings.TrimSpace(value)
	case json.Number:
		return value.String()
	case float64:
		return strconv.FormatFloat(value, 'f', -1, 64)
	case float32:
		return strconv.FormatFloat(float64(value), 'f', -1, 32)
	case int:
		return strconv.Itoa(value)
	case int64:
		return strconv.FormatInt(value, 10)
	case int32:
		return strconv.FormatInt(int64(value), 10)
	case uint:
		return strconv.FormatUint(uint64(value), 10)
	case uint64:
		return strconv.FormatUint(value, 10)
	case uint32:
		return strconv.FormatUint(uint64(value), 10)
	default:
		return ""
	}
}

func (p *Provider) RefreshTokenAvailable() bool {
	return true
}

func (p *Provider) RefreshToken(refreshToken string) (*oauth2.Token, error) {
	if strings.TrimSpace(refreshToken) == "" {
		return nil, errors.New("refresh token is required")
	}
	token := &oauth2.Token{RefreshToken: refreshToken}
	return p.config.TokenSource(goth.ContextForClient(p.Client()), token).Token()
}
