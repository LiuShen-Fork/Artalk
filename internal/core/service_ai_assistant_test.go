package core

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/artalkjs/artalk/v2/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAssistantAPITypeAndEndpoint(t *testing.T) {
	tests := []struct {
		name     string
		apiType  config.AIAPIType
		path     string
		expected config.AIAPIType
	}{
		{name: "default", path: "/responses", expected: config.AIAPITypeResponses},
		{name: "responses", apiType: config.AIAPITypeResponses, path: "/responses", expected: config.AIAPITypeResponses},
		{name: "chat completions", apiType: config.AIAPITypeChatCompletions, path: "/chat/completions", expected: config.AIAPITypeChatCompletions},
		{name: "deepseek", apiType: config.AIAPITypeDeepSeekJSON, path: "/chat/completions", expected: config.AIAPITypeDeepSeekJSON},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			conf := config.AIAssistantConf{APIType: tt.apiType, BaseURL: "https://example.com/v1", Model: "model"}
			apiType, err := assistantAPIType(conf)
			require.NoError(t, err)
			assert.Equal(t, tt.expected, apiType)
			endpoint, err := assistantEndpoint(conf)
			require.NoError(t, err)
			assert.Equal(t, "https://example.com/v1"+tt.path, endpoint)
		})
	}

	_, err := assistantEndpoint(config.AIAssistantConf{
		APIType: config.AIAPIType("unsupported"), BaseURL: "https://example.com/v1", Model: "model",
	})
	assert.ErrorContains(t, err, "unknown ai_assistant api_type")
}

func TestAssistantTriggerUsesFixedAtPrefixAndName(t *testing.T) {
	assert.Equal(t, "@清羽酱", assistantTrigger(config.AIAssistantConf{}))
	assert.Equal(t, "@小助手", assistantTrigger(config.AIAssistantConf{Name: "小助手"}))
	assert.Equal(t, "@小助手", assistantTrigger(config.AIAssistantConf{Name: "@小助手"}))
}

func TestExtractAssistantText(t *testing.T) {
	responsesBody := []byte(`{"output":[{"content":[{"type":"output_text","text":"响应内容"}]}]}`)
	text, err := extractAssistantText(config.AIAPITypeResponses, responsesBody)
	require.NoError(t, err)
	assert.Equal(t, "响应内容", text)

	chatBody := []byte(`{"choices":[{"message":{"content":"聊天回复"}}]}`)
	text, err = extractAssistantText(config.AIAPITypeChatCompletions, chatBody)
	require.NoError(t, err)
	assert.Equal(t, "聊天回复", text)

	_, err = extractAssistantText(config.AIAPITypeChatCompletions, []byte(`{"choices":[]}`))
	assert.ErrorContains(t, err, "no choices")
}

func TestFetchPageTextAndLimitRunes(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`<html><head><style>hidden</style></head><body><h1>页面标题</h1><script>ignored()</script><p>正文内容</p></body></html>`))
	}))
	defer server.Close()

	text, err := fetchPageText(server.Client(), server.URL, 100)
	require.NoError(t, err)
	assert.Contains(t, text, "页面标题")
	assert.Contains(t, text, "正文内容")
	assert.NotContains(t, text, "ignored")
	assert.Equal(t, "你好世", limitRunes("你好世界", 3))
}
