# Technology Stack: v1.2 New Capabilities

**Project:** Cloud CLI Proxy v1.2 -- 用户自助面板与 Bootstrap 重设计
**Researched:** 2026-03-28
**Confidence:** HIGH
**Scope:** 仅涵盖 v1.2 新增功能所需的技术选型，不重复已验证的 v1.0/v1.1 基线。

---

## 核心结论：零新依赖

v1.2 的全部五项新功能均可通过扩展现有代码实现，**不需要引入任何新的 Go 模块或 npm 包**。这是因为 v1.0/v1.1 的技术选型已经足够覆盖 v1.2 所需能力。

---

## 1. 用户认证（区别于管理员 JWT）

### 现状分析

当前管理员认证使用 `golang-jwt/jwt/v5` + HS256，claims 中 Subject 固定为 `"admin"`，单用户名/密码硬编码在配置中（`AdminConfig` 结构体）。用户侧仅在 bootstrap entry 流程中使用明文 `entry_password` 做一次性认证（无 token、无会话）。用户表已有 `password_hash` 列（bcrypt）。

### 推荐方案：扩展现有 JWT 体系，增加 role claim

| 技术 | 版本 | 用途 | 推荐原因 |
|------|------|------|----------|
| `golang-jwt/jwt/v5` | v5.x (已有) | 用户 JWT 签发与校验 | 已在项目中使用，无需引入新依赖。扩展 claims 加入 `role`、`user_id`、`username` 即可。 |
| `golang.org/x/crypto/bcrypt` | (已有) | 用户密码校验 | 用户表已有 `password_hash` 列，bootstrap 流程已有 bcrypt 验证逻辑。 |

**不需要新增任何后端依赖。**

### 实现策略

**后端：**

```go
// 统一 claims 结构，替代当前的 jwt.RegisteredClaims 裸用
type AppClaims struct {
    jwt.RegisteredClaims
    Role     string `json:"role"`     // "admin" | "user"
    UserID   string `json:"user_id"`  // UUID，仅 role=user 时有值
    Username string `json:"username"`
}
```

- 新增 `POST /v1/user/login` 端点，接收 username + password，校验 `users.password_hash`（bcrypt），签发带 `role: "user"` 的 JWT。
- 新增 `UserAuthMiddleware`，解析 JWT 并从 claims 提取 `user_id`，注入 request context。
- 用户 API 路由前缀 `/v1/user/` 下挂载该中间件，所有查询自动限定 `WHERE user_id = $ctx.user_id`。
- 管理员 JWT 保持不变（`role: "admin"`，Subject: `"admin"`），可同步升级为 `AppClaims` 但行为不变。
- 两种 JWT 使用同一个 `jwtSecret` 签名——通过 `role` claim 区分权限边界。

**前端：**

| 技术 | 版本 | 用途 | 推荐原因 |
|------|------|------|----------|
| TanStack Router (已有) | ^1.120.0 | 路由分割：`/_admin` 和 `/_user` 两套 layout | 已有 file-based routing，新增路由组即可。`beforeLoad` 钩子做角色检查。 |

- 扩展 `lib/auth.ts`：存储 token 时同时解析 role（从 JWT payload base64 解码，无需验签——验签在后端）。
- 登录页统一入口，后端返回 token 后前端根据 role 跳转到 `/_admin` 或 `/_user` 布局。
- 用户侧布局复用现有 Sidebar/Topbar 组件，仅渲染不同菜单项。

### 明确不做

| 不做 | 原因 | 替代 |
|------|------|------|
| OAuth2 / OIDC | 用户规模小，自有用户体系足够，引入外部 IdP 增加部署复杂度 | 自签 JWT + bcrypt |
| Refresh Token | v1.2 用户面板低频使用，24 小时 token 过期后重新登录即可 | 单 access token，24h TTL |
| 前端 JWT 验签 | 前端无需验签，role 提取仅用于路由分发，安全边界在后端中间件 | base64 解码 payload |
| 独立认证微服务 | 单宿主机场景，控制面内嵌即可 | 控制面内新增 handler |
| 独立 JWT secret | 共享 secret 通过 role claim 隔离足够安全，拆分 secret 增加配置但无额外安全收益 | 同一 secret + role 字段 |

**Confidence:** HIGH -- 所有技术均已在项目中验证，仅做逻辑扩展。

---

## 2. KasmVNC 嵌入用户面板

### 现状分析

- KasmVNC 已在受管容器中安装并运行于 6080 端口（`entrypoint.sh` 中配置）。
- `AdminVNCProxyHandler` 已实现完整的 HTTP + WebSocket 反向代理（包含路径重写和 hijack 双向转发）。
- KasmVNC 配置为无密码模式（`kasmvnc.yaml` 中 `require_ssl: false`，无 auth 配置），安全性由控制面反代保护。
- 管理员通过 `/v1/admin/hosts/{hostID}/vnc/{path...}` 访问，admin JWT 认证。
- Admin auth 中间件已支持 query param token（`?token=xxx`）+ cookie 回写，专为 VNC 子资源加载设计。

### 推荐方案：复用现有 VNC 代理，新增用户路由

| 技术 | 版本 | 用途 | 推荐原因 |
|------|------|------|----------|
| 现有 `AdminVNCProxyHandler` 逻辑 | -- | 提取为通用 `VNCProxyHandler`，同时服务管理员和用户路由 | 反代 + WebSocket hijack 逻辑已验证可用，不需要重写。 |

**不需要新增任何依赖。**

### 实现策略

**后端：**

- 重构 `AdminVNCProxyHandler` 为通用 `VNCProxyHandler`，接受一个 `HostResolver` 接口（给定请求上下文 + hostID，返回 host 详情或报错）。
- 管理员路由的 resolver：直接查询，无额外限制。
- 用户路由的 resolver：查询后校验 `host.user_id == ctx.user_id`，不匹配返回 403。
- 用户路由注册为 `/v1/user/hosts/{hostID}/vnc/{path...}`，挂载 UserAuthMiddleware。
- WebSocket 升级路径不变，token 通过 query param 传入后 cookie 回写（复用已有逻辑，cookie path 改为 `/v1/user/`）。

**前端：**

- 用户面板嵌入 `<iframe>` 指向 `/v1/user/hosts/{hostID}/vnc/?token=xxx`。
- iframe 属性：`sandbox="allow-scripts allow-same-origin"` + `allow="clipboard-read; clipboard-write"`。
- KasmVNC 前端代码（已打包在容器内）通过反代直接提供 HTML/JS/CSS，无需额外前端库。

### 关键注意事项

- **同源策略**：VNC 反代挂在同一域名下（`/v1/user/hosts/...`），iframe 与主应用同源，无 CORS 问题。
- **WebSocket 升级**：Go `net/http` 的 `Hijack()` 已验证工作正常，不需要额外 WebSocket 库（gorilla/websocket 等）。
- **Cookie 作用域**：用户 token cookie 的 Path 应设为 `/v1/user/` 以防与 admin cookie（Path `/v1/admin/`）冲突。
- **nginx 配置**：如果使用 nginx 反代，需确保 WebSocket 升级头透传 + `proxy_buffering off`（参考 KasmVNC 官方文档）。

### 明确不做

| 不做 | 原因 | 替代 |
|------|------|------|
| 引入 noVNC 或 guacamole 等第三方 VNC 客户端 | KasmVNC 自带完整 Web 客户端，功能更完善（剪贴板、文件传输等） | 使用 KasmVNC 内置 Web 客户端 |
| WebSocket 库（gorilla/websocket 等） | 现有 hijack 方案已验证，WebSocket 库反而增加复杂度且有 API 差异 | 继续使用 `net/http` Hijacker |
| 独立 VNC 认证层 | KasmVNC 已配置为无密码模式，安全由反代 JWT 保护 | 反代层的 JWT 中间件 |
| 独立 VNC 反向代理进程 | 控制面已能做 HTTP/WS 代理，无需额外进程 | 控制面内 handler 层代理 |

**Confidence:** HIGH -- 现有 VNC 代理代码已完整工作，仅需提取通用化 + 新增路由。

---

## 3. Bootstrap ASCII 艺术字欢迎横幅

### 现状分析

当前 bootstrap 脚本（`/entry/{shortId}` 返回的 shell 脚本）是极简的：读密码 -> curl 认证 -> SSH 接入。无欢迎信息、无品牌展示。脚本由 Go `EntryHandler.Script()` 方法动态生成（`fmt.Sprintf` 模板）。

### 推荐方案：在 Go 端生成的 shell 脚本中内嵌预渲染 ASCII art

| 技术 | 版本 | 用途 | 推荐原因 |
|------|------|------|----------|
| 硬编码 ASCII art 字符串 | -- | 在 Go 模板中直接嵌入预渲染的 ASCII 横幅 | 零依赖、跨平台、不要求用户机器安装 figlet。 |
| ANSI 转义码 | -- | 颜色和样式 | 所有现代终端都支持，`\033[` 序列足够。 |

**不需要新增任何依赖。**

### 实现策略

- 在开发机上用 figlet（`figlet -f slant "Cloud CLI Proxy"` 或类似命令）生成品牌 ASCII art。
- 将生成结果存为 Go 常量字符串（`const asciiBanner = ...`）。
- 在 `EntryHandler.Script()` 生成的 bash 脚本开头插入 `cat << 'BANNER'` heredoc 块，输出带 ANSI 颜色的横幅。
- 横幅内容：产品名 ASCII art + 用户名问候 + 简短说明行。
- 使用 `\033[36m`（cyan）作为主色调，`\033[0m` 重置，增加视觉层次但不过度花哨。

### 为什么不在运行时调用 figlet

- 用户机器不保证安装 figlet/toilet（macOS 需 homebrew，部分 Linux 未预装）。
- bootstrap 脚本的设计约束是**零外部依赖**——仅依赖 bash + curl + ssh。
- 预渲染 ASCII art 嵌入脚本后体积增加约 500 字节，完全可接受。

### 明确不做

| 不做 | 原因 | 替代 |
|------|------|------|
| 在 Go 端引入 figlet 库（如 `go-figure`、`github.com/common-nighthawk/go-figure`） | 运行时生成不如预渲染可控，且增加依赖 | 预渲染常量 |
| 要求用户安装 figlet | 破坏"零依赖一条命令"体验 | 脚本内嵌 |
| 复杂动画效果（逐字打印等） | `curl | bash` 管道模式下动画效果不可靠（取决于缓冲行为） | 静态横幅 + ANSI 颜色 |
| lolcat 彩虹效果 | 终端兼容性不确定，视觉效果分散注意力 | 单色调 ANSI |

**Confidence:** HIGH -- 纯字符串嵌入，无技术风险。

---

## 4. Bootstrap 实时状态流推送

### 现状分析

当前 bootstrap 流程涉及三个 API：
1. `POST /v1/entry/{shortId}/auth` -- 认证 + 获取 SSH 信息。如果主机未运行，返回 `status: "not_ready"`。
2. `GET /v1/bootstrap/tasks/{taskID}` -- 轮询任务状态（JSON 快照，非流式）。使用 `resolveStage()` 将事件映射为可读状态。
3. `GET /v1/bootstrap/tasks/{taskID}/handoff` -- 获取 SSH 连接信息。

v1.2 需要将步骤 2 从客户端轮询改为服务端主动推送。

### 推荐方案：Go 标准库 SSE（Server-Sent Events）

| 技术 | 版本 | 用途 | 推荐原因 |
|------|------|------|----------|
| Go `net/http` + `http.Flusher` | Go 1.26.1 (已有) | 服务端 SSE 推送 | Go 标准库原生支持 SSE 所需的 chunked 传输和 flush 控制，无需第三方库。实现量约 30-40 行代码。 |
| `curl -N` (client side) | -- | 客户端接收 SSE 流 | curl 的 `--no-buffer` 模式天然支持逐行读取 SSE 文本流，配合 bash `while read` 循环解析即可。 |

**不需要新增任何后端或前端依赖。**

### 实现策略

**后端：**

新增 `GET /v1/bootstrap/tasks/{taskID}/stream` SSE 端点：

```go
// 关键 headers
w.Header().Set("Content-Type", "text/event-stream")
w.Header().Set("Cache-Control", "no-cache")
w.Header().Set("Connection", "keep-alive")
w.Header().Set("X-Accel-Buffering", "no") // nginx 反代时禁用缓冲

// 事件循环
ticker := time.NewTicker(2 * time.Second)
for {
    select {
    case <-r.Context().Done():
        return // 客户端断开
    case <-ticker.C:
        stage := resolveStage(ctx, events, taskID, task.Status) // 复用现有逻辑
        fmt.Fprintf(w, "data: {\"stage_code\":\"%s\",\"stage_text\":\"%s\",\"task_status\":\"%s\"}\n\n",
            stage.code, stage.text, task.Status)
        flusher.Flush()
        if task.Status == "succeeded" || task.Status == "failed" {
            return // 终态，关闭连接
        }
    }
}
```

- 轮询间隔 2 秒，服务端查询 task 状态 + 最新 event，推送 SSE data 行。
- 完全复用现有的 `resolveStage()` 函数和 `stagesByEventType` 映射表。
- 心跳：每 15 秒发送 `:heartbeat\n\n` SSE 注释行，防止代理超时断开。
- 终态（succeeded / failed）推送后主动关闭连接。

**客户端（bash 脚本）：**

```bash
curl -sN "${BASE_URL}/v1/bootstrap/tasks/${TASK_ID}/stream" | while IFS= read -r line; do
    case "$line" in
        data:*)
            json="${line#data: }"
            stage=$(echo "$json" | grep -o '"stage_text":"[^"]*"' | cut -d'"' -f4)
            status=$(echo "$json" | grep -o '"task_status":"[^"]*"' | cut -d'"' -f4)
            printf "\r\033[K  %s %s" "$spinner_char" "$stage"
            [ "$status" = "succeeded" ] && break
            [ "$status" = "failed" ] && { echo "\nError: $stage"; exit 1; }
            ;;
    esac
done
```

### 为什么选 SSE 而非 WebSocket

| 比较维度 | SSE | WebSocket |
|----------|-----|-----------|
| curl 兼容 | 原生支持（`curl -N`） | 不支持，需要 `websocat` 等额外工具 |
| 实现复杂度 | 单向推送，Go 标准库 ~30 行 | 双向协议，需第三方库或手写协议升级 |
| 使用场景匹配度 | 服务端→客户端单向推送 -- 完全匹配 | 需要双向通信时才有优势 |
| 代理穿透性 | 普通 HTTP，穿透性好 | 部分代理/防火墙可能拦截升级请求 |
| 自动重连 | SSE 规范内置 retry 机制 | 需客户端自行实现 |

### 明确不做

| 不做 | 原因 | 替代 |
|------|------|------|
| 引入 SSE 库（如 `r3labs/sse`） | 标准库实现仅 ~30 行，库的 pub/sub 抽象对单连接场景过重 | Go `net/http` + `Flusher` |
| WebSocket | curl 不原生支持 WebSocket，破坏"一条命令"体验 | SSE |
| 引入消息队列（Redis pub/sub 等） | 单宿主机单进程，直接轮询 DB 即可，消息队列是过度工程化 | 直接查询 PostgreSQL |
| 长轮询 | SSE 更标准化且效率更高 | SSE |

**Confidence:** HIGH -- Go SSE 是广泛使用的 pattern，curl -N 兼容性经官方 discussion 确认。

---

## 5. Multi-Claude-Account 数据模型

### 现状分析

当前数据模型：
- `users` 1:N `hosts`（通过 `hosts.user_id`），实际通过 `UNIQUE(user_id, slot_key)` 按 slot 区分。
- 无 Claude 账号概念，无 Claude 相关表或字段。
- `hosts` 表不关联任何外部服务账号。

### 推荐方案：新增 `claude_accounts` 表 + 外键关联

| 技术 | 版本 | 用途 | 推荐原因 |
|------|------|------|----------|
| PostgreSQL 18.x (已有) | 18.3 | 新增 `claude_accounts` 表 | 标准关系模型，不需要新技术。模式与已有的 `egress_ips` + `host_egress_bindings` 一致。 |
| pgx v5.x (已有) | v5.x | 查询新表 | 已有驱动，新增 repository 方法即可。 |

**不需要新增任何依赖。**

### 数据模型设计

```sql
-- Migration: 0006_claude_accounts.sql

CREATE TABLE claude_accounts (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id     UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    label       TEXT NOT NULL,                    -- 管理员可读标签，如 "Claude Pro #1"
    email       TEXT NOT NULL,                    -- Claude 账号邮箱
    account_type TEXT NOT NULL DEFAULT 'pro',     -- 'pro' | 'team' | 'enterprise'
    status      TEXT NOT NULL DEFAULT 'active',   -- 'active' | 'suspended' | 'expired'
    host_id     UUID REFERENCES hosts(id) ON DELETE SET NULL,  -- 绑定的主机（可空 = 未绑定）
    metadata    JSONB NOT NULL DEFAULT '{}'::jsonb,  -- 灵活扩展字段（团队信息等）
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- 一个 Claude 账号只能绑定一台主机（部分唯一索引，允许多个 NULL）
CREATE UNIQUE INDEX idx_claude_accounts_host_id ON claude_accounts(host_id) WHERE host_id IS NOT NULL;
-- 一个用户下 email 唯一
CREATE UNIQUE INDEX idx_claude_accounts_user_email ON claude_accounts(user_id, email);
-- 快速查找用户的所有账号
CREATE INDEX idx_claude_accounts_user_id ON claude_accounts(user_id);
```

### 关系图

```
users (1) ──┬── (N) claude_accounts
             │           │
             │           │ host_id (0..1)
             │           ▼
             └── (N) hosts (1) ──── (N) host_egress_bindings ──── egress_ips
```

### 关键设计决策

| 决策 | 原因 |
|------|------|
| `host_id` 放在 `claude_accounts` 侧 | 一个 Claude 账号绑定一台主机，但可暂时不绑定（host_id = NULL）。正向关联比中间表更直观。 |
| 直接外键而非中间表 | 关系是 1:1（一个 Claude 账号至多绑定一台主机），中间表是过度设计。 |
| `metadata` JSONB 列 | 不同类型的 Claude 账号可能有不同属性，JSONB 保留灵活性。与 `egress_ips.proxy_config` 的设计模式一致。 |
| `ON DELETE SET NULL` for host_id | 主机删除/重建时 Claude 账号保留，可以重新绑定到新主机。 |
| `ON DELETE CASCADE` for user_id | 用户删除时级联清理其所有 Claude 账号。 |
| 部分唯一索引 `WHERE host_id IS NOT NULL` | 允许多个未绑定账号（host_id = NULL），但已绑定的 host_id 必须唯一。 |

### API 设计

**管理员 API（`/v1/admin/` 前缀，admin JWT 认证）：**
- `GET /v1/admin/claude-accounts` -- 列表（支持 `?user_id=` 过滤）
- `POST /v1/admin/claude-accounts` -- 创建（指定 user_id）
- `GET /v1/admin/claude-accounts/{id}` -- 详情
- `PUT /v1/admin/claude-accounts/{id}` -- 更新
- `DELETE /v1/admin/claude-accounts/{id}` -- 删除
- `POST /v1/admin/claude-accounts/{id}/bind` -- 绑定主机（body: `{host_id}`）
- `POST /v1/admin/claude-accounts/{id}/unbind` -- 解绑主机

**用户 API（`/v1/user/` 前缀，user JWT 认证）：**
- `GET /v1/user/claude-accounts` -- 查看自己的 Claude 账号列表（自动限定 user_id = ctx.user_id）

### 敏感字段处理

- Claude 账号的密码/session token **不存入数据库** -- 这些信息已在容器环境中预配置。
- `metadata` JSONB 中明确不存储任何密码类信息。
- 用户 API 返回时仅展示：label、email、account_type、status、绑定状态（有/无主机 + 主机状态）。

### 明确不做

| 不做 | 原因 | 替代 |
|------|------|------|
| 多对多 `claude_accounts <-> hosts` 中间表 | 关系是 1:1，过度设计 | `claude_accounts.host_id` 直接外键 |
| 存储 Claude API key / session token | 安全风险高，且容器环境已预配置 | 仅存储展示信息 |
| 自动同步 Claude 账号状态 | v1.2 无需与 Anthropic API 对接 | 管理员手动管理状态 |

**Confidence:** HIGH -- 标准关系模型扩展，模式与项目已有的 egress_ips / bindings 完全一致。

---

## 总体依赖变化汇总

### 后端新增 Go 依赖：无

| 功能 | 使用的技术 | 新增依赖 |
|------|-----------|----------|
| 用户认证 | golang-jwt/jwt/v5 (已有) + bcrypt (已有) | 无 |
| KasmVNC 嵌入 | VNCProxyHandler (已有逻辑) + net/http (标准库) | 无 |
| ASCII 横幅 | 预渲染字符串常量 | 无 |
| 实时状态流 | net/http + http.Flusher (标准库) | 无 |
| Claude 账号模型 | PostgreSQL + pgx (已有) | 无 |

### 前端新增 npm 依赖：无

| 功能 | 使用的技术 | 新增依赖 |
|------|-----------|----------|
| 用户登录/路由拆分 | TanStack Router beforeLoad (已有) | 无 |
| VNC 嵌入 | 原生 `<iframe>` | 无 |
| Claude 账号展示 | TanStack Query (已有) + 现有 UI 组件 | 无 |

### 数据库变更

| 变更 | 文件 |
|------|------|
| 新增 `claude_accounts` 表 + 3 个索引 | `0006_claude_accounts.sql` |

### Nginx 配置变更

| 变更 | 说明 |
|------|------|
| 短 URL 重写规则 | `location ~ ^/[a-zA-Z0-9]{6,8}$` 重写到 `/entry/$1`（避免 Go 路由冲突） |
| SSE 不缓冲 | bootstrap stream 端点添加 `proxy_buffering off` + `X-Accel-Buffering: no` |
| 用户 VNC WebSocket 透传 | `/v1/user/hosts/*/vnc/` 路径添加 `Upgrade` 和 `Connection` 头透传 |

---

## Alternatives Considered

| 分类 | 推荐方案 | 备选方案 | 什么时候才考虑备选 |
|------|---------|---------|-------------------|
| 用户认证 | 共享 JWT secret + role claim | 独立 JWT secret / OAuth2 | 只有当用户规模超过数千且需要第三方登录时 |
| 实时推送 | SSE (text/event-stream) | WebSocket | 只有当需要客户端→服务端双向通信时（bootstrap 不需要） |
| 前端路由 RBAC | TanStack Router beforeLoad | 自定义 HOC 包裹 | 不推荐——beforeLoad 是 TanStack Router 官方推荐模式 |
| 短 URL 路由 | Nginx location 正则重写 | Go 顶级路由 `/{shortId}` | 不推荐——Go 中注册 `/{shortId}` 会与 `/healthz` 等路由冲突 |
| VNC 代理 | 复用现有 VNCProxyHandler | 独立 VNC 网关进程 | 只有当 VNC 流量大到影响控制面性能时 |

## What NOT to Use

| 不要用 | 原因 | 替代方案 |
|--------|------|----------|
| 独立的用户 SPA 应用 | 增加构建/部署/维护成本，两个 SPA 共享大量 UI 组件 | 同一 React SPA，路由层按 role 拆分 |
| OAuth2 / OIDC Provider | 过度工程化，当前只有 admin 和 user 两种角色 | JWT role claim |
| Socket.IO / ws 库用于 Bootstrap 状态 | shell 脚本不好消费 WebSocket，需要额外客户端工具 | SSE（curl -N 即可） |
| gorilla/websocket | 现有 Hijacker 方案工作正常，引入 gorilla 改变了整个 VNC 代理的编程模型 | 继续使用 net/http Hijacker |
| go-figure / figlet Go 库 | 运行时生成 ASCII art 增加依赖但不增加价值 | 预渲染常量字符串 |
| Redis / 消息队列 | 单宿主机单进程，直接查 DB 足够 | PostgreSQL 轮询 |

---

## 来源

- 项目代码 `internal/controlplane/http/admin_auth.go` -- 确认 JWT 实现（HS256、query param token、cookie 回写）
- 项目代码 `internal/controlplane/http/admin_vnc_proxy.go` -- 确认 VNC 反代实现（HTTP + WebSocket hijack）
- 项目代码 `internal/controlplane/http/entry.go` -- 确认 bootstrap 入口和脚本生成逻辑
- 项目代码 `internal/controlplane/http/bootstrap_status.go` -- 确认 resolveStage 和事件映射
- 项目代码 `internal/store/repository/models.go` -- 确认数据模型结构
- 项目代码 `internal/store/migrations/0001_initial.sql` -- 确认表结构和约束
- 项目代码 `deploy/docker/managed-user/entrypoint.sh` -- 确认 KasmVNC 配置
- [KasmVNC reverse proxy docs](https://kasmweb.com/kasmvnc/docs/master/how_to/reverse_proxy.html) -- WebSocket 透传配置
- [golang-jwt/jwt v5](https://pkg.go.dev/github.com/golang-jwt/jwt/v5) -- 自定义 claims 支持
- [Go SSE patterns](https://www.freecodecamp.org/news/how-to-implement-server-sent-events-in-go/) -- 标准库 SSE 实现
- [Go real-time SSE](https://oneuptime.com/blog/post/2026-02-01-go-realtime-applications-sse/view) -- SSE 生产实践
- [curl SSE support](https://github.com/curl/curl/discussions/13395) -- curl -N 对 SSE 的兼容性

---
*Stack research for: v1.2 用户自助面板与 Bootstrap 重设计*
*Researched: 2026-03-28*
