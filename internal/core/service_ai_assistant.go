package core

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/artalkjs/artalk/v2/internal/config"
	"github.com/artalkjs/artalk/v2/internal/entity"
	"github.com/artalkjs/artalk/v2/internal/log"
	"golang.org/x/net/html"
)

var _ Service = (*AIAssistantService)(nil)

const aiAssistantLogRetention = 90 * 24 * time.Hour

type AIAssistantService struct {
	app    *App
	client *http.Client
	mu     sync.Mutex
	userID uint
}

func NewAIAssistantService(app *App) *AIAssistantService {
	return &AIAssistantService{app: app}
}

func (s *AIAssistantService) Init() error {
	s.client = &http.Client{Timeout: s.timeout()}
	s.pruneLogs()
	return nil
}

func (s *AIAssistantService) Dispose() error {
	s.client = nil
	s.userID = 0
	return nil
}

// ReplyToComment handles a matching comment after the normal comment jobs.
// It is intentionally synchronous within the caller's background job so a
// failed request can be recorded with the triggering comment ID.
func (s *AIAssistantService) ReplyToComment(comment *entity.Comment) {
	if comment == nil {
		return
	}
	conf := s.app.Conf().AIAssistant
	trigger := assistantTrigger(conf)
	if !conf.Enabled || trigger == "" || !strings.Contains(comment.Content, trigger) {
		return
	}

	if err := s.reply(comment, conf, trigger); err != nil {
		log.Errorf("[AIAssistant] comment=%d failed: %v", comment.ID, err)
		s.record(comment, nil, trigger, entity.AIAssistantLogStatusError, "", err.Error())
	}
}

func (s *AIAssistantService) reply(comment *entity.Comment, conf config.AIAssistantConf, trigger string) error {
	latest := s.app.Dao().FindComment(comment.ID)
	if latest.IsEmpty() {
		return fmt.Errorf("trigger comment %d not found", comment.ID)
	}
	if latest.IsPending && !conf.ReplyToPending {
		return nil
	}

	assistant, err := s.assistantUser(conf)
	if err != nil {
		return fmt.Errorf("prepare assistant user: %w", err)
	}
	if latest.UserID == assistant.ID {
		return nil
	}
	var existing entity.Comment
	result := s.app.Dao().DB().Where("rid = ? AND user_id = ?", latest.ID, assistant.ID).
		Order("created_at DESC").Limit(1).Find(&existing)
	if result.Error != nil {
		return fmt.Errorf("check existing assistant reply: %w", result.Error)
	}
	if existing.ID != 0 {
		return nil
	}

	page := s.app.Dao().FindPage(latest.PageKey, latest.SiteName)
	pageURL := s.app.Dao().GetPageAccessibleURL(&page)
	pageText, err := fetchPageTextWithSelectors(s.client, pageURL, conf.MaxPageChars, conf.ContentSelector, conf.ExcludeSelectors)
	if err != nil {
		return fmt.Errorf("fetch page context: %w", err)
	}

	comments := s.parentCommentContext(&latest)
	prompt := s.buildAssistantPrompt(trigger, page, pageURL, pageText, comments, latest.Content)
	response, err := s.request(prompt, conf)
	if err != nil {
		return err
	}
	maxReplyChars := conf.MaxReplyChars
	if maxReplyChars <= 0 || maxReplyChars > 300 {
		maxReplyChars = 300
	}
	response = limitRunes(strings.TrimSpace(response), maxReplyChars)
	if response == "" {
		return fmt.Errorf("AI assistant returned an empty response")
	}

	reply := entity.Comment{
		Content:     response,
		PageKey:     latest.PageKey,
		SiteName:    latest.SiteName,
		UserID:      assistant.ID,
		Rid:         latest.ID,
		RootID:      s.app.Dao().FindCommentRootID(latest.ID),
		IsVerified:  true,
		IsPending:   false,
		IsCollapsed: false,
	}
	if err := s.app.Dao().CreateComment(&reply); err != nil {
		return fmt.Errorf("save assistant reply: %w", err)
	}
	if notifyService, err := AppService[*NotifyService](s.app); err == nil {
		if err := notifyService.Push(&reply, &latest); err != nil {
			return fmt.Errorf("notify assistant reply: %w", err)
		}
	} else {
		return fmt.Errorf("get notify service: %w", err)
	}
	s.record(&latest, &reply, trigger, entity.AIAssistantLogStatusSuccess, response, "")
	return nil
}

func (s *AIAssistantService) assistantUser(conf config.AIAssistantConf) (entity.User, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.userID != 0 {
		user := s.app.Dao().FindUserByID(s.userID)
		if !user.IsEmpty() {
			return user, nil
		}
		s.userID = 0
	}
	name := assistantName(conf)
	email := strings.TrimSpace(conf.Email)
	if name == "" || email == "" {
		return entity.User{}, fmt.Errorf("ai_assistant name and email are required")
	}
	user, err := s.app.Dao().FindCreateUser(name, email, conf.Link)
	if err != nil {
		return entity.User{}, err
	}
	if !user.IsAIAssistant {
		user.IsAIAssistant = true
		if err := s.app.Dao().UpdateUser(&user); err != nil {
			return entity.User{}, fmt.Errorf("mark assistant user: %w", err)
		}
	}
	s.userID = user.ID
	return user, nil
}

func assistantName(conf config.AIAssistantConf) string {
	name := strings.TrimSpace(conf.Name)
	name = strings.TrimLeft(name, "@")
	if name == "" {
		name = "清羽酱"
	}
	return name
}

func assistantTrigger(conf config.AIAssistantConf) string {
	name := assistantName(conf)
	if name == "" {
		return ""
	}
	return "@" + name
}

func (s *AIAssistantService) parentCommentContext(trigger *entity.Comment) []entity.Comment {
	if trigger == nil || trigger.Rid == 0 {
		return nil
	}
	parent := s.app.Dao().FindComment(trigger.Rid)
	if parent.IsEmpty() {
		return nil
	}
	return []entity.Comment{parent}
}

func (s *AIAssistantService) request(prompt string, conf config.AIAssistantConf) (string, error) {
	endpoint, err := assistantEndpoint(conf)
	if err != nil {
		return "", err
	}
	apiType, err := assistantAPIType(conf)
	if err != nil {
		return "", err
	}
	maxTokens := conf.MaxTokens
	if maxTokens < 0 {
		return "", fmt.Errorf("ai_assistant max_tokens must not be negative")
	}
	messages := []map[string]string{
		{"role": "system", "content": assistantPrompt(conf)},
		{"role": "user", "content": prompt},
	}
	bodyMap := map[string]any{"model": strings.TrimSpace(conf.Model)}
	switch apiType {
	case config.AIAPITypeResponses:
		bodyMap["input"] = messages
		if maxTokens > 0 {
			bodyMap["max_output_tokens"] = maxTokens
		}
		if conf.DisableThinking != nil && *conf.DisableThinking {
			bodyMap["reasoning"] = map[string]any{"effort": "none"}
		}
	case config.AIAPITypeAnthropic:
		bodyMap["system"] = assistantPrompt(conf)
		bodyMap["messages"] = []map[string]string{{"role": "user", "content": prompt}}
		if maxTokens > 0 {
			bodyMap["max_tokens"] = maxTokens
		} else {
			bodyMap["max_tokens"] = 1024
		}
	default:
		bodyMap["messages"] = messages
		if maxTokens > 0 {
			bodyMap["max_tokens"] = maxTokens
		}
		if conf.DisableThinking != nil && *conf.DisableThinking {
			bodyMap["thinking"] = map[string]any{"type": "disabled"}
		}
	}
	body, err := json.Marshal(bodyMap)
	if err != nil {
		return "", fmt.Errorf("marshal assistant request: %w", err)
	}
	req, err := http.NewRequest(http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("create assistant request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if key := strings.TrimSpace(conf.APIKey); key != "" {
		if apiType == config.AIAPITypeAnthropic {
			req.Header.Set("x-api-key", key)
		} else {
			req.Header.Set("Authorization", "Bearer "+key)
		}
	}
	if apiType == config.AIAPITypeAnthropic {
		req.Header.Set("anthropic-version", "2023-06-01")
	}
	client := s.client
	if client == nil {
		client = &http.Client{Timeout: s.timeout()}
	}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("request assistant API: %w", err)
	}
	defer resp.Body.Close()
	respBody, err := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if err != nil {
		return "", fmt.Errorf("read assistant response: %w", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("assistant API returned HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(respBody)))
	}
	return extractAssistantText(apiType, respBody)
}

func assistantEndpoint(conf config.AIAssistantConf) (string, error) {
	base := strings.TrimRight(strings.TrimSpace(conf.BaseURL), "/")
	u, err := url.Parse(base)
	if err != nil || u.Scheme == "" || u.Host == "" || (u.Scheme != "http" && u.Scheme != "https") || u.RawQuery != "" || u.Fragment != "" {
		return "", fmt.Errorf("invalid ai_assistant base_url")
	}
	if !strings.HasSuffix(u.Path, "/v1") {
		return "", fmt.Errorf("ai_assistant base_url must end with /v1")
	}
	if strings.TrimSpace(conf.Model) == "" {
		return "", fmt.Errorf("ai_assistant model is required")
	}
	apiType, err := assistantAPIType(conf)
	if err != nil {
		return "", err
	}
	switch apiType {
	case config.AIAPITypeResponses:
		return base + "/responses", nil
	case config.AIAPITypeAnthropic:
		return base + "/messages", nil
	default:
		return base + "/chat/completions", nil
	}
}

func assistantAPIType(conf config.AIAssistantConf) (config.AIAPIType, error) {
	apiType := config.AIAPIType(strings.TrimSpace(string(conf.APIType)))
	if apiType == "" {
		return config.AIAPITypeResponses, nil
	}
	switch apiType {
	case config.AIAPITypeResponses, config.AIAPITypeChatCompletions, config.AIAPITypeAnthropic, config.AIAPITypeDeepSeekJSON:
		return apiType, nil
	default:
		return "", fmt.Errorf("unknown ai_assistant api_type %q", conf.APIType)
	}
}

func extractAssistantText(apiType config.AIAPIType, body []byte) (string, error) {
	if apiType == config.AIAPITypeResponses {
		var response struct {
			OutputText string `json:"output_text"`
			Output     []struct {
				Content []struct {
					Type string `json:"type"`
					Text string `json:"text"`
				} `json:"content"`
			} `json:"output"`
		}
		if err := json.Unmarshal(body, &response); err != nil {
			return "", fmt.Errorf("decode assistant responses: %w", err)
		}
		if strings.TrimSpace(response.OutputText) != "" {
			return response.OutputText, nil
		}
		for _, output := range response.Output {
			for _, content := range output.Content {
				if strings.TrimSpace(content.Text) != "" {
					return content.Text, nil
				}
			}
		}
		return "", fmt.Errorf("assistant response contains no output text")
	}

	if apiType == config.AIAPITypeAnthropic {
		var response struct {
			Content []struct {
				Type string `json:"type"`
				Text string `json:"text"`
			} `json:"content"`
		}
		if err := json.Unmarshal(body, &response); err != nil {
			return "", fmt.Errorf("decode assistant anthropic response: %w", err)
		}
		for _, content := range response.Content {
			if content.Type == "text" && strings.TrimSpace(content.Text) != "" {
				return content.Text, nil
			}
		}
		return "", fmt.Errorf("assistant anthropic response contains no text")
	}

	var response struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
				Refusal string `json:"refusal"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.Unmarshal(body, &response); err != nil {
		return "", fmt.Errorf("decode assistant chat response: %w", err)
	}
	if len(response.Choices) == 0 {
		return "", fmt.Errorf("assistant response contains no choices")
	}
	if strings.TrimSpace(response.Choices[0].Message.Refusal) != "" {
		return "", fmt.Errorf("assistant response refused: %s", response.Choices[0].Message.Refusal)
	}
	return response.Choices[0].Message.Content, nil
}

func assistantPrompt(conf config.AIAssistantConf) string {
	if prompt := strings.TrimSpace(conf.Prompt); prompt != "" {
		return prompt
	}
	return config.DefaultAIAssistantPrompt
}

func (s *AIAssistantService) buildAssistantPrompt(trigger string, page entity.Page, pageURL, pageText string, comments []entity.Comment, current string) string {
	var b strings.Builder
	fmt.Fprintf(&b, "触发标签：%s\n页面标题：%s\n页面地址：%s\n页面正文：\n%s\n\n已有评论：\n", trigger, page.Title, pageURL, pageText)
	for _, c := range comments {
		user := "匿名用户"
		fetched := s.app.Dao().FetchUserForComment(&c)
		if fetched.Name != "" {
			user = fetched.Name
		}
		fmt.Fprintf(&b, "- %s：%s\n", user, limitRunes(c.Content, 1000))
	}
	fmt.Fprintf(&b, "\n当前评论：\n%s\n\n请直接给出回复。", current)
	return b.String()
}

func (s *AIAssistantService) record(comment, reply *entity.Comment, trigger string, status entity.AIAssistantLogStatus, response, errText string) {
	row := entity.AIAssistantLog{CommentID: comment.ID, UserID: comment.UserID, SiteName: comment.SiteName, PageKey: comment.PageKey, Trigger: trigger, Status: string(status), AIModel: s.app.Conf().AIAssistant.Model, Response: response, Error: errText}
	if reply != nil {
		row.ReplyCommentID = reply.ID
	}
	page := s.app.Dao().FindPage(comment.PageKey, comment.SiteName)
	row.PageURL = s.app.Dao().GetPageAccessibleURL(&page)
	if err := s.app.Dao().DB().Create(&row).Error; err != nil {
		log.Errorf("[AIAssistant] record log failed: %v", err)
	}
	s.pruneLogs()
}

func (s *AIAssistantService) pruneLogs() {
	cutoff := time.Now().Add(-aiAssistantLogRetention)
	if err := s.app.Dao().DB().Where("created_at < ? OR status = ?", cutoff, "skipped").Delete(&entity.AIAssistantLog{}).Error; err != nil {
		log.Errorf("[AIAssistant] prune logs failed: %v", err)
	}
}

func fetchPageText(client *http.Client, pageURL string, maxChars int) (string, error) {
	return fetchPageTextWithSelectors(client, pageURL, maxChars, "", nil)
}

func fetchPageTextWithSelectors(client *http.Client, pageURL string, maxChars int, contentSelector string, excludeSelectors []string) (string, error) {
	if pageURL == "" {
		return "", nil
	}
	if maxChars <= 0 {
		maxChars = 12000
	}
	if maxChars > 50000 {
		maxChars = 50000
	}
	if !strings.HasPrefix(pageURL, "http://") && !strings.HasPrefix(pageURL, "https://") {
		return "", nil
	}
	if client == nil {
		client = http.DefaultClient
	}
	resp, err := client.Get(pageURL)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("page returned HTTP %d", resp.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
	if err != nil {
		return "", err
	}
	doc, err := html.Parse(bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	root := doc
	if strings.TrimSpace(contentSelector) != "" {
		matches := findHTMLNodes(doc, contentSelector)
		if len(matches) == 0 {
			return "", nil
		}
		root = matches[0]
	}
	excludes := make([]string, 0, len(excludeSelectors)+3)
	excludes = append(excludes, "script", "style", "noscript")
	for _, selector := range excludeSelectors {
		if strings.TrimSpace(selector) != "" {
			excludes = append(excludes, selector)
		}
	}
	var b strings.Builder
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.ElementNode {
			for _, selector := range excludes {
				if matchesHTMLSelector(n, selector) {
					return
				}
			}
		}
		if n.Type == html.TextNode {
			b.WriteString(n.Data)
			b.WriteByte(' ')
		}
		for child := n.FirstChild; child != nil; child = child.NextSibling {
			walk(child)
		}
	}
	walk(root)
	return limitRunes(strings.Join(strings.Fields(b.String()), " "), maxChars), nil
}

// findHTMLNodes implements the small, predictable selector subset useful for
// article extraction: comma groups, descendant selectors, tag, #id and .class.
func findHTMLNodes(root *html.Node, selector string) []*html.Node {
	var result []*html.Node
	for _, group := range strings.Split(selector, ",") {
		parts := strings.Fields(strings.TrimSpace(group))
		if len(parts) == 0 {
			continue
		}
		var walk func(*html.Node)
		walk = func(n *html.Node) {
			if n.Type == html.ElementNode && matchesSelectorChain(n, parts) {
				result = append(result, n)
			}
			for child := n.FirstChild; child != nil; child = child.NextSibling {
				walk(child)
			}
		}
		walk(root)
	}
	return result
}

func matchesHTMLSelector(n *html.Node, selector string) bool {
	parts := strings.Fields(strings.TrimSpace(selector))
	if len(parts) == 0 {
		return false
	}
	for _, group := range strings.Split(selector, ",") {
		parts = strings.Fields(strings.TrimSpace(group))
		if len(parts) > 0 && matchesSelectorChain(n, parts) {
			return true
		}
	}
	return false
}

func matchesSelectorChain(n *html.Node, parts []string) bool {
	if !matchesSimpleSelector(n, parts[len(parts)-1]) {
		return false
	}
	ancestor := n.Parent
	for i := len(parts) - 2; i >= 0; i-- {
		for ancestor != nil && (ancestor.Type != html.ElementNode || !matchesSimpleSelector(ancestor, parts[i])) {
			ancestor = ancestor.Parent
		}
		if ancestor == nil {
			return false
		}
		ancestor = ancestor.Parent
	}
	return true
}

func matchesSimpleSelector(n *html.Node, selector string) bool {
	selector = strings.TrimSpace(selector)
	if selector == "" || n == nil || n.Type != html.ElementNode {
		return false
	}
	tag := selector
	if i := strings.IndexAny(tag, "#."); i >= 0 {
		tag = tag[:i]
	}
	if tag != "" && tag != "*" && !strings.EqualFold(tag, n.Data) {
		return false
	}
	attrs := make(map[string]string, len(n.Attr))
	for _, attr := range n.Attr {
		attrs[strings.ToLower(attr.Key)] = attr.Val
	}
	if id := selectorPart(selector, '#'); id != "" && attrs["id"] != id {
		return false
	}
	if classes := selectorParts(selector, '.'); len(classes) > 0 {
		present := map[string]bool{}
		for _, class := range strings.Fields(attrs["class"]) {
			present[class] = true
		}
		for _, class := range classes {
			if !present[class] {
				return false
			}
		}
	}
	return true
}

func selectorPart(selector string, prefix byte) string {
	parts := selectorParts(selector, prefix)
	if len(parts) == 0 {
		return ""
	}
	return parts[0]
}

func selectorParts(selector string, prefix byte) []string {
	var result []string
	for _, part := range strings.Split(selector, string(prefix))[1:] {
		part = strings.Trim(part, "#.")
		if i := strings.IndexAny(part, "#."); i >= 0 {
			part = part[:i]
		}
		if part != "" {
			result = append(result, part)
		}
	}
	return result
}

func limitRunes(value string, max int) string {
	value = strings.TrimSpace(value)
	if max <= 0 || utf8.RuneCountInString(value) <= max {
		return value
	}
	runes := []rune(value)
	return strings.TrimSpace(string(runes[:max]))
}

func (s *AIAssistantService) timeout() time.Duration {
	seconds := s.app.Conf().AIAssistant.TimeoutSeconds
	if seconds <= 0 {
		seconds = 30
	}
	if seconds > 120 {
		seconds = 120
	}
	return time.Duration(seconds) * time.Second
}
