package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/artalkjs/artalk/v2/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLskyUploadEndpoint(t *testing.T) {
	endpoint, err := lskyUploadEndpoint("https://img.example.com")
	require.NoError(t, err)
	assert.Equal(t, "https://img.example.com/api/v1/upload", endpoint)

	endpoint, err = lskyUploadEndpoint("https://img.example.com/api/v1/")
	require.NoError(t, err)
	assert.Equal(t, "https://img.example.com/api/v1/upload", endpoint)
}

func TestUploadToLsky(t *testing.T) {
	var gotAuth string
	var gotPermission string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/v1/upload", r.URL.Path)
		gotAuth = r.Header.Get("Authorization")
		require.NoError(t, r.ParseMultipartForm(4<<20))
		gotPermission = r.FormValue("permission")
		file, _, err := r.FormFile("file")
		require.NoError(t, err)
		require.NoError(t, file.Close())
		_ = json.NewEncoder(w).Encode(map[string]any{
			"status": true,
			"data": map[string]any{
				"links": map[string]string{"url": "https://cdn.example.com/a.png"},
			},
		})
	}))
	defer server.Close()

	filePath := filepath.Join(t.TempDir(), "a.png")
	require.NoError(t, os.WriteFile(filePath, []byte("png"), 0o600))

	url, err := uploadToLsky(config.LskyConf{
		Enabled:    true,
		BaseURL:    server.URL,
		Token:      "token",
		Permission: "public",
	}, filePath, "a.png")
	require.NoError(t, err)
	assert.Equal(t, "https://cdn.example.com/a.png", url)
	assert.Equal(t, "Bearer token", gotAuth)
	assert.Equal(t, "1", gotPermission)
}
