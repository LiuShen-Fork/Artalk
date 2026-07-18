# Unified Comment and AI Moderation Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Extend Artalk moderation so every checker reviews nickname plus normalized visible comment text, and add configurable OpenAI-compatible AI moderation through Responses or Chat Completions with strict JSON Schema output.

**Architecture:** Add a focused review-text normalizer and explicit raw/review fields to `CheckerParams`, while preserving the current sequential asynchronous checker pipeline, existing `moderator.pending_default` semantics, and the rule that 管理员 comments skip automatic moderation. Integrate an `AIChecker` as another existing-style checker, and extend generated configuration plus the dynamic sidebar settings UI for AI-specific controls.

**Tech Stack:** Go, Goldmark, Bluemonday, `golang.org/x/net/html`, `net/http`, YAML config generation, Vue 3, TypeScript, Vitest/testify-style Go tests.

---

## File structure

- Create `internal/anti_spam/review_text.go`: Markdown/HTML-to-visible-text normalization and review text assembly.
- Create `internal/anti_spam/review_text_test.go`: normalization and nickname/body assembly tests.
- Create `internal/anti_spam/ai.go`: OpenAI-compatible Responses and Chat Completions checker.
- Create `internal/anti_spam/ai_test.go`: request/response/config/error tests using `httptest.Server`.
- Modify `internal/anti_spam/base.go`: checker parameters, AI checker registration, safe logging.
- Modify `internal/anti_spam/keywords.go`: field-aware nickname/body matching and safe raw replacement.
- Modify `internal/anti_spam/keywords_test.go`: nickname, normalized body, replacement, split markup, entity and ignored URL/emoticon coverage.
- Modify `internal/anti_spam/base_test.go`: checker ordering and `api_fail_block` integration.
- Modify `internal/anti_spam/akismet.go`, `tencent.go`, `aliyun.go`: send unified `ReviewText` while retaining provider-specific author metadata.
- Modify `internal/core/service_anti_spam.go`: prepare `RawContent`, `ReviewContent`, and `ReviewText` once per comment.
- Modify `internal/config/config.go`: `AI` moderation configuration types.
- Modify `conf/artalk.example.yml` and `conf/artalk.example.zh-CN.yml`: defaults, descriptions, select metadata and default prompt.
- Regenerate `internal/config/cache.go`, `docs/docs/en/guide/env.md`, and `docs/docs/zh/guide/env.md`.
- Modify `ui/artalk-sidebar/src/lib/settings-option.ts`: path-based textarea and placeholder presentation metadata.
- Modify `ui/artalk-sidebar/src/components/PreferenceItem.vue`: render textarea and placeholder.
- Modify `ui/artalk-sidebar/src/pages/settings.vue`: textarea styling.
- Modify `ui/artalk-sidebar/src/lib/settings-sensitive.ts`: hide `moderator.ai.api_key`.

### Task 1: Normalize review text

**Files:**
- Create: `internal/anti_spam/review_text.go`
- Create: `internal/anti_spam/review_text_test.go`
- Modify: `go.mod`

- [ ] **Step 1: Write failing normalization tests**

Add table tests that call:

```go
reviewContent, err := NormalizeReviewContent(input)
require.NoError(t, err)
assert.Equal(t, expected, reviewContent)
```

Required cases include:

```go
{"plain", "普通评论", "普通评论"},
{"markdown link", "[点击领取](https://spam.example/ad)", "点击领取"},
{"html link", `<a href="https://spam.example/ad">点击领取</a>`, "点击领取"},
{"markdown image", `![](https://spam.example/ad.jpg)`, "[图片]"},
{"html image", `<img src="https://spam.example/ad.jpg" alt="广告">`, "[图片]"},
{"emoticon", `<img src="https://owo.example/a.png" atk-emoticon="blobcat">`, ""},
{"split tag", `广<strong>告</strong>`, "广告"},
{"entity", `敏感&amp;内容`, "敏感&内容"},
{"invisible", `<script>alert(1)</script><style>.x{}</style><!--x-->正文`, "正文"},
{"whitespace", "第一行\n\n   第二行", "第一行 第二行"},
```

Also assert:

```go
assert.Equal(t, "昵称: 张三\n评论: 正文", BuildReviewText("张三", "正文"))
```

- [ ] **Step 2: Run the focused test and verify failure**

Run:

```bash
go test ./internal/anti_spam -run 'TestNormalizeReviewContent|TestBuildReviewText' -count=1
```

Expected: build failure because normalization helpers do not exist.

- [ ] **Step 3: Implement Markdown/HTML normalization**

Implement `NormalizeReviewContent(content string) (string, error)` by calling the existing `utils.Marked`, parsing the returned fragment with `html.ParseFragment`, traversing text nodes, skipping `script`, `style`, and Artalk emoticon `img` nodes, replacing other `img` nodes with `[图片]`, and joining `strings.Fields` with one space. Implement:

```go
func BuildReviewText(userName, reviewContent string) string {
    return fmt.Sprintf("昵称: %s\n评论: %s", userName, reviewContent)
}
```

Promote `golang.org/x/net` from indirect to direct dependency through `go mod tidy` after importing `golang.org/x/net/html`.

- [ ] **Step 4: Run normalization tests**

Run the focused test command again. Expected: PASS.

- [ ] **Step 5: Commit normalization unit**

```bash
git add internal/anti_spam/review_text.go internal/anti_spam/review_text_test.go go.mod go.sum
git commit -m "feat(moderator): normalize comment review text"
```

### Task 2: Make existing moderation field-aware

**Files:**
- Modify: `internal/anti_spam/base.go`
- Modify: `internal/core/service_anti_spam.go`
- Modify: `internal/anti_spam/akismet.go`
- Modify: `internal/anti_spam/tencent.go`
- Modify: `internal/anti_spam/aliyun.go`
- Modify: `internal/anti_spam/keywords.go`
- Modify: `internal/anti_spam/keywords_test.go`
- Modify: `internal/anti_spam/base_test.go`

- [ ] **Step 1: Write failing keyword behavior tests**

Add tests proving:

```go
// nickname always blocks, even in replace mode
params := &CheckerParams{UserName: "广告用户", RawContent: "正常正文", ReviewContent: "正常正文"}
pass, err := checker.Check(params)
assert.NoError(t, err)
assert.False(t, pass)
assert.False(t, updated)

// visible body replaces raw body when the raw substring is contiguous
params = &CheckerParams{RawContent: "<b>广告</b>", ReviewContent: "广告"}
assert.True(t, pass)
assert.Equal(t, "<b>**</b>", updatedContent)

// split markup is visible to review but unsafe to replace
params = &CheckerParams{RawContent: "广<strong>告</strong>", ReviewContent: "广告"}
assert.False(t, pass)
assert.False(t, updated)

// URL and emoticon fields excluded by normalization do not match
```

- [ ] **Step 2: Run keyword tests and verify failure**

```bash
go test ./internal/anti_spam -run 'TestNewKeywordsChecker|TestAntiSpam' -count=1
```

Expected: FAIL because the checker still reads `Content` only and does not inspect nickname.

- [ ] **Step 3: Extend checker parameters and prepare them once**

Replace the ambiguous content field with:

```go
RawContent    string
ReviewContent string
ReviewText    string
```

In `payload2CheckerParams`, call `NormalizeReviewContent(payload.Comment.Content)` and `BuildReviewText(user.Name, reviewContent)`. If normalization unexpectedly fails, log the error and use whitespace-normalized raw content as the review fallback so a single formatting failure does not skip all moderation.

Update anti-spam debug logs to quote `RawContent`, never API secrets.

- [ ] **Step 4: Switch existing external checkers to unified review input**

Use `p.ReviewText` for Akismet `comment_content`, Tencent `Content`, and Aliyun `Content`. Continue passing Akismet `CommentAuthor` and Tencent `UserName` through their existing dedicated author fields.

- [ ] **Step 5: Implement nickname-first keyword behavior**

Load keywords once as today. For each keyword:

```go
if strings.Contains(p.UserName, keyword) {
    return false, nil
}
```

Collect all keywords present in `ReviewContent`. Block immediately in block mode. In replace mode, first verify every matched keyword exists contiguously in `RawContent`; if any does not, return false without updating. Otherwise replace all occurrences in a temporary copy and invoke `OnUpdateComment` exactly once.

- [ ] **Step 6: Run field-aware moderation tests**

Run:

```bash
go test ./internal/anti_spam -run 'TestNormalizeReviewContent|TestBuildReviewText|TestNewKeywordsChecker|TestAntiSpam' -count=1
```

Expected: PASS.

- [ ] **Step 7: Commit field-aware moderation**

```bash
git add internal/anti_spam internal/core/service_anti_spam.go
git commit -m "feat(moderator): review nickname and visible comment text"
```

### Task 3: Add AI checker with strict structured output

**Files:**
- Create: `internal/anti_spam/ai.go`
- Create: `internal/anti_spam/ai_test.go`
- Modify: `internal/config/config.go`
- Modify: `internal/anti_spam/base.go`
- Modify: `internal/anti_spam/base_test.go`

- [ ] **Step 1: Write failing Responses and Chat tests**

Use `httptest.Server` handlers to assert:

```go
assert.Equal(t, "/v1/responses", r.URL.Path)
assert.Equal(t, "Bearer test-key", r.Header.Get("Authorization"))
assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
```

Decode request bodies and assert Responses uses `instructions`, `input`, and `text.format.type == "json_schema"`; Chat uses system/user messages and `response_format.type == "json_schema"`.

Return representative provider responses containing:

```json
{"sensitive":true,"reason":"需要人工复核"}
```

Assert `Check` returns `false`. Add corresponding `sensitive:false` tests that return `true`.

- [ ] **Step 2: Write failing validation/error tests**

Cover invalid base URL, base URL not ending `/v1`, empty model, HTTP 500, invalid JSON, absent output, missing required fields, wrong field types and an empty API key that omits Authorization.

Run:

```bash
go test ./internal/anti_spam -run 'TestAIChecker' -count=1
```

Expected: build failure because `AIChecker` does not exist.

- [ ] **Step 3: Add AI configuration types**

In `ModeratorConf`, add:

```go
AI AIAntispamConf `koanf:"ai" json:"ai"`
```

Define:

```go
type AIAPIType string

const (
    AIAPITypeResponses       AIAPIType = "responses"
    AIAPITypeChatCompletions AIAPIType = "chat_completions"
)

type AIAntispamConf struct {
    Enabled bool      `koanf:"enabled" json:"enabled"`
    APIType AIAPIType `koanf:"api_type" json:"api_type"`
    BaseURL string    `koanf:"base_url" json:"base_url"`
    APIKey  string    `koanf:"api_key" json:"api_key"`
    Model   string    `koanf:"model" json:"model"`
    Prompt  string    `koanf:"prompt" json:"prompt"`
}
```

- [ ] **Step 4: Implement shared schema and result parsing**

Define an internal result:

```go
type aiModerationResult struct {
    Sensitive bool   `json:"sensitive"`
    Reason    string `json:"reason"`
}
```

Use a strict object schema with both required fields and `additionalProperties:false`. Decode through a temporary representation or schema-aware validation so missing `sensitive` is not silently accepted as Go's zero value. Reject unknown fields and missing/incorrect fields.

- [ ] **Step 5: Implement Responses adapter**

POST to `{baseURL}/responses` with:

```json
{
  "model": "configured-model",
  "instructions": "configured prompt",
  "input": "昵称: ...\n评论: ...",
  "text": {
    "format": {
      "type": "json_schema",
      "name": "comment_moderation",
      "strict": true,
      "schema": {}
    }
  }
}
```

Extract the first `output[].content[]` item whose type is `output_text`, then parse the JSON result.

- [ ] **Step 6: Implement Chat Completions adapter**

POST to `{baseURL}/chat/completions` with system/user messages and:

```json
"response_format": {
  "type": "json_schema",
  "json_schema": {
    "name": "comment_moderation",
    "strict": true,
    "schema": {}
  }
}
```

Extract `choices[0].message.content` and parse the result.

- [ ] **Step 7: Register AI checker in pipeline**

When `moderator.ai.enabled` is true, append `NewAIChecker(...)` after Aliyun and before Keywords. Use a 30-second HTTP timeout. Let checker errors flow through the existing `api_fail_block` handling.

- [ ] **Step 8: Run AI and anti-spam tests**

```bash
go test ./internal/anti_spam -count=1
```

Expected: PASS.

- [ ] **Step 9: Commit AI backend**

```bash
git add internal/anti_spam internal/config/config.go
git commit -m "feat(moderator): add OpenAI-compatible AI checker"
```

### Task 4: Add generated configuration and defaults

**Files:**
- Modify: `conf/artalk.example.yml`
- Modify: `conf/artalk.example.zh-CN.yml`
- Regenerate: `internal/config/cache.go`
- Regenerate: `docs/docs/en/guide/env.md`
- Regenerate: `docs/docs/zh/guide/env.md`

- [ ] **Step 1: Add English and Chinese AI configuration blocks**

Add `moderator.ai` between Aliyun and Keywords. Use selector comments:

```yaml
# AI API type ["responses", "chat_completions"]
api_type: responses
```

Set `enabled:false`, `base_url:https://api.openai.com/v1`, empty API key/model, and a complete multiline default prompt that maps advertising, illegal, sexual, violent, hateful, privacy-leaking, politically sensitive and clearly review-worthy comments to `sensitive:true`, protects against prompt injection, and requires the schema.

- [ ] **Step 2: Regenerate cache and environment docs**

Run:

```bash
make update-conf
make update-conf-docs
```

Expected: `internal/config/cache.go` and both language environment docs include `moderator.ai.*` and `ATK_MODERATOR_AI_*` entries.

- [ ] **Step 3: Verify generated configuration**

Run:

```bash
go test ./internal/config/... -count=1
rg -n "MODERATOR_AI|moderator.ai" internal/config/cache.go docs/docs/en/guide/env.md docs/docs/zh/guide/env.md
```

Expected: tests pass and all AI paths are present.

- [ ] **Step 4: Commit configuration**

```bash
git add conf internal/config/cache.go docs/docs/en/guide/env.md docs/docs/zh/guide/env.md
git commit -m "feat(config): expose AI moderation settings"
```

### Task 5: Extend sidebar settings controls

**Files:**
- Modify: `ui/artalk-sidebar/src/lib/settings-option.ts`
- Modify: `ui/artalk-sidebar/src/components/PreferenceItem.vue`
- Modify: `ui/artalk-sidebar/src/pages/settings.vue`
- Modify: `ui/artalk-sidebar/src/lib/settings-sensitive.ts`

- [ ] **Step 1: Add option presentation metadata**

Extend `OptionNode` with:

```ts
control?: 'input' | 'textarea'
placeholder?: string
```

Add a centralized path map:

```ts
const optionPresentation = {
  'moderator.ai.base_url': { placeholder: 'https://xxx.xxx.com/v1' },
  'moderator.ai.prompt': { control: 'textarea' },
} as const
```

Merge matching metadata into each constructed node.

- [ ] **Step 2: Render textarea and placeholders**

In `PreferenceItem.vue`, render `<textarea>` when `node.control === 'textarea'`; otherwise retain the existing sensitive input. Bind `:placeholder="node.placeholder"` to both. Preserve env-controlled disabled behavior and save on change.

- [ ] **Step 3: Mark API key sensitive and style textarea**

Add `moderator.ai.api_key` to `SensitiveConfigPaths`. Extend settings page styles so textarea uses the same width, colors and border as inputs, has a practical minimum height, vertical resizing, and visible focus state.

- [ ] **Step 4: Build the sidebar**

Run:

```bash
pnpm -F @artalk/artalk-sidebar build
```

Expected: `vue-tsc` and Vite build succeed.

- [ ] **Step 5: Commit sidebar settings**

```bash
git add ui/artalk-sidebar/src/lib/settings-option.ts ui/artalk-sidebar/src/components/PreferenceItem.vue ui/artalk-sidebar/src/pages/settings.vue ui/artalk-sidebar/src/lib/settings-sensitive.ts
git commit -m "feat(sidebar): configure AI moderation"
```

### Task 6: Full verification and curl examples

**Files:**
- Modify only if verification reveals defects.

- [ ] **Step 1: Format Go files**

```bash
gofmt -w internal/anti_spam internal/core/service_anti_spam.go internal/config/config.go
```

- [ ] **Step 2: Run focused and full Go tests**

```bash
go test ./internal/anti_spam ./internal/core ./internal/config/... -count=1
go test ./... -count=1
```

Expected: PASS.

- [ ] **Step 3: Run frontend validation**

```bash
pnpm -F @artalk/artalk-sidebar build
```

Expected: PASS.

- [ ] **Step 4: Verify repository diff and generated files**

```bash
git diff --check
git status --short
```

Expected: no whitespace errors; only intended implementation files are modified.

- [ ] **Step 5: Prepare manual API examples**

Provide complete curl requests for both `/v1/responses` and `/v1/chat/completions`, including Authorization, model, default prompt, nickname/comment input and the exact strict JSON Schema. Keep Base URL in the documented form `https://xxx.xxx.com/v1` and append only the endpoint path.

- [ ] **Step 6: Commit any final verification fixes**

```bash
git add <only-files-fixed-during-verification>
git commit -m "test(moderator): verify AI moderation integration"
```

Skip this commit when verification requires no further changes.



