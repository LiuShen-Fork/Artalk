package generic

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"
)

func validOptions(baseURL string) Options {
	return Options{
		ClientID:       "client-id",
		ClientSecret:   "client-secret",
		CallbackURL:    "https://artalk.example.com/api/v2/auth/generic/callback",
		AuthorizeURL:   baseURL + "/authorize",
		TokenURL:       baseURL + "/token",
		UserInfoURL:    baseURL + "/userinfo",
		Scopes:         []string{" openid ", "profile", "email", "email", ""},
		UserIDPath:     "data.account.id",
		UserNamePath:   "data.account.name",
		UserEmailPath:  "data.account.email",
		UserAvatarPath: "data.account.avatar.url",
		UserLinkPath:   "data.account.profile_url",
	}
}

func TestNewValidatesRequiredOptions(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*Options)
		want   string
	}{
		{name: "client ID", mutate: func(o *Options) { o.ClientID = "" }, want: "client ID is required"},
		{name: "client secret", mutate: func(o *Options) { o.ClientSecret = "" }, want: "client secret is required"},
		{name: "user ID path", mutate: func(o *Options) { o.UserIDPath = "" }, want: "user ID path is required"},
		{name: "authorize URL", mutate: func(o *Options) { o.AuthorizeURL = "ftp://example.com/auth" }, want: "invalid authorize URL"},
		{name: "token URL", mutate: func(o *Options) { o.TokenURL = "/token" }, want: "invalid token URL"},
		{name: "user info URL", mutate: func(o *Options) { o.UserInfoURL = "" }, want: "invalid user info URL"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			options := validOptions("https://provider.example.com")
			tt.mutate(&options)
			_, err := New(options)
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("New() error = %v, want error containing %q", err, tt.want)
			}
		})
	}
}

func TestBeginAuthBuildsAuthorizationURL(t *testing.T) {
	provider, err := New(validOptions("https://provider.example.com"))
	if err != nil {
		t.Fatal(err)
	}

	session, err := provider.BeginAuth("state-value")
	if err != nil {
		t.Fatal(err)
	}
	authURL, err := session.GetAuthURL()
	if err != nil {
		t.Fatal(err)
	}
	parsed, err := url.Parse(authURL)
	if err != nil {
		t.Fatal(err)
	}

	query := parsed.Query()
	if got := parsed.String(); !strings.HasPrefix(got, "https://provider.example.com/authorize?") {
		t.Fatalf("authorization URL = %q", got)
	}
	assertQueryValue(t, query, "client_id", "client-id")
	assertQueryValue(t, query, "redirect_uri", "https://artalk.example.com/api/v2/auth/generic/callback")
	assertQueryValue(t, query, "response_type", "code")
	assertQueryValue(t, query, "state", "state-value")
	assertQueryValue(t, query, "scope", "openid profile email")
}

func assertQueryValue(t *testing.T, values url.Values, key, want string) {
	t.Helper()
	if got := values.Get(key); got != want {
		t.Fatalf("query %s = %q, want %q", key, got, want)
	}
}

func TestAuthorizeAndFetchUser(t *testing.T) {
	var tokenRequestSeen bool
	var userInfoRequestSeen bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/token":
			tokenRequestSeen = true
			if err := r.ParseForm(); err != nil {
				t.Errorf("parse token form: %v", err)
				return
			}
			if got := r.Form.Get("code"); got != "authorization-code" {
				t.Errorf("token code = %q", got)
			}
			if got := r.Form.Get("redirect_uri"); got != "https://artalk.example.com/api/v2/auth/generic/callback" {
				t.Errorf("token redirect_uri = %q", got)
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"access_token":  "access-token",
				"token_type":    "Bearer",
				"refresh_token": "refresh-token",
				"expires_in":    3600,
				"id_token":      "id-token",
			})
		case "/userinfo":
			userInfoRequestSeen = true
			if got := r.Header.Get("Authorization"); got != "Bearer access-token" {
				t.Errorf("Authorization = %q", got)
			}
			if got := r.Header.Get("Accept"); got != "application/json" {
				t.Errorf("Accept = %q", got)
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"data":{"account":{"id":9007199254740993,"name":"Alice","email":"alice@example.com","avatar":{"url":"https://cdn.example.com/alice.png"},"profile_url":"https://example.com/@alice"}}}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	provider, err := New(validOptions(server.URL))
	if err != nil {
		t.Fatal(err)
	}
	sessionValue, err := provider.BeginAuth("state-value")
	if err != nil {
		t.Fatal(err)
	}
	session := sessionValue.(*Session)
	if _, err := session.Authorize(provider, url.Values{"code": {"authorization-code"}}); err != nil {
		t.Fatal(err)
	}
	if !tokenRequestSeen {
		t.Fatal("token endpoint was not called")
	}
	if session.IDToken != "id-token" {
		t.Fatalf("ID token = %q", session.IDToken)
	}

	user, err := provider.FetchUser(session)
	if err != nil {
		t.Fatal(err)
	}
	if !userInfoRequestSeen {
		t.Fatal("user-info endpoint was not called")
	}
	if user.Provider != ProviderName {
		t.Errorf("Provider = %q", user.Provider)
	}
	if user.UserID != "9007199254740993" {
		t.Errorf("UserID = %q", user.UserID)
	}
	if user.Name != "Alice" || user.NickName != "Alice" {
		t.Errorf("Name/NickName = %q/%q", user.Name, user.NickName)
	}
	if user.Email != "alice@example.com" {
		t.Errorf("Email = %q", user.Email)
	}
	if user.AvatarURL != "https://cdn.example.com/alice.png" {
		t.Errorf("AvatarURL = %q", user.AvatarURL)
	}
	if user.RawData[RawDataUserLinkKey] != "https://example.com/@alice" {
		t.Errorf("mapped link = %#v", user.RawData[RawDataUserLinkKey])
	}
	if user.AccessToken != "access-token" || user.RefreshToken != "refresh-token" || user.IDToken != "id-token" {
		t.Errorf("tokens were not copied to goth user")
	}
	if time.Until(user.ExpiresAt) < 59*time.Minute {
		t.Errorf("ExpiresAt = %v", user.ExpiresAt)
	}
}

func TestFetchUserNameFallbackDoesNotExposeEmail(t *testing.T) {
	tests := []struct {
		name     string
		response string
		wantName string
	}{
		{name: "email remains private", response: `{"id":"user-1","email":"fallback@example.com"}`, wantName: ""},
		{name: "ID", response: `{"id":"user-2"}`, wantName: "user-2"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				_, _ = w.Write([]byte(tt.response))
			}))
			defer server.Close()
			options := validOptions(server.URL)
			options.UserIDPath = "id"
			options.UserNamePath = "name"
			options.UserEmailPath = "email"
			provider, err := New(options)
			if err != nil {
				t.Fatal(err)
			}
			user, err := provider.FetchUser(&Session{AccessToken: "token"})
			if err != nil {
				t.Fatal(err)
			}
			if user.Name != tt.wantName {
				t.Fatalf("Name = %q, want %q", user.Name, tt.wantName)
			}
		})
	}
}

func TestFetchUserErrors(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		body       string
		want       string
	}{
		{name: "non success", statusCode: http.StatusUnauthorized, body: `{}`, want: "status 401"},
		{name: "invalid JSON", statusCode: http.StatusOK, body: `{`, want: "decode generic user information"},
		{name: "multiple JSON values", statusCode: http.StatusOK, body: `{"data":{}} {}`, want: "multiple JSON values"},
		{name: "missing ID", statusCode: http.StatusOK, body: `{"data":{"account":{"name":"Alice"}}}`, want: "does not contain a value at ID path"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.statusCode)
				_, _ = w.Write([]byte(tt.body))
			}))
			defer server.Close()
			provider, err := New(validOptions(server.URL))
			if err != nil {
				t.Fatal(err)
			}
			_, err = provider.FetchUser(&Session{AccessToken: "token"})
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("FetchUser() error = %v, want containing %q", err, tt.want)
			}
		})
	}
}

func TestFetchUserRejectsOversizedResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"data":{"account":{"id":"user-1","padding":"`))
		_, _ = w.Write([]byte(strings.Repeat("x", maxUserInfoBodySize)))
		_, _ = w.Write([]byte(`"}}}`))
	}))
	defer server.Close()

	provider, err := New(validOptions(server.URL))
	if err != nil {
		t.Fatal(err)
	}
	_, err = provider.FetchUser(&Session{AccessToken: "token"})
	if err == nil || !strings.Contains(err.Error(), "response exceeds") {
		t.Fatalf("FetchUser() error = %v", err)
	}
}

func TestSessionMarshalRoundTrip(t *testing.T) {
	provider, err := New(validOptions("https://provider.example.com"))
	if err != nil {
		t.Fatal(err)
	}
	original := Session{AuthURL: "https://provider.example.com/authorize?state=x", AccessToken: "token", TokenType: "Bearer"}
	decodedValue, err := provider.UnmarshalSession(original.Marshal())
	if err != nil {
		t.Fatal(err)
	}
	decoded := decodedValue.(*Session)
	if decoded.AuthURL != original.AuthURL || decoded.AccessToken != original.AccessToken || decoded.TokenType != original.TokenType {
		t.Fatalf("decoded session = %#v", decoded)
	}
}
