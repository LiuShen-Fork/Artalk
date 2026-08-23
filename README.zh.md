# Artalk Fork

这是基于 [ArtalkJS/Artalk](https://github.com/ArtalkJS/Artalk) 的个人维护版本。Artalk 是一个可自托管的评论系统，后端使用 Go，前端使用原生 JavaScript，支持 Docker、SQLite/MySQL/PostgreSQL 等部署方式。

本仓库主要面向后端功能和自用部署场景。前端评论组件、公开 API 以及上游的基础功能保持兼容；Fork 自定义内容集中在评论审核、管理端、登录、图片上传、IP 属地和发布流程。

## 与上游的关系

- 上游仓库：[ArtalkJS/Artalk](https://github.com/ArtalkJS/Artalk)
- 本仓库：[LiuShen-Fork/Artalk](https://github.com/LiuShen-Fork/Artalk)
- 上游文档：[artalk.js.org](https://artalk.js.org)
- 上游变更会持续通过 `upstream` 远程同步，Fork 的定制提交会保留在本仓库自己的分支上。
- 本 Fork 不发布 npm 前端包；前端评论包仍以当前上游版本为基础，主要维护服务端和管理端。

## 相对上游的改动

### AI 评论审核

在原有审核器基础上增加了可选的 AI 审核。AI 审核默认异步执行，评论提交后先按原有流程显示，检测结果返回后再根据结果设置待审或更新内容。

- 昵称和评论正文会一起发送给审核器，审核输入形如 `昵称: 评论正文`。
- 只将评论正文中的命中内容替换为 `***`；昵称命中时不修改昵称，直接将评论设为待审。
- 审核前会生成适合模型理解的纯文本：Markdown 超链接保留链接文本，图片替换为 `[图片]`，表情包图片和 HTML 标签不会作为语义内容发送。
- 支持三种接口模式：
  - `responses`：OpenAI Responses API 的 JSON Schema。
  - `chat_completions`：OpenAI Chat Completions API 的 JSON Schema。
  - `deepseek_json_output`：DeepSeek Chat Completions API 的 JSON Output。
- JSON Schema 和输出结构由 Artalk 内置，固定返回 `sensitive` 与非空的 `reason`，用户只需要填写提示词、模型、API 地址和密钥。
- AI 审核提示词可编辑，默认规则覆盖广告推广、垃圾信息、违法、色情、暴力威胁、仇恨骚扰、隐私泄露和政治敏感内容。
- 思考模式可以配置，默认关闭，以减少审核请求的 token 消耗。
- AI 请求失败时可按 `moderator.api_fail_block` 决定放行或转为待审。

示例配置：

~~~yaml
moderator:
  ai:
    enabled: true
    api_type: responses
    base_url: https://api.openai.com/v1
    api_key: "sk-..."
    model: gpt-5-mini
    max_tokens: 256
    disable_thinking: true
    prompt: >-
      你是评论内容审核分类器。仅判断昵称和评论正文是否敏感，普通交流和技术讨论返回正常。
~~~

`base_url` 必须填写到 `/v1`，不要在末尾增加斜杠。详细字段可参考 [中文配置示例](./conf/artalk.example.zh-CN.yml) 和 [环境变量文档](./docs/docs/zh/guide/env.md)。

### AI 评论助手

新增可选的 AI 评论助手。评论中提及配置的助手名称时，助手会结合页面内容、近期评论和当前评论生成回复，并避免在嵌套评论中重复回复。

- 支持 `responses`、`chat_completions`、`deepseek_json_output` 和 `anthropic_messages` 四种接口模式，最后一种对应 Claude Messages API。
- `reply_to_pending` 控制待审核评论是否可以触发 AI 回复。
- AI 助手提示词可在管理端设置页编辑，并使用多行输入框。

示例配置和环境变量说明可参考 [中文配置示例](./conf/artalk.example.zh-CN.yml) 和 [环境变量文档](./docs/docs/zh/guide/env.md)。

### 审核记录和管理端

- 新增审核记录页面，用于查看异常审核、审核失败和内容替换记录。
- 正常通过的审核结果不再写入列表，避免每条评论产生多条无用记录。
- 审核记录默认保留 90 天，也支持管理员单条删除和批量清空。
- 审核记录中提供“通过评论”和“删除评论”的快捷操作，便于处理误判和广告评论。
- 管理员总览页、评论页、审核页、站点页和设置页统一了页面头部、卡片、按钮、间距和响应式样式。
- 设置页支持展开并编辑配置项，默认配置和管理端界面使用中文。
- 管理员进入控制中心时默认显示最近评论，Dashboard 仍可从管理端导航进入。

### 通用 OAuth 2.0 登录

新增可配置的通用 OAuth 2.0 授权码登录，可用于 GitHub 之外的 OAuth 2.0 服务。需要配置：

- 客户端 ID 和客户端密钥
- 授权端点地址
- 令牌端点地址
- 用户信息端点地址
- 授权范围 `scopes`
- 登录方式显示名称 `label`

OAuth 应用的回调地址为：

~~~text
https://你的 Artalk 地址/api/v2/auth/generic/callback
~~~

登录弹窗会显示 `label` 的值，默认是 `OAuth 2.0`，可以改成“公司账号”“自建登录”等名称。用户信息端点需要返回 JSON，并提供可用于创建或匹配 Artalk 用户的身份和邮箱信息。

另外保留了面向已有 OIDC 登录态的 SSO 令牌交换功能。SSO 的 `issuer` 应填写 OIDC 服务的 issuer 或基础地址，Artalk 会调用该服务的 `/userinfo` 校验访问令牌；它适合外部页面已经完成 OIDC 登录、无需再次显示 Artalk 登录弹窗的场景。

### 图片上传

- 新增 Lsky Pro 兰空图床 API 上传，Artalk 会直接以 multipart 请求将图片发送到兰空，不依赖 Upgit 的本地路径上传方式。
- 支持配置兰空的 API 地址、Token、权限、相册 ID、策略 ID 和上传成功后删除本地文件。
- Upgit 上传方式继续保留，适合必须通过本地文件路径处理的场景。

### IPv6 IP 属地

IP 属地支持单独配置 IPv6 的 ip2region `.xdb` 数据库：

~~~yaml
ip_region:
  db_path: ./data/ip2region.xdb
  db_path_v6: ./data/ip2region_v6.xdb
~~~

IPv4 使用 `db_path`，IPv6 在配置了 `db_path_v6` 且文件存在时使用 IPv6 数据库；未配置时继续使用原有逻辑。

### CI/CD 和 Docker 发布

Fork 清理了不适合本仓库的 npm 包、文档、E2E 和部分强绑定上游的 Action，并将发布流程改为手动触发。

在 GitHub Actions 中手动执行发布工作流时填写版本号，例如 `v1.0.0`，流程会：

1. 创建或更新对应版本标签。
2. 构建各平台二进制文件。
3. 创建 GitHub Release 并上传二进制文件。
4. 构建并推送 Docker 镜像 `willowgod/artalk`。

仓库需要配置 Docker Hub 密钥：

- `DOCKERHUB_USERNAME`
- `DOCKERHUB_TOKEN`

`dry_run` 用于只执行构建和检查、不创建 Release 或推送镜像。正常发布时保持关闭。

## 本地调试

### 环境要求

- Go 版本以 [go.mod](./go.mod) 为准。
- Node.js 和 pnpm 版本以根目录 [package.json](./package.json) 为准。

启动后端服务：

~~~powershell
go run . -c .\artalk.yml server --host 127.0.0.1 --port 23366
~~~

启动管理端开发服务器：

~~~powershell
pnpm dev:sidebar
~~~

开发调试时不需要先构建整个项目。管理端开发服务器会提供热更新；只有发布或验证产物时才需要执行构建。

### 创建管理员

首次启动前可以执行：

~~~powershell
go run . -c .\artalk.yml admin --name admin --email admin@example.com --password "替换成强密码"
~~~

仓库不提供固定的默认管理员密码。管理员账号由上述命令创建，密码应使用自己的强密码。

默认本地数据位置为：

~~~text
.\data\artalk.db
.\data\artalk.log
~~~

`data/`、`artalk.yml` 和日志文件已加入 `.gitignore`，不会进入 Git 提交。

## Docker 使用

~~~bash
docker run -d \
  --name artalk \
  -p 8080:23366 \
  -v "$(pwd)/data:/data" \
  -e TZ=Asia/Shanghai \
  -e ATK_LOCALE=zh-CN \
  -e ATK_SITE_DEFAULT="我的站点" \
  -e ATK_SITE_URL="https://example.com" \
  willowgod/artalk
~~~

生产环境建议挂载独立的数据目录，并通过环境变量或配置文件注入数据库、管理员、邮件、OAuth 和 AI 密钥，不要把密钥提交到仓库。

## 使用原版功能

本 Fork 继续包含上游的基础能力，包括：

- Markdown、表情包、图片上传和评论回复
- 多站点、页面管理、评论搜索、置顶和投票
- 邮件通知、Webhook 和多种管理员通知方式
- 验证码、关键词审核、Akismet、腾讯云和阿里云审核
- SQLite、MySQL、PostgreSQL、SQL Server
- Docker、二进制和源码部署
- 多语言、暗色模式、懒加载、灯箱和 IP 属地

完整的原版配置和 API 说明请以上游 [文档](https://artalk.js.org) 为准。

## 同步上游

~~~powershell
git remote add upstream https://github.com/ArtalkJS/Artalk.git
git fetch upstream
git rebase upstream/master
~~~

如果上游改动与 Fork 的定制功能发生冲突，应优先检查审核器、配置结构、管理端页面和发布工作流。完成 rebase 后重新执行后端测试和管理端构建。

## 许可证

本项目遵循 [MIT License](./LICENSE)。
