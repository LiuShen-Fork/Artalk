# 安全防护

> 本功能由本仓库（LiuShen-Fork）新增，上游 Artalk 没有 `security` 配置段，也不会拦截超限请求。
> 它对官方前端组件完全透明：超限时的 `need_captcha` / `img_data` / `iframe` 响应与原有的
> [验证码](./captcha.md) 逻辑一致，前端无需任何改动。

安全防护在 Artalk 原有的「操作计数 + 验证码」之上，提供一套完整的 **限频 → 拦截 → 记录 → 自动封禁** 链路：

- **限频**：同时按「单 IP」和「全站」统计，窗口分为每分钟、每小时、每天；
- **超限动作**：要求验证码、直接拒绝（429）、或自动把 IP 加入黑名单；
- **自动封禁**：一段时间内反复被拦截的 IP 自动进入黑名单；
- **攻击报表**：按类型、日期、IP、路径汇总拦截情况；
- **攻击记录**：逐条保留被拦截的请求（IP、路径、UA、站点、账号、计数），便于回查与手动封禁。

配置可以在 [控制中心](../frontend/sidebar.md) 的「安全」页面在线修改并立即生效（只改动配置文件中的
`security` 与 `captcha` 两段，保留文件中的注释与其他配置，不会重启服务）。

## 配置文件

完整的 `security` 配置如下，默认全部关闭：

```yaml
# 安全防护 (IP 封禁、限频、验证码挑战、攻击报表)
security:
  enabled: false # 总开关
  action: captcha # 超限动作：captcha | reject | ban
  rate_limit:
    ip:
      minute: 0 # 单个 IP 每分钟允许的请求数，0 表示不限制
      hour: 0
      day: 0
    global:
      minute: 0 # 全站每分钟允许的请求数，0 表示不限制
      hour: 0
      day: 0
  auto_ban:
    enabled: false # 自动封禁反复被拦截的 IP
    threshold: 5 # 统计窗口内允许的违规次数
    window: 3600 # 统计窗口 (单位：s)
    duration: 86400 # 封禁时长 (单位：s，-1 表示永久)
  blacklist: [] # 永久黑名单，支持单个 IP 与 CIDR 网段
  whitelist: [] # 白名单，白名单 IP 不计数、不封禁
  event_retention_days: 90 # 攻击记录保留天数
  exclude_paths: [] # 额外排除的路径，默认为 /captcha/verify
```

::: tip
`action: captcha` 依赖 `captcha.enabled: true`。若验证码未启用，配置加载时会自动降级为
`action: reject` 并输出一条警告日志，避免出现「要求验证码但发不出验证码」的情况。
:::

## 来源 IP 与可信代理

限频、封禁、攻击记录都建立在「来源 IP」之上。当 Artalk 位于反向代理或 CDN 之后，真实 IP 只能从
代理头读取（`http.proxy_header`，默认 `X-Forwarded-For`）；而这个头客户端可以自己构造——
如果无条件放行，攻击者每次换一个假 IP 就能拿到新的计数桶，限频与封禁都形同虚设。

```yaml
http:
  proxy_header: X-Forwarded-For # 代理头名，直连部署请留空
  trusted_proxies: # 只有这些地址发来的代理头才作数（IP 或 CIDR）
    - 127.0.0.1
    - 10.0.0.0/8
```

- 留空（默认）：保持旧行为，任何来源的代理头都生效；
- 填写后：连接地址不在列表中的请求按连接地址计数，伪造的 `X-Forwarded-For` 不再起作用。

::: warning
只有把代理所在网段写进 `trusted_proxies` 时，`proxy_header` 才有意义；直连部署（没有代理）
请把 `proxy_header` 留空，此时来源 IP 取连接地址，无法伪造。
:::

## 统计规则

- 只统计**写操作**请求（`POST` / `PUT` / `PATCH` / `DELETE`），`GET` / `HEAD` / `OPTIONS` 不计数；
- 管理员（持有有效管理员 Token）与白名单 IP 的请求不计数、不拦截；
- `exclude_paths` 中的路径跳过（默认包含 `/captcha/verify`，否则验证码请求本身会被限频卡住）；
- 计数在内存中完成，单进程独立计数，重启后清零；多实例部署时每个实例各自计数，如需全局配额请使用共享存储或前置网关限频；
- 计数使用固定窗口，统计实现为 O(1) 内存，代价是在窗口边界处最多可能出现两倍突发。

超限时的响应：

| 动作 | 状态码 | 响应体 |
| --- | --- | --- |
| `reject` | 429 | `{"rate_limited": true, "retry_after": <秒>}`，同时带 `Retry-After` 响应头 |
| `captcha` | 403 | `{"need_captcha": true, "img_data": "..."}` 或 `{"need_captcha": true, "iframe": true}` |
| `ban` | 403 | `{"ip_banned": true}` |

## 自动封禁

开启 `auto_ban.enabled` 后，每次请求被拦截都会为该 IP 记一次违规；在 `window` 秒内累计达到
`threshold` 次时，该 IP 会被自动写入黑名单（来源标记为 `auto`），时长取自 `auto_ban.duration`。

- `threshold`、`window`、`duration` 填 `0` 时分别回退到默认值 `5`、`3600`、`86400`；
- `duration: -1` 表示永久封禁；
- 同一 IP 再次被自动封禁时会更新已有的黑名单条目，不会重复插入；
- 手动封禁的条目在被自动封禁覆盖时保留原操作人。

## 黑名单与白名单

- 黑名单支持单个 IP（`203.0.113.9`）与 CIDR 网段（`10.0.0.0/8`、`2001:db8::/32`）；
- 配置文件中的 `security.blacklist` 会在启动时以及每小时同步到内存，这类条目在控制中心里
  是**只读**的（`source: config`），需要删除时请修改配置文件；
- 控制中心里手动/自动封禁的条目保存在数据库 `ip_bans` 表中，可直接解封；
- 白名单优先于黑名单：白名单中的 IP 即使命中黑名单也不会被拦截。

## 攻击记录与报表

- 每条被拦截的请求都会写入 `security_events` 表（同一 IP、同一类型、同一窗口内每分钟最多记录一条，避免刷屏）；
- 记录包含 IP、IP 属地、类型（`rate_limit` / `captcha_failed` / `banned` / `auto_ban`）、作用域、窗口、
  实际计数、路径、方法、站点、账号、UA 与时间；
- 超过 `event_retention_days` 的记录会每小时自动清理；
- 「安全」页面提供最近 1 / 7 / 30 天的报表：总记录数、拦截数、按类型、按日期、高频 IP、高频路径，
  并对高频 IP 提供一键封禁。

验证码校验失败时也会记录一条 `captcha_failed` 事件，并计入自动封禁的违规次数。

## 接口

所有接口都需要管理员身份（`AdminGuard`）：

| 方法 | 路径 | 说明 |
| --- | --- | --- |
| `GET` | `/api/v2/security/config` | 读取 `security`、`captcha` 配置与运行时状态 |
| `PUT` | `/api/v2/security/config` | 部分更新配置，仅改动 `security` 与 `captcha` 两段，立即生效 |
| `GET` | `/api/v2/security/stats` | 运行时统计（追踪 IP 数、生效封禁数、全站计数），可带 `ip` 参数查询单个 IP |
| `GET` | `/api/v2/security/events` | 攻击记录列表，支持 `ip`、`kind`、`scope`、`days`、`limit`、`offset` |
| `DELETE` | `/api/v2/security/events/:id` | 删除单条记录 |
| `DELETE` | `/api/v2/security/events` | 按筛选条件清空记录 |
| `GET` | `/api/v2/security/report` | 攻击报表，`days` 默认 7，最大 365 |
| `GET` | `/api/v2/security/bans` | 黑名单列表，支持 `ip`、`active`、`limit`、`offset` |
| `POST` | `/api/v2/security/bans` | 手动封禁，`duration` 单位秒，`<= 0` 表示永久 |
| `DELETE` | `/api/v2/security/bans/:id` | 解封（配置黑名单条目返回 400） |

## 与原有验证码配置的关系

启用 `security.enabled` 后，原有的 `LimiterGuard`（`captcha.action_limit` 的计数）会自动让位，
由安全防护统一计数与挑战，避免同一请求被两次计数或连续弹出两次验证码。

未启用 `security.enabled` 时，行为与上游完全一致。
