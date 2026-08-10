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
	if apiKey := strings.TrimSpace(c.conf.APIKey); apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+apiKey)
	}

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
	default:
		return "", fmt.Errorf("unknown AI API type %q", c.conf.APIType)
	}
}

func (c *AIChecker) requestBody(reviewText string) (map[string]any, error) {
	schema := aiModerationJSONSchema()
	apiType := c.apiType()
	if c.conf.MaxTokens < 0 {
		return nil, fmt.Errorf("AI max tokens must not be negative")
	}

	switch apiType {
	case AIAPITypeResponses:
		if c.outputFormat() == AIOutputFormatJSONObject {
			return nil, fmt.Errorf("AI output format %q is only supported by chat_completions", c.outputFormat())
		} else if c.outputFormat() != AIOutputFormatJSONSchema {
			return nil, fmt.Errorf("unknown AI output format %q", c.outputFormat())
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
			request["reasoning"] = map[string]any{"effort": "none"}
		}
		return request, nil
	case AIAPITypeChatCompletions, AIAPITypeDeepSeekJSON:
		request := map[string]any{
			"model": strings.TrimSpace(c.conf.Model),
			"messages": []map[string]string{
				{"role": "system", "content": c.systemPrompt()},
				{"role": "user", "content": reviewText},
			},
		}
		if c.usesDeepSeekJSONOutput() {
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
		if c.conf.DisableThinking {
			request["thinking"] = map[string]any{"type": "disabled"}
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

func (c *AIChecker) usesDeepSeekJSONOutput() bool {
	return c.apiType() == AIAPITypeDeepSeekJSON ||
		(c.apiType() == AIAPITypeChatCompletions && c.outputFormat() == AIOutputFormatJSONObject)
}

func (c *AIChecker) systemPrompt() string {
	prompt := strings.TrimSpace(c.conf.Prompt)
	if prompt == "" {
		prompt = config.DefaultAIModerationPrompt
	}
	if !c.usesDeepSeekJSONOutput() {
		return prompt
	}

	return prompt + `

Return JSON only. The response must be a single JSON object and must exactly match this shape:
{"sensitive": false, "reason": "Non-sensitive technical discussion."}

Rules for the JSON object:
- Use only the keys "sensitive" and "reason".
- "sensitive" must be a boolean.
- "reason" must be a non-empty string for both sensitive=true and sensitive=false.
- Do not wrap the JSON in Markdown or add any extra text.`
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
