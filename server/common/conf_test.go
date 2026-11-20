package common

import (
	"testing"

	artalktest "github.com/artalkjs/artalk/v2/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLocalePluginURLs(t *testing.T) {
	assert.Nil(t, localePluginURLs(""))
	assert.Nil(t, localePluginURLs("en"))
	assert.Nil(t, localePluginURLs("zh-CN"))
	assert.Equal(t, []string{"dist/i18n/ja.js"}, localePluginURLs("ja"))
	// "auto" is decided by the browser, so every external bundle has to be on
	// the page for the widget to pick the right one.
	urls := localePluginURLs("auto")
	assert.Len(t, urls, len(externalLocales))
	assert.Contains(t, urls, "dist/i18n/ja.js")
	assert.Contains(t, urls, "dist/i18n/zh-TW.js")
}

func TestHandlePluginURLs(t *testing.T) {
	app, err := artalktest.NewTestApp()
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, app.Cleanup())
	})

	app.Conf().TrustedDomains = []string{"https://trusted.example"}

	urls := handlePluginURLs(app.App, []string{
		"",
		"dist/plugins/local.js",
		"dist/plugins/local.js",
		"https://trusted.example/plugin.js",
		"https://untrusted.example/plugin.js",
	})

	assert.Equal(t, []string{
		"dist/plugins/local.js",
		"https://trusted.example/plugin.js",
	}, urls)
}
