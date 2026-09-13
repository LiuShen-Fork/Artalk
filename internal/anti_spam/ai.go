package anti_spam

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/artalkjs/artalk/v2/internal/config"
)

var _ Checker = (*AIChecker)(nil)

type AIAPIType string

const (
	AIAPITypeResponses       AIAPIType = "responses"
	AIAPITypeChatCompletions AIAPIType = "chat_completions"
	AIAPITypeDeepSeekJSON    AIAPIType = "deepseek_json_output"
	AIAPITypeAnthropic       AIAPIType = "anthropic_messages"
)

const (
	// reasoningEffortDisabled is used when required thinking is turned off.
	// "none" and "minimal" are model-specific and rejected by most providers,
	// while "medium" is accepted wherever reasoning_effort exists at all.
	reasoningEffortDisabled = "medium"

	anthropicVersion       = "2023-06-01"
	anthropicDefaultTokens = 512

	// aiJSONOutputHint is appended to the prompt whenever the structured schema
	// alone cannot guarantee JSON output. Providers offering json_object mode
	// literally require the prompt to mention JSON, and the Anthropic Messages
	// API treats output_format as an optional hint on many models.
	aiJSONOutputHint = `

Return JSON only. The response must be a single JSON object and must exactly match this shape:
{"sensitive": false, "reason": "Non-sensitive technical discussion."}

Rules for the JSON object:
- Use only the keys "sensitive" and "reason".
- "sensitive" must be a boolean.
- "reason" must be a non-empty string for both sensitive=true and sensitive=false.
- Do not wrap the JSON in Markdown or add any extra text.`
)

type AIOutputFormat string

const (
	AIOutputFormatJSONSchema AIOutputFormat = "json_schema"
	AIOutputFormatJSONObject AIOutputFormat = "json_object"
)

type AICheckerConf struct {
	APIType AIAPIType
	BaseURL string
	APIKey  string
	Model   string
	Prompt  string

	OutputFormat    AIOutputFormat
	MaxTokens       int
	DisableThinking bool
}

type AIChecker struct {
	conf   AICheckerConf
	client *http.Client
}

type aiModerationResult struct {
	Sensitive bool   `json:"sensitive"`
	Reason    string `json:"reason"`
}

func NewAIChecker(conf AICheckerConf) *AIChecker {
	return &AIChecker{
		conf: conf,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

func (*AIChecker) Name() string {
	return "ai"
}

func (c *AIChecker) Check(p *CheckerParams) (bool, error) {
	endpoint, err := c.endpoint()
	if err != nil {
		return false, err
	}

	requestBody, err := c.requestBody(p.ReviewText)
	if err != nil {
		return false, err
	}
	body, err := json.Marshal(requestBody)
	if err != nil {
		return false, fmt.Errorf("marshal AI moderation request: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return false, fmt.Errorf("create AI moderation request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	c.applyAuthHeaders(req)

	resp, err := c.client.Do(req)
	if err != nil {
		return false, fmt.Errorf("request AI moderation API: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if err != nil {
		return false, fmt.Errorf("read AI moderation response: %w", err)
	}
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return false, fmt.Errorf("AI moderation API returned HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(respBody)))
	}

	resultJSON, err := c.extractResultJSON(respBody)
	if err != nil {
		return false, err
	}
	result, err := parseAIModerationResult(resultJSON)
	if err != nil {
		return false, err
	}
	p.ResultReason = result.Reason

	return !result.Sensitive, nil
}

func (c *AIChecker) endpoint() (string, error) {
	baseURL := strings.TrimRight(strings.TrimSpace(c.conf.BaseURL), "/")
	parsed, err := url.Parse(baseURL)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" || (parsed.Scheme != "http" && parsed.Scheme != "https") {
		return "", fmt.Errorf("invalid AI base URL %q", c.conf.BaseURL)
	}
	if parsed.RawQuery != "" || parsed.Fragment != "" {
		return "", fmt.Errorf("AI base URL must not contain query or fragment")
	}
	if !strings.HasSuffix(parsed.Path, "/v1") {
		return "", fmt.Errorf("AI base URL must end with /v1")
	}
	if strings.TrimSpace(c.conf.Model) == "" {
		return "", fmt.Errorf("AI model is required")
	}

	switch c.apiType() {
	case AIAPITypeResponses:
		return baseURL + "/responses", nil
	case AIAPITypeChatCompletions, AIAPITypeDeepSeekJSON:
		return baseURL + "/chat/completions", nil
	case AIAPITypeAnthropic:
		return baseURL + "/messages", nil
	default:
		return "", fmt.Errorf("unknown AI API type %q", c.conf.APIType)
	}
}

// anthropicMessagesRequest reports whether the Anthropic Messages API is used,
// which needs different authentication headers and a mandatory max_tokens.
func (c *AIChecker) anthropicMessagesRequest() bool {
	return c.apiType() == AIAPITypeAnthropic
}

func (c *AIChecker) requestBody(reviewText string) (map[string]any, error) {
	schema := aiModerationJSONSchema()
	apiType := c.apiType()
	if c.conf.MaxTokens < 0 {
		return nil, fmt.Errorf("AI max tokens must not be negative")
	}

	switch apiType {
	case AIAPITypeResponses:
		if c.outputFormat() != AIOutputFormatJSONSchema {
			// The Responses API only supports json_schema.
			return nil, fmt.Errorf("AI output format %q is only supported by chat_completions", c.outputFormat())
		}
		request := map[string]any{
			"model": strings.TrimSpace(c.conf.Model),
			"input": []map[string]string{
				{"role": "system", "content": c.systemPrompt()},
				{"role": "user", "content": reviewText},
			},
			"text": map[string]any{
				"format": map[string]any{
					"type":   "json_schema",
					"name":   "comment_safety_check",
					"strict": true,
					"schema": schema,
				},
			},
		}
		if c.conf.MaxTokens > 0 {
			request["max_output_tokens"] = c.conf.MaxTokens
		}
		if c.conf.DisableThinking {
			// "none" and "minimal" are model-specific and rejected by most
			// providers, so fall back to the lowest widely supported effort.
			request["reasoning"] = map[string]any{"effort": reasoningEffortDisabled}
		}
		return request, nil

	case AIAPITypeAnthropic:
		maxTokens := c.conf.MaxTokens
		if maxTokens <= 0 {
			// max_tokens is mandatory for the Messages API; fall back to a
			// budget that still leaves room for a short JSON answer.
			maxTokens = anthropicDefaultTokens
		}
		request := map[string]any{
			"model":  strings.TrimSpace(c.conf.Model),
			"system": c.systemPrompt(),
			"messages": []map[string]string{
				{"role": "user", "content": reviewText},
			},
			"max_tokens": maxTokens,
		}
		if c.outputFormat() == AIOutputFormatJSONSchema {
			// Structured Outputs use a bare {"type": "json_schema", "schema": {...}}.
			// Anthropic does not accept the OpenAI name/strict wrapper.
			request["output_format"] = map[string]any{
				"type":   "json_schema",
				"schema": schema,
			}
		} else if c.outputFormat() != AIOutputFormatJSONObject {
			return nil, fmt.Errorf("unknown AI output format %q", c.outputFormat())
		}
		// Anthropic thinking is opt-in, so omitting the parameter already means
		// "disabled". Claude-specific thinking configs are rejected by the
		// Anthropic-compatible endpoints of other providers, so never send it.
		return request, nil

	case AIAPITypeChatCompletions, AIAPITypeDeepSeekJSON:
		request := map[string]any{
			"model": strings.TrimSpace(c.conf.Model),
			"messages": []map[string]string{
				{"role": "system", "content": c.systemPrompt()},
				{"role": "user", "content": reviewText},
			},
		}
		if c.usesPlainJSONObjectFormat() {
			// Legacy JSON mode. Providers reject it unless the conversation
			// mentions JSON somewhere, which systemPrompt guarantees.
			request["response_format"] = map[string]any{"type": "json_object"}
		} else if c.outputFormat() == AIOutputFormatJSONSchema {
			request["response_format"] = map[string]any{
				"type": "json_schema",
				"json_schema": map[string]any{
					"name":   "comment_safety_check",
					"strict": true,
					"schema": schema,
				},
			}
		} else {
			return nil, fmt.Errorf("unknown AI output format %q", c.outputFormat())
		}
		if c.conf.MaxTokens > 0 {
			request["max_tokens"] = c.conf.MaxTokens
		}
		// OpenAI-style thinking configs are model-specific and rejected by many
		// OpenAI-compatible providers, so only set the portable effort knob.
		if c.conf.DisableThinking {
			request["reasoning_effort"] = reasoningEffortDisabled
		}
		return request, nil

	default:
		return nil, fmt.Errorf("unknown AI API type %q", c.conf.APIType)
	}
}

func (c *AIChecker) apiType() AIAPIType {
	apiType := AIAPIType(strings.TrimSpace(string(c.conf.APIType)))
	if apiType == "" {
		return AIAPITypeResponses
	}
	return apiType
}

func (c *AIChecker) outputFormat() AIOutputFormat {
	format := AIOutputFormat(strings.TrimSpace(string(c.conf.OutputFormat)))
	if format == "" {
		return AIOutputFormatJSONSchema
	}
	return format
}

// usesPlainJSONObjectFormat reports whether the legacy response_format
// json_object mode should be requested instead of a machine-enforced schema.
func (c *AIChecker) usesPlainJSONObjectFormat() bool {
	return c.apiType() == AIAPITypeDeepSeekJSON ||
		(c.apiType() == AIAPITypeChatCompletions && c.outputFormat() == AIOutputFormatJSONObject)
}

// usesPlainJSONPrompt reports whether the request depends on the model emitting
// JSON by itself instead of a machine-enforced schema. Those providers require
// the prompt to spell out the JSON contract, and json_object providers may even
// reject requests whose prompt never mentions JSON.
func (c *AIChecker) usesPlainJSONPrompt() bool {
	switch c.apiType() {
	case AIAPITypeDeepSeekJSON:
		return true
	case AIAPITypeChatCompletions:
		return c.outputFormat() == AIOutputFormatJSONObject
	case AIAPITypeAnthropic:
		return c.outputFormat() == AIOutputFormatJSONObject
	default:
		return false
	}
}

func (c *AIChecker) systemPrompt() string {
	prompt := strings.TrimSpace(c.conf.Prompt)
	if prompt == "" {
		prompt = config.DefaultAIModerationPrompt
	}
	if !c.usesPlainJSONPrompt() {
		return prompt
	}

	return prompt + aiJSONOutputHint
}

// applyAuthHeaders sets the provider-specific authentication headers. The
// Anthropic Messages API expects x-api-key/anthropic-version instead of the
// OpenAI-style bearer token.
func (c *AIChecker) applyAuthHeaders(req *http.Request) {
	apiKey := strings.TrimSpace(c.conf.APIKey)
	if c.anthropicMessagesRequest() {
		req.Header.Set("anthropic-version", anthropicVersion)
		if apiKey != "" {
			req.Header.Set("x-api-key", apiKey)
		}
		return
	}
	if apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+apiKey)
	}
}

func aiModerationJSONSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"sensitive": map[string]any{
				"type":        "boolean",
				"description": "Whether the nickname or comment contains sensitive content.",
			},
			"reason": map[string]any{
				"type":        "string",
				"description": "A short non-empty reason for the decision.",
				"minLength":   1,
			},
		},
		"required":             []string{"sensitive", "reason"},
		"additionalProperties": false,
	}
}

func (c *AIChecker) extractResultJSON(responseBody []byte) ([]byte, error) {
	switch c.apiType() {
	case AIAPITypeResponses:
		var response struct {
			OutputText string `json:"output_text"`
			Output     []struct {
				Content []struct {
					Type string `json:"type"`
					Text string `json:"text"`
				} `json:"content"`
			} `json:"output"`
		}
		if err := json.Unmarshal(responseBody, &response); err != nil {
			return nil, fmt.Errorf("decode AI responses result: %w", err)
		}
		if strings.TrimSpace(response.OutputText) != "" {
			return []byte(response.OutputText), nil
		}
		for _, output := range response.Output {
			for _, content := range output.Content {
				if (content.Type == "output_text" || content.Type == "text") && strings.TrimSpace(content.Text) != "" {
					return []byte(content.Text), nil
				}
			}
		}
		return nil, fmt.Errorf("AI responses result contains no output_text")

	case AIAPITypeChatCompletions, AIAPITypeDeepSeekJSON:
		var response struct {
			Choices []struct {
				Message struct {
					Content string `json:"content"`
					Refusal string `json:"refusal"`
				} `json:"message"`
			} `json:"choices"`
		}
		if err := json.Unmarshal(responseBody, &response); err != nil {
			return nil, fmt.Errorf("decode AI chat completions result: %w", err)
		}
		if len(response.Choices) == 0 {
			return nil, fmt.Errorf("AI chat completions result contains no choices")
		}
		if strings.TrimSpace(response.Choices[0].Message.Refusal) != "" {
			return nil, fmt.Errorf("AI chat completions request was refused: %s", response.Choices[0].Message.Refusal)
		}
		content := strings.TrimSpace(response.Choices[0].Message.Content)
		if content == "" {
			return nil, fmt.Errorf("AI chat completions result contains no message content")
		}
		return []byte(content), nil

	case AIAPITypeAnthropic:
		var response struct {
			Content []struct {
				Type string `json:"type"`
				Text string `json:"text"`
			} `json:"content"`
			StopReason string `json:"stop_reason"`
		}
		if err := json.Unmarshal(responseBody, &response); err != nil {
			return nil, fmt.Errorf("decode AI anthropic result: %w", err)
		}
		for _, content := range response.Content {
			if content.Type == "text" && strings.TrimSpace(content.Text) != "" {
				return []byte(content.Text), nil
			}
		}
		if response.StopReason == "max_tokens" {
			// Likely spent the whole budget on thinking before emitting JSON.
			return nil, fmt.Errorf("AI anthropic result contains no text (stop_reason=max_tokens, raise max_tokens)")
		}
		return nil, fmt.Errorf("AI anthropic result contains no text content")

	default:
		return nil, fmt.Errorf("unknown AI API type %q", c.conf.APIType)
	}
}

func parseAIModerationResult(data []byte) (aiModerationResult, error) {
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(data, &fields); err != nil {
		return aiModerationResult{}, fmt.Errorf("decode AI moderation JSON: %w", err)
	}
	if len(fields) != 2 {
		return aiModerationResult{}, fmt.Errorf("AI moderation JSON must contain exactly sensitive and reason")
	}

	sensitiveJSON, hasSensitive := fields["sensitive"]
	reasonJSON, hasReason := fields["reason"]
	if !hasSensitive || !hasReason {
		return aiModerationResult{}, fmt.Errorf("AI moderation JSON must contain sensitive and reason")
	}

	var result aiModerationResult
	if err := json.Unmarshal(sensitiveJSON, &result.Sensitive); err != nil {
		return aiModerationResult{}, fmt.Errorf("AI moderation sensitive must be boolean: %w", err)
	}
	if err := json.Unmarshal(reasonJSON, &result.Reason); err != nil {
		return aiModerationResult{}, fmt.Errorf("AI moderation reason must be string: %w", err)
	}
	if strings.TrimSpace(result.Reason) == "" {
		return aiModerationResult{}, fmt.Errorf("AI moderation reason must not be empty")
	}

	return result, nil
}
