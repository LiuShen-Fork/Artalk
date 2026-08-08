package anti_spam

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAICheckerResponses(t *testing.T) {
	var received map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v1/responses", r.URL.Path)
		assert.Equal(t, "Bearer test-key", r.Header.Get("Authorization"))
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
		require.NoError(t, json.NewDecoder(r.Body).Decode(&received))
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"output":[{"content":[{"type":"output_text","text":"{\"sensitive\":false,\"reason\":\"normal discussion\"}"}]}]}`))
	}))
	defer server.Close()

	checker := NewAIChecker(AICheckerConf{
		APIType: AIAPITypeResponses,
		BaseURL: server.URL + "/v1/",
		APIKey:  "test-key",
		Model:   "test-model",
		Prompt:  "classify this comment",
	})
	pass, err := checker.Check(&CheckerParams{ReviewText: "昵称: 张三\n评论: 正常内容"})

	require.NoError(t, err)
	assert.True(t, pass)
	assert.Equal(t, "test-model", received["model"])
	assert.Equal(t, "classify this comment", received["instructions"])
	assert.Equal(t, "昵称: 张三\n评论: 正常内容", received["input"])
	format := received["text"].(map[string]any)["format"].(map[string]any)
	assert.Equal(t, "json_schema", format["type"])
	assert.Equal(t, "comment_moderation", format["name"])
	assert.Equal(t, true, format["strict"])
	assert.Equal(t, false, format["schema"].(map[string]any)["additionalProperties"])
}

func TestAICheckerResponsesTopLevelOutputText(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"output_text":"{\"sensitive\":true,\"reason\":\"spam\"}"}`))
	}))
	defer server.Close()

	checker := NewAIChecker(AICheckerConf{
		APIType: AIAPITypeResponses,
		BaseURL: server.URL + "/v1",
		Model:   "test-model",
	})
	pass, err := checker.Check(&CheckerParams{ReviewText: "nickname: spam"})

	require.NoError(t, err)
	assert.False(t, pass)
}

func TestAICheckerChatCompletionsSensitive(t *testing.T) {
	var received map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v1/chat/completions", r.URL.Path)
		assert.Empty(t, r.Header.Get("Authorization"))
		require.NoError(t, json.NewDecoder(r.Body).Decode(&received))
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"{\"sensitive\":true,\"reason\":\"advertisement\"}"}}]}`))
	}))
	defer server.Close()

	checker := NewAIChecker(AICheckerConf{
		APIType: AIAPITypeChatCompletions,
		BaseURL: server.URL + "/v1",
		Model:   "test-model",
		Prompt:  "system prompt",
	})
	pass, err := checker.Check(&CheckerParams{ReviewText: "昵称: 广告号\n评论: 点击领取"})

	require.NoError(t, err)
	assert.False(t, pass)
	messages := received["messages"].([]any)
	assert.Equal(t, "system", messages[0].(map[string]any)["role"])
	assert.Equal(t, "system prompt", messages[0].(map[string]any)["content"])
	assert.Equal(t, "user", messages[1].(map[string]any)["role"])
	responseFormat := received["response_format"].(map[string]any)
	assert.Equal(t, "json_schema", responseFormat["type"])
	assert.Equal(t, true, responseFormat["json_schema"].(map[string]any)["strict"])
}

func TestAICheckerErrors(t *testing.T) {
	t.Run("base URL must end with v1", func(t *testing.T) {
		checker := NewAIChecker(AICheckerConf{APIType: AIAPITypeResponses, BaseURL: "https://example.com/api", Model: "model"})
		pass, err := checker.Check(&CheckerParams{})
		assert.ErrorContains(t, err, "end with /v1")
		assert.False(t, pass)
	})

	t.Run("model is required", func(t *testing.T) {
		checker := NewAIChecker(AICheckerConf{APIType: AIAPITypeResponses, BaseURL: "https://example.com/v1"})
		pass, err := checker.Check(&CheckerParams{})
		assert.ErrorContains(t, err, "model")
		assert.False(t, pass)
	})

	t.Run("unknown API type", func(t *testing.T) {
		checker := NewAIChecker(AICheckerConf{APIType: "unknown", BaseURL: "https://example.com/v1", Model: "model"})
		pass, err := checker.Check(&CheckerParams{})
		assert.ErrorContains(t, err, "API type")
		assert.False(t, pass)
	})

	tests := []struct {
		name string
		body string
	}{
		{name: "missing sensitive", body: `{"output":[{"content":[{"type":"output_text","text":"{\"reason\":\"missing field\"}"}]}]}`},
		{name: "wrong sensitive type", body: `{"output":[{"content":[{"type":"output_text","text":"{\"sensitive\":\"false\",\"reason\":\"wrong type\"}"}]}]}`},
		{name: "extra result field", body: `{"output":[{"content":[{"type":"output_text","text":"{\"sensitive\":false,\"reason\":\"ok\",\"extra\":1}"}]}]}`},
		{name: "empty output", body: `{"output":[]}`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(tt.body))
			}))
			defer server.Close()

			checker := NewAIChecker(AICheckerConf{APIType: AIAPITypeResponses, BaseURL: server.URL + "/v1", Model: "model"})
			pass, err := checker.Check(&CheckerParams{})
			assert.Error(t, err)
			assert.False(t, pass)
		})
	}

	t.Run("non success HTTP response", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			http.Error(w, "rate limited", http.StatusTooManyRequests)
		}))
		defer server.Close()

		checker := NewAIChecker(AICheckerConf{APIType: AIAPITypeResponses, BaseURL: server.URL + "/v1", Model: "model"})
		pass, err := checker.Check(&CheckerParams{})
		assert.ErrorContains(t, err, "429")
		assert.False(t, pass)
	})
}
