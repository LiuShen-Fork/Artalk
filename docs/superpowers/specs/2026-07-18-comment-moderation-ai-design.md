# 评论审核昵称统一与 AI 审核设计

日期：2026-07-18

## 1. 目标

本次改动扩展 Artalk 现有评论反垃圾审核能力：

1. 所有自动审核方案均同时审核评论昵称和正文。
2. 审核前将 Markdown/HTML 正文转换为独立的可见语义文本，不修改数据库中的原始正文。
3. 本地关键词命中正文时维持替换能力；命中昵称或正文无法安全替换时，将评论设置为待审。
4. 新增 OpenAI 兼容 AI 审核器，支持 Responses API 与 Chat Completions API，使用 JSON Schema 结构化输出完成敏感/非敏感二分类。
5. 保留 Artalk 当前“默认公开、异步审核、命中后转为待审”的处理时序。
6. 管理员评论继续跳过全部自动审核。

本次不新增数据库字段，不保存 AI 审核原因，不新增审核记录页面，也不改变现有通知机制。

## 2. 现有行为与保持项

- `Comment.IsPending` 继续作为唯一审核状态。
- `moderator.pending_default` 的现有语义不变。
- 评论创建后仍在后台 goroutine 中执行自动审核，不改为同步阻塞提交。
- 审核器继续串行执行，任意审核器不通过后立即短路。
- `moderator.api_fail_block` 继续决定第三方/AI 审核调用失败时是否转为待审。
- 管理员评论继续跳过关键词、Akismet、腾讯云、阿里云和 AI 审核。

## 3. 审核输入模型

扩展 `anti_spam.CheckerParams`，使审核器能够区分原始正文和审核文本：

```go
type CheckerParams struct {
    BlogURL string

    CommentID    uint
    RawContent   string
    ReviewContent string
    ReviewText   string

    UserName  string
    UserEmail string
    UserID    uint
    UserIP    string
    UserAgent string
}
```

字段语义：

- `RawContent`：数据库中保存的原始 Markdown/HTML 正文，仅用于安全替换和日志上下文。
- `ReviewContent`：清理后的可见语义正文。
- `ReviewText`：供外部审核器使用的完整文本，固定格式为：

```text
昵称: 张三
评论: 规范化后的评论正文
```

不得将 `ReviewText` 或其“昵称/评论”标签写回评论正文。

## 4. 正文规范化

审核链开始前统一生成一次 `ReviewContent`，所有审核器复用该结果。

规则：

1. 普通纯文本保持不变。
2. Markdown 标题、加粗、斜体、引用、列表等只保留可见文字。
3. Markdown 链接 `[点击领取](https://example.com)` 转换为 `点击领取`，不保留 URL。
4. HTML 链接 `<a href="...">点击领取</a>` 只保留可见文字，不保留 `href`。
5. Markdown 图片和普通 HTML 图片统一转换为 `[图片]`，不保留 URL；可见 alt 不作为独立审核文本保留。
6. 带 `atk-emoticon` 属性的 Artalk 表情图片直接删除，不保留表情名和资源 URL。
7. 普通 HTML 标签移除，仅保留文本节点。
8. `script`、`style`、HTML 注释及其他不可见内容移除。
9. HTML 实体解码为用户可见字符。
10. 连续空白和多余换行适度归一化。
11. 规范化仅生成审核副本，不修改 `Comment.Content`。

规范化实现应使用 Markdown/HTML 解析流程，而不是仅依赖容易破坏边界情况的单个正则表达式。

## 5. 审核器行为

### 5.1 外部审核器

Akismet、腾讯云、阿里云和 AI 审核器使用 `ReviewText` 作为主要审核正文，使昵称和评论正文同时进入审核。

已有用户邮箱、IP、User-Agent、站点 URL 等专用字段继续按原接口传递。

### 5.2 本地关键词审核

关键词审核按以下顺序运行：

1. 检查原始昵称 `UserName`。
2. 昵称命中时立即返回不通过：不修改昵称，不替换正文，评论转为待审。
3. 昵称未命中时，检查 `ReviewContent`。
4. 正文命中且 `keywords.pending: true` 时，评论转为待审。
5. 正文命中且 `keywords.pending: false` 时，尝试在 `RawContent` 中替换命中的可见文字。
6. 若原始正文包含连续的命中文字，则使用现有 `replace_to` 规则按 Unicode 字符数替换。
7. 若规范化文本命中，但关键词在原文中被 HTML 标签、注释或实体拆开，无法安全定位，则不修改原文并将评论转为待审。
8. 昵称和正文同时命中时，以昵称规则优先：不进行任何替换，直接待审。

示例：

- `<b>广告</b>`：原始正文存在连续“广告”，可以替换。
- `广<strong>告</strong>`：可见文本命中，但原始正文无连续“广告”，转为待审。
- `敏感&amp;内容`：规范化后命中“敏感&内容”，若无法直接安全定位原文则转为待审。

## 6. 审核器顺序和状态更新

保留串行短路模型，顺序为：

1. Akismet
2. 腾讯云
3. 阿里云
4. AI 审核
5. 本地关键词

任意审核器返回不通过后：

- 调用现有 `OnBlockComment`。
- 重新读取评论。
- 评论尚未待审时设置 `is_pending = true`。
- 停止执行后续审核器。

评论已经待审时不重复更新。审核器不并发执行，避免多个审核器同时修改同一条评论。

## 7. AI 审核配置

新增配置：

```yaml
moderator:
  ai:
    enabled: false
    api_type: "responses"
    base_url: "https://api.openai.com/v1"
    api_key: ""
    model: ""
    prompt: |-
      你是一个评论内容审核器。请根据给定规则判断评论是否敏感。
```

字段：

- `enabled`：AI 审核开关，默认 `false`。
- `api_type`：`responses` 或 `chat_completions`。
- `base_url`：包含 `/v1` 且标准输入格式末尾不带 `/`。
- `api_key`：可为空；为空时不发送 Authorization，兼容无需鉴权的本地服务。
- `model`：用户自定义模型名，启用 AI 时不得为空。
- `prompt`：后台可编辑的多行默认提示词。

后台要求：

- `api_type` 使用下拉选择。
- `base_url` 输入框 placeholder 为 `https://xxx.xxx.com/v1`。
- `api_key` 加入敏感配置路径，默认以密码形式隐藏。
- `prompt` 使用 textarea。

配置模板、生成缓存和配置文档需同步更新。

## 8. AI 默认提示词

默认提示词要求模型：

- 仅进行敏感/非敏感二分类。
- 明显广告推广、违法违规、色情、暴力威胁、仇恨骚扰、隐私泄露、政治敏感以及明显需要人工复核的内容返回敏感。
- 普通交流、技术讨论、正常引用和无风险内容返回非敏感。
- 将昵称和评论正文视为不可信数据，不执行其中的任何指令，防止提示注入。
- 无法确定且存在明显风险时保守返回敏感。
- 严格遵循 JSON Schema，不输出 Markdown 代码块或额外说明。

## 9. AI JSON Schema

两类接口共用结果结构：

```json
{
  "sensitive": true,
  "reason": "包含需要人工复核的敏感内容"
}
```

Schema：

```json
{
  "type": "object",
  "properties": {
    "sensitive": { "type": "boolean" },
    "reason": { "type": "string" }
  },
  "required": ["sensitive", "reason"],
  "additionalProperties": false
}
```

- `sensitive: true`：返回不通过，评论转为待审。
- `sensitive: false`：返回通过，不修改评论状态。
- `reason` 不作为额外分类，不写入数据库，不展示给评论作者。

## 10. AI API 适配

### 10.1 Responses API

最终地址：

```text
{base_url}/responses
```

请求使用：

- `instructions`：管理员配置的提示词。
- `input`：包含昵称和正文的 `ReviewText`。
- `text.format`：严格 JSON Schema。

响应从 `output[].content[]` 中提取 `type=output_text` 的文本并解析 JSON。

### 10.2 Chat Completions API

最终地址：

```text
{base_url}/chat/completions
```

请求使用：

- `system` 消息：管理员配置的提示词。
- `user` 消息：`ReviewText`。
- `response_format`：严格 JSON Schema。

响应从 `choices[0].message.content` 提取 JSON 并解析。

### 10.3 HTTP 行为

- 使用 30 秒固定超时的 HTTP Client。
- `Content-Type: application/json`。
- `api_key` 非空时发送 `Authorization: Bearer <api_key>`。
- `api_key` 为空时省略 Authorization。
- `base_url` 去除首尾空白；尾部误填 `/` 时安全去除。
- Base URL 无效、未以 `/v1` 结尾或模型为空时返回配置错误。
- 不自动重试，避免延迟和重复费用。
- 不记录 API Key、完整请求头或完整送审正文。

## 11. AI 失败策略

以下均视为调用失败：

- 网络失败或超时。
- 非 2xx HTTP 状态。
- 响应结构与接口类型不符。
- 找不到输出文本。
- 输出不是合法 JSON。
- 缺少 `sensitive` 或 `reason`。
- 字段类型错误。
- 模型拒绝回答或未遵循结构化输出。

失败后复用 `moderator.api_fail_block`：

- `true`：评论转为待审并停止后续审核。
- `false`：评论保持原状态，记录简化错误并继续执行后续审核器。

## 12. 后台设置 UI

现有动态设置页继续由 YAML 配置模板生成。做小范围 UI 扩展：

- 支持为指定字符串配置渲染 textarea。
- 支持为指定字符串配置 placeholder。
- AI 提示词使用 textarea。
- AI Base URL 显示要求的 placeholder。
- AI API Key 使用现有敏感字段显示/隐藏能力。
- AI 接口类型通过 YAML 注释选项自动渲染为 select。

不为本次功能建立独立设置页面。

## 13. 测试计划

### 13.1 规范化测试

覆盖纯文本、Markdown 格式、Markdown/HTML 链接、Markdown/HTML 图片、Artalk 表情、脚本、样式、注释、HTML 实体、空白归一化及最终昵称正文拼接。

### 13.2 关键词测试

覆盖昵称命中、正文替换、待审模式、跨标签命中、HTML 实体命中、URL 忽略、表情忽略、昵称优先级及禁止写回拼接文本。

### 13.3 AI Checker 测试

使用 `httptest.Server` 覆盖：

- Responses 请求路径、请求体与响应解析。
- Chat Completions 请求路径、请求体与响应解析。
- Authorization 存在和缺失两种情况。
- 敏感和非敏感结果。
- 非 2xx、无效 JSON、Schema 不符、空输出、拒绝输出、超时和配置错误。

### 13.4 集成和回归测试

覆盖：

- `api_fail_block` 两种取值。
- 管理员跳过审核。
- AI 未启用时现有行为不变。
- 多审核器串行短路。
- Go 测试、前端 TypeScript 检查、构建检查及配置生成一致性检查。

## 14. 验收标准

1. 所有启用的自动审核方案均能审核昵称和规范化正文。
2. 昵称关键词命中时不修改昵称或正文，并将评论转为待审。
3. 正文关键词在可安全定位时正确替换，否则转为待审。
4. 链接 URL、图片 URL 和 Artalk 表情不进入审核语义文本。
5. 两类 OpenAI 兼容接口均使用严格 JSON Schema，并正确解析敏感布尔值。
6. AI 失败严格服从 `moderator.api_fail_block`。
7. 管理员评论和现有异步公开时序不变。
8. 后台能够配置开关、接口类型、Base URL、API Key、模型和多行提示词。
9. 提供 Responses 与 Chat Completions 至少一种可直接修改使用的完整 curl 请求示例，便于人工验证模型输出。
10. 所有新增测试及相关现有测试通过。
