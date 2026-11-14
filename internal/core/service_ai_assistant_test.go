package core

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/artalkjs/artalk/v2/internal/config"
	"github.com/artalkjs/artalk/v2/internal/entity"
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
		{name: "anthropic messages", apiType: config.AIAPITypeAnthropic, path: "/messages", expected: config.AIAPITypeAnthropic},
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

func TestCommentTargetsAssistant(t *testing.T) {
	trigger := assistantTrigger(config.AIAssistantConf{Name: "小助手"})

	assert.True(t, commentTargetsAssistant("请回答 @小助手", trigger, entity.User{}))
	assert.False(t, commentTargetsAssistant("普通评论", trigger, entity.User{}))
	assert.True(t, commentTargetsAssistant("普通评论", trigger, entity.User{IsAIAssistant: true}))
}

func TestLegacyAssistantIdentityMatching(t *testing.T) {
	conf := config.AIAssistantConf{Name: "小助手", Email: "AI@example.com", Link: "https://example.com"}
	matching := entity.User{Name: "小助手", Email: "ai@EXAMPLE.com", Link: "https://example.com"}
	matching.ID = 1
	nonMatching := entity.User{Name: "小助手", Email: "other@example.com", Link: "https://example.com"}
	nonMatching.ID = 2
	assert.True(t, legacyAssistantIdentityMatches(matching, conf))
	assert.False(t, legacyAssistantIdentityMatches(nonMatching, conf))
	assert.False(t, legacyAssistantIdentityMatches(entity.User{IsAIAssistant: true}, conf))
}

func TestCollectAncestorComments(t *testing.T) {
	root := entity.Comment{Rid: 0, Content: "root"}
	root.ID = 1
	parent := entity.Comment{Rid: 1, Content: "parent"}
	parent.ID = 2
	currentParent := entity.Comment{Rid: 2, Content: "current parent"}
	currentParent.ID = 3
	comments := map[uint]entity.Comment{1: root, 2: parent, 3: currentParent}
	trigger := entity.Comment{Rid: 3}
	trigger.ID = 4
	context := collectAncestorComments(&trigger, 20, func(id uint) entity.Comment {
		return comments[id]
	})

	require.Len(t, context, 3)
	assert.Equal(t, []uint{1, 2, 3}, []uint{context[0].ID, context[1].ID, context[2].ID})
	limited := collectAncestorComments(&trigger, 2, func(id uint) entity.Comment {
		return comments[id]
	})
	require.Len(t, limited, 2)
	assert.Equal(t, []uint{2, 3}, []uint{limited[0].ID, limited[1].ID})
}

func TestAssistantPromptKeepsPagePrefixBeforeConversation(t *testing.T) {
	service := (*AIAssistantService)(nil)
	prompt := service.buildAssistantPrompt("@小助手", entity.Page{Title: "页面标题"}, "https://example.com", "稳定正文", nil, "当前问题")

	assert.Less(t, strings.Index(prompt, "稳定正文"), strings.Index(prompt, "当前问题"))
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

	anthropicBody := []byte(`{"content":[{"type":"text","text":"Claude 回复"}]}`)
	text, err = extractAssistantText(config.AIAPITypeAnthropic, anthropicBody)
	require.NoError(t, err)
	assert.Equal(t, "Claude 回复", text)

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
