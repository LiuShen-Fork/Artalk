package core

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFetchPageTextSelectors(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`<main id="article"><p>正文</p><figure>图片说明</figure><div class="link-card app-card">链接卡片</div><div class="keep">保留</div></main><aside>侧栏</aside>`))
	}))
	defer server.Close()

	text, err := fetchPageTextWithSelectors(server.Client(), server.URL, 100, "main#article", []string{"figure", "#article .link-card.app-card"})
	require.NoError(t, err)
	assert.Contains(t, text, "正文")
	assert.Contains(t, text, "保留")
	assert.NotContains(t, text, "图片说明")
	assert.NotContains(t, text, "链接卡片")

	text, err = fetchPageTextWithSelectors(server.Client(), server.URL, 100, "article#missing", nil)
	require.NoError(t, err)
	assert.Empty(t, text)
}
