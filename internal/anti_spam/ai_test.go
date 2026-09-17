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
	pass, err := checker.Check(&CheckerParams{ReviewText: "nickname: Alice\ncomment: normal discussion"})

	require.NoError(t, err)
	assert.True(t, pass)
	assert.Equal(t, "test-model", received["model"])
	input := received["input"].([]any)
	assert.Equal(t, "system", input[0].(map[string]any)["role"])
	assert.Equal(t, "classify this comment", input[0].(map[string]any)["content"])
	assert.Equal(t, "user", input[1].(map[string]any)["role"])
	assert.Equal(t, "nickname: Alice\ncomment: normal discussion", input[1].(map[string]any)["content"])

	format := received["text"].(map[string]any)["format"].(map[string]any)
	assert.Equal(t, "json_schema", format["type"])
	assert.Equal(t, "comment_safety_check", format["name"])
	assert.Equal(t, true, format["strict"])
	assertReasonSchema(t, format["schema"].(map[string]any))
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

func TestAICheckerChatCompletionsJSONSchema(t *testing.T) {
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
	pass, err := checker.Check(&CheckerParams{ReviewText: "nickname: ad\ncomment: click to claim"})

	require.NoError(t, err)
	assert.False(t, pass)
	messages := received["messages"].([]any)
	assert.Equal(t, "system", messages[0].(map[string]any)["role"])
	assert.Equal(t, "system prompt", messages[0].(map[string]any)["content"])
	assert.Equal(t, "user", messages[1].(map[string]any)["role"])

	responseFormat := received["response_format"].(map[string]any)
	assert.Equal(t, "json_schema", responseFormat["type"])
	jsonSchema := responseFormat["json_schema"].(map[string]any)
	assert.Equal(t, "comment_safety_check", jsonSchema["name"])
	assert.Equal(t, true, jsonSchema["strict"])
	assertReasonSchema(t, jsonSchema["schema"].(map[string]any))
}

func TestAICheckerDeepSeekJSONOutput(t *testing.T) {
	var received map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v1/chat/completions", r.URL.Path)
		require.NoError(t, json.NewDecoder(r.Body).Decode(&received))
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"{\"sensitive\":false,\"reason\":\"normal discussion\"}"}}]}`))
	}))
	defer server.Close()

	checker := NewAIChecker(AICheckerConf{
		APIType:         AIAPITypeDeepSeekJSON,
		BaseURL:         server.URL + "/v1",
		Model:           "deepseek-v4-flash",
		Prompt:          "Classify the comment.",
		MaxTokens:       256,
		DisableThinking: true,
	})
	pass, err := checker.Check(&CheckerParams{ReviewText: "nickname: user\ncomment: normal discussion"})

	require.NoError(t, err)
	assert.True(t, pass)
	responseFormat := received["response_format"].(map[string]any)
	assert.Equal(t, "json_object", responseFormat["type"])
	assert.NotContains(t, responseFormat, "json_schema")
	assert.Equal(t, float64(256), received["max_tokens"])
	// DeepSeek keeps thinking on by default, so the effort knob alone would
	// leave it running; the thinking field is what actually switches it off.
	assert.Equal(t, "disabled", received["thinking"].(map[string]any)["type"])
	assert.NotContains(t, received, "reasoning_effort")
	messages := received["messages"].([]any)
	systemPrompt := messages[0].(map[string]any)["content"].(string)
	assert.Contains(t, systemPrompt, "JSON")
	assert.Contains(t, systemPrompt, `{"sensitive": false, "reason": "Non-sensitive technical discussion."}`)
	assert.Contains(t, systemPrompt, "non-empty string")
}

func TestAICheckerChatCompletionsJSONObjectCompatibility(t *testing.T) {
	var received map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.NoError(t, json.NewDecoder(r.Body).Decode(&received))
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"{\"sensitive\":false,\"reason\":\"normal discussion\"}"}}]}`))
	}))
	defer server.Close()

	checker := NewAIChecker(AICheckerConf{
		APIType:      AIAPITypeChatCompletions,
		BaseURL:      server.URL + "/v1",
		Model:        "deepseek-v4-flash",
		OutputFormat: AIOutputFormatJSONObject,
	})
	pass, err := checker.Check(&CheckerParams{ReviewText: "nickname: user\ncomment: normal discussion"})

	require.NoError(t, err)
	assert.True(t, pass)
	responseFormat := received["response_format"].(map[string]any)
	assert.Equal(t, "json_object", responseFormat["type"])
	// DeepSeek rejects json_object unless the conversation mentions JSON.
	messages := received["messages"].([]any)
	assert.Contains(t, messages[0].(map[string]any)["content"].(string), "JSON")
}

func TestAICheckerAnthropicMessages(t *testing.T) {
	var received map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v1/messages", r.URL.Path)
		assert.Equal(t, "test-key", r.Header.Get("x-api-key"))
		assert.Equal(t, "2023-06-01", r.Header.Get("anthropic-version"))
		assert.Empty(t, r.Header.Get("Authorization"))
		require.NoError(t, json.NewDecoder(r.Body).Decode(&received))
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"stop_reason":"end_turn","content":[{"type":"thinking","thinking":"..."},{"type":"text","text":"{\"sensitive\":true,\"reason\":\"advertisement\"}"}]}`))
	}))
	defer server.Close()

	checker := NewAIChecker(AICheckerConf{
		APIType:         AIAPITypeAnthropic,
		BaseURL:         server.URL + "/v1",
		APIKey:          "test-key",
		Model:           "claude-test",
		Prompt:          "classify this comment",
		MaxTokens:       256,
		DisableThinking: true,
	})
	pass, err := checker.Check(&CheckerParams{ReviewText: "nickname: ad\ncomment: click to claim"})

	require.NoError(t, err)
	assert.False(t, pass)
	assert.Equal(t, "classify this comment", received["system"])
	assert.Equal(t, float64(256), received["max_tokens"])
	// Claude rejects the OpenAI name/strict wrapper on output_format.
	outputFormat := received["output_format"].(map[string]any)
	assert.Equal(t, "json_schema", outputFormat["type"])
	assert.NotContains(t, outputFormat, "name")
	assert.NotContains(t, outputFormat, "strict")
	assertReasonSchema(t, outputFormat["schema"].(map[string]any))
	// Thinking is its own switch, separate from the effort knob. It is on by
	// default, so disabling it means sending an explicit "disabled" rather than
	// leaving the field out.
	thinking := received["thinking"].(map[string]any)
	assert.Equal(t, "disabled", thinking["type"])
	assert.NotContains(t, received, "output_config")
	messages := received["messages"].([]any)
	assert.Len(t, messages, 1)
	assert.Equal(t, "user", messages[0].(map[string]any)["role"])
}

func TestAICheckerAnthropicMaxTokensDefault(t *testing.T) {
	checker := NewAIChecker(AICheckerConf{
		APIType: AIAPITypeAnthropic,
		Model:   "claude-test",
	})
	request, err := checker.requestBody("comment")

	require.NoError(t, err)
	// max_tokens is mandatory for the Messages API.
	assert.Equal(t, anthropicDefaultTokens, request["max_tokens"])
}

func TestAICheckerAnthropicJSONObjectUsesPromptHint(t *testing.T) {
	checker := NewAIChecker(AICheckerConf{
		APIType:      AIAPITypeAnthropic,
		Model:        "claude-test",
		Prompt:       "Classify the comment.",
		OutputFormat: AIOutputFormatJSONObject,
	})
	request, err := checker.requestBody("comment")

	require.NoError(t, err)
	// Anthropic has no JSON mode, so the schema must stay absent and the
	// prompt has to carry the JSON contract instead.
	assert.NotContains(t, request, "output_format")
	assert.Contains(t, request["system"].(string), "JSON")
}

func TestAICheckerResponsesRequestOptions(t *testing.T) {
	checker := NewAIChecker(AICheckerConf{
		APIType:         AIAPITypeResponses,
		Model:           "test-model",
		MaxTokens:       128,
		DisableThinking: true,
	})
	request, err := checker.requestBody("comment")

	require.NoError(t, err)
	assert.Equal(t, 128, request["max_output_tokens"])
	// The Responses API folds the thinking toggle into the effort field, where
	// "none" is the documented way to turn thinking off.
	assert.Equal(t, "none", request["reasoning"].(map[string]any)["effort"])
}

// TestAICheckerThinkingToggle pins the thinking switch per protocol. Thinking
// is on by default everywhere, so the effort knob only tunes its depth and a
// real off switch is a separate field.
func TestAICheckerThinkingToggle(t *testing.T) {
	tests := []struct {
		name            string
		apiType         AIAPIType
		disableThinking bool
		assertBody      func(t *testing.T, request map[string]any)
	}{
		{
			name:            "responses disables thinking with effort none",
			apiType:         AIAPITypeResponses,
			disableThinking: true,
			assertBody: func(t *testing.T, request map[string]any) {
				assert.Equal(t, "none", request["reasoning"].(map[string]any)["effort"])
			},
		},
		{
			name:            "responses keeps thinking on with a portable effort",
			apiType:         AIAPITypeResponses,
			disableThinking: false,
			assertBody: func(t *testing.T, request map[string]any) {
				assert.Equal(t, "medium", request["reasoning"].(map[string]any)["effort"])
			},
		},
		{
			name:            "chat completions disables thinking with the thinking field",
			apiType:         AIAPITypeChatCompletions,
			disableThinking: true,
			assertBody: func(t *testing.T, request map[string]any) {
				assert.Equal(t, "disabled", request["thinking"].(map[string]any)["type"])
				assert.NotContains(t, request, "reasoning_effort")
			},
		},
		{
			name:            "chat completions keeps thinking on with a portable effort",
			apiType:         AIAPITypeChatCompletions,
			disableThinking: false,
			assertBody: func(t *testing.T, request map[string]any) {
				assert.NotContains(t, request, "thinking")
				assert.Equal(t, "medium", request["reasoning_effort"])
			},
		},
		{
			name:            "anthropic disables thinking with the thinking field",
			apiType:         AIAPITypeAnthropic,
			disableThinking: true,
			assertBody: func(t *testing.T, request map[string]any) {
				assert.Equal(t, "disabled", request["thinking"].(map[string]any)["type"])
				assert.NotContains(t, request, "output_config")
			},
		},
		{
			name:            "anthropic keeps thinking on with a portable effort",
			apiType:         AIAPITypeAnthropic,
			disableThinking: false,
			assertBody: func(t *testing.T, request map[string]any) {
				assert.NotContains(t, request, "thinking")
				assert.Equal(t, "medium", request["output_config"].(map[string]any)["effort"])
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			checker := NewAIChecker(AICheckerConf{
				APIType:         tt.apiType,
				BaseURL:         "https://example.com/v1",
				Model:           "test-model",
				DisableThinking: tt.disableThinking,
			})
			request, err := checker.requestBody("comment")

			require.NoError(t, err)
			tt.assertBody(t, request)
		})
	}
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

	t.Run("json object is unsupported by responses", func(t *testing.T) {
		checker := NewAIChecker(AICheckerConf{
			APIType:      AIAPITypeResponses,
			BaseURL:      "https://example.com/v1",
			Model:        "model",
			OutputFormat: AIOutputFormatJSONObject,
		})
		pass, err := checker.Check(&CheckerParams{})
		assert.ErrorContains(t, err, "only supported by chat_completions")
		assert.False(t, pass)
	})

	t.Run("negative max tokens", func(t *testing.T) {
		checker := NewAIChecker(AICheckerConf{
			APIType:   AIAPITypeResponses,
			BaseURL:   "https://example.com/v1",
			Model:     "model",
			MaxTokens: -1,
		})
		pass, err := checker.Check(&CheckerParams{})
		assert.ErrorContains(t, err, "max tokens")
		assert.False(t, pass)
	})

	t.Run("anthropic result without text", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"stop_reason":"max_tokens","content":[{"type":"thinking","thinking":"..."}]}`))
		}))
		defer server.Close()

		checker := NewAIChecker(AICheckerConf{
			APIType: AIAPITypeAnthropic,
			BaseURL: server.URL + "/v1",
			Model:   "claude-test",
		})
		pass, err := checker.Check(&CheckerParams{})
		assert.ErrorContains(t, err, "max_tokens")
		assert.False(t, pass)
	})

	tests := []struct {
		name string
		body string
	}{
		{name: "missing sensitive", body: `{"output":[{"content":[{"type":"output_text","text":"{\"reason\":\"missing field\"}"}]}]}`},
		{name: "missing reason", body: `{"output":[{"content":[{"type":"output_text","text":"{\"sensitive\":false}"}]}]}`},
		{name: "empty reason", body: `{"output":[{"content":[{"type":"output_text","text":"{\"sensitive\":false,\"reason\":\"   \"}"}]}]}`},
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

func assertReasonSchema(t *testing.T, schema map[string]any) {
	t.Helper()

	assert.Equal(t, false, schema["additionalProperties"])
	reason := schema["properties"].(map[string]any)["reason"].(map[string]any)
	assert.Equal(t, "string", reason["type"])
	assert.Equal(t, float64(1), reason["minLength"])
}
