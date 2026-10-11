package handler_test

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http/httptest"
	"testing"

	"github.com/artalkjs/artalk/v2/server/common"
	"github.com/artalkjs/artalk/v2/server/handler"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestUserInfoUpdateNormalizesLink covers the profile form. A homepage typed
// without a scheme has to be stored with one: every endpoint that saves the
// link later refuses a value ValidateURL rejects, so an unusable link only
// surfaces afterwards as a 400 on some unrelated save such as approving a
// comment.
func TestUserInfoUpdateNormalizesLink(t *testing.T) {
	tests := []struct {
		name     string
		link     string
		wantLink string
	}{
		{
			name:     "bare host gains a scheme",
			link:     "myhome.example.org",
			wantLink: "https://myhome.example.org",
		},
		{
			name:     "valid link is kept as typed",
			link:     "https://myhome.example.org/a?b=1",
			wantLink: "https://myhome.example.org/a?b=1",
		},
		{
			name:     "unusable link is dropped instead of stored",
			link:     "javascript:alert(1)",
			wantLink: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app, fiberApp := NewApiTestApp()
			defer app.Cleanup()

			handler.UserInfoUpdate(app.App, fiberApp)

			user := app.Dao().FindUserByID(1001)
			require.False(t, user.IsEmpty(), "fixture user 1001 is missing")

			token, err := common.LoginGetUserToken(user, app.Conf().AppKey, 3600)
			require.NoError(t, err)

			// The email is sent unchanged so the handler does not ask for a
			// verification code, which this test is not about.
			payload, err := json.Marshal(map[string]string{
				"email": user.Email,
				"name":  user.Name,
				"link":  tt.link,
			})
			require.NoError(t, err)

			req := httptest.NewRequest("POST", "/user?token="+token, bytes.NewReader(payload))
			req.Header.Set("Content-Type", "application/json")
			resp, err := fiberApp.Test(req)
			require.NoError(t, err)

			body, err := io.ReadAll(resp.Body)
			require.NoError(t, err)
			require.Equal(t, 200, resp.StatusCode, string(body))

			assert.Equal(t, tt.wantLink, app.Dao().FindUserByID(1001).Link)
		})
	}
}
