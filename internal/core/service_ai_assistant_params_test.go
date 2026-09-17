package core

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/artalkjs/artalk/v2/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestAssistantRequestParameters pins the wire format of each API protocol so
// model-specific options (thinking/reasoning effort) never leak into providers
// that reject them.
func TestAssistantRequestParameters(t *testing.T) {
	tests := []struct {
		name            string
		apiType         config.AIAPIType
		disableThinking bool
		expectedURL     string
		assertBody      func(t *testing.T, body map[string]any)
	}{
		{
			name:            "responses disables thinking with effort none",
			apiType:         config.AIAPITypeResponses,
			disableThinking: true,
			expectedURL:     "/v1/responses",
			assertBody: func(t *testing.T, body map[string]any) {
				assert.Equal(t, "none", body["reasoning"].(map[string]any)["effort"])
				assert.NotContains(t, body, "thinking")
			},
		},
		{
			name:        "responses keeps thinking on with a portable effort",
			apiType:     config.AIAPITypeResponses,
			expectedURL: "/v1/responses",
			assertBody: func(t *testing.T, body map[string]any) {
				assert.Equal(t, "medium", body["reasoning"].(map[string]any)["effort"])
			},
		},
		{
			name:            "chat completions disables thinking with the thinking field",
			apiType:         config.AIAPITypeChatCompletions,
			disableThinking: true,
			expectedURL:     "/v1/chat/completions",
			assertBody: func(t *testing.T, body map[string]any) {
				assert.Equal(t, "disabled", body["thinking"].(map[string]any)["type"])
				assert.NotContains(t, body, "reasoning_effort")
			},
		},
		{
			name:        "chat completions keeps thinking on with a portable effort",
			apiType:     config.AIAPITypeChatCompletions,
			expectedURL: "/v1/chat/completions",
			assertBody: func(t *testing.T, body map[string]any) {
				assert.NotContains(t, body, "thinking")
				assert.Equal(t, "medium", body["reasoning_effort"])
			},
		},
		{
			name:            "anthropic disables thinking with the thinking field",
			apiType:         config.AIAPITypeAnthropic,
			disableThinking: true,
			expectedURL:     "/v1/messages",
			assertBody: func(t *testing.T, body map[string]any) {
				assert.Equal(t, "disabled", body["thinking"].(map[string]any)["type"])
				assert.NotContains(t, body, "output_config")
				assert.Contains(t, body, "system")
				assert.Equal(t, float64(512), body["max_tokens"])
			},
		},
		{
			name:        "anthropic keeps thinking on with a portable effort",
			apiType:     config.AIAPITypeAnthropic,
			expectedURL: "/v1/messages",
			assertBody: func(t *testing.T, body map[string]any) {
				assert.NotContains(t, body, "thinking")
				assert.Equal(t, "medium", body["output_config"].(map[string]any)["effort"])
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var body map[string]any
			var path string
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				path = r.URL.Path
				require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
				w.Header().Set("Content-Type", "application/json")
				switch tt.apiType {
				case config.AIAPITypeAnthropic:
					_, _ = w.Write([]byte(`{"content":[{"type":"text","text":"回复"}]}`))
				case config.AIAPITypeResponses:
					_, _ = w.Write([]byte(`{"output":[{"content":[{"type":"output_text","text":"回复"}]}]}`))
				default:
					_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"回复"}}]}`))
				}
			}))
			defer server.Close()

			disableThinking := tt.disableThinking
			// Provide the client directly so the request helper does not need a
			// fully wired App only to read the timeout config.
			service := &AIAssistantService{client: server.Client()}
			text, err := service.request("问题", config.AIAssistantConf{
				APIType:         tt.apiType,
				BaseURL:         server.URL + "/v1",
				Model:           "test-model",
				MaxTokens:       512,
				DisableThinking: &disableThinking,
			})

			require.NoError(t, err)
			assert.Equal(t, "回复", text)
			assert.Equal(t, tt.expectedURL, path)
			tt.assertBody(t, body)
		})
	}
}

// TestAssistantAnthropicAuthHeaders verifies the Anthropic Messages API uses
// x-api-key/anthropic-version instead of the OpenAI-style bearer token.
func TestAssistantAnthropicAuthHeaders(t *testing.T) {
	var header http.Header
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		header = r.Header.Clone()
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"content":[{"type":"text","text":"回复"}]}`))
	}))
	defer server.Close()

	service := &AIAssistantService{client: server.Client()}
	_, err := service.request("问题", config.AIAssistantConf{
		APIType: config.AIAPITypeAnthropic,
		BaseURL: server.URL + "/v1",
		APIKey:  "test-key",
		Model:   "claude-test",
	})

	require.NoError(t, err)
	assert.Equal(t, "test-key", header.Get("x-api-key"))
	assert.Equal(t, "2023-06-01", header.Get("anthropic-version"))
	assert.Empty(t, header.Get("Authorization"))
}
