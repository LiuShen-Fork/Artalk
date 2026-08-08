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
)

var _ Checker = (*AIChecker)(nil)

type AIAPIType string

const (
	AIAPITypeResponses       AIAPIType = "responses"
	AIAPITypeChatCompletions AIAPIType = "chat_completions"
)

type AICheckerConf struct {
	APIType AIAPIType
	BaseURL string
	APIKey  string
	Model   string
	Prompt  string
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

	switch c.conf.APIType {
	case AIAPITypeResponses:
		return baseURL + "/responses", nil
	case AIAPITypeChatCompletions:
		return baseURL + "/chat/completions", nil
	default:
		return "", fmt.Errorf("unknown AI API type %q", c.conf.APIType)
	}
}

func (c *AIChecker) requestBody(reviewText string) (map[string]any, error) {
	schema := aiModerationJSONSchema()

	switch c.conf.APIType {
	case AIAPITypeResponses:
		return map[string]any{
			"model":        strings.TrimSpace(c.conf.Model),
			"instructions": c.conf.Prompt,
			"input":        reviewText,
			"text": map[string]any{
				"format": map[string]any{
					"type":   "json_schema",
					"name":   "comment_moderation",
					"strict": true,
					"schema": schema,
				},
			},
		}, nil
	case AIAPITypeChatCompletions:
		return map[string]any{
			"model": strings.TrimSpace(c.conf.Model),
			"messages": []map[string]string{
				{"role": "system", "content": c.conf.Prompt},
				{"role": "user", "content": reviewText},
			},
			"response_format": map[string]any{
				"type": "json_schema",
				"json_schema": map[string]any{
					"name":   "comment_moderation",
					"strict": true,
					"schema": schema,
				},
			},
		}, nil
	default:
		return nil, fmt.Errorf("unknown AI API type %q", c.conf.APIType)
	}
}

func aiModerationJSONSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"sensitive": map[string]any{"type": "boolean"},
			"reason":    map[string]any{"type": "string"},
		},
		"required":             []string{"sensitive", "reason"},
		"additionalProperties": false,
	}
}

func (c *AIChecker) extractResultJSON(responseBody []byte) ([]byte, error) {
	switch c.conf.APIType {
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

	case AIAPITypeChatCompletions:
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

	return result, nil
}
