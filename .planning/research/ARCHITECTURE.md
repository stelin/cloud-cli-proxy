# Architecture Research: v1.2 用户自助面板与 Bootstrap 重设计

**Domain:** 用户认证、角色路由、KasmVNC 代理、Claude 账号模型、Bootstrap 重设计、实时状态推送
**Researched:** 2026-03-28
**Confidence:** HIGH

## 现有架构基线

```
                    Nginx (:80)
                   /          \
           /v1/ ->             / ->
     Control Plane (:8080)    React SPA (静态)
      |          |
      |    AdminAuthMiddleware (JWT, sub="admin")
      |
  Unix Socket -> Host Agent -> Docker / Network
      |
   PostgreSQL
      |
  SSH Proxy (:2222) -> Container SSH (:22 via mgmt net)
```

**当前 JWT 模型：** 管理员专用。`AdminLoginHandler` 校验硬编码用户名/密码，签发 `sub=admin` 的 HS256 JWT，24 小时有效。无 role claim。
**当前路由守卫：** `_dashboard.tsx` 的 `beforeLoad` 仅检查 `localStorage.admin_token` 是否存在。
**当前 VNC 代理：** `AdminVNCProxyHandler` 已实现 HTTP 反向代理 + WebSocket hijack，路径重写 `/v1/admin/hosts/{hostID}/vnc/{path...}` -> 容器 `:6080`。
**当前 Bootstrap：** `GET /entry/{shortId}` 返回 shell 脚本 -> `POST /v1/entry/{shortId}/auth` 认证 -> 返回 SSH 连接信息。无实时状态推送。
**当前用户模型：** `users` 表有 `short_id`、`entry_password`（明文）。一个用户对应一个 host（通过 `GetPrimaryHostByUserID`）。

---

## 1. 用户认证：共享 JWT + role claim

### 方案

**使用同一套 JWT secret，通过 role claim 区分管理员和普通用户。** 不要拆成两套独立的 JWT 签发体系。

理由：
- 当前只有一个 JWT secret（`AdminJWTSecret`），复用它可以避免配置膨胀
- 管理员和用户共享同一个 React SPA，同一个 Nginx 入口，拆 secret 没有实际安全收益
- role claim 是 JWT RBAC 的标准做法

### 具体变更

**后端 — 新文件：`internal/controlplane/http/user_auth.go`**

```go
// 自定义 claims，扩展 RegisteredClaims
type AppClaims struct {
    jwt.RegisteredClaims
    Role   string `json:"role"`   // "admin" | "user"
    UserID string `json:"uid"`    // 用户 UUID（仅 role=user 时有值）
}
```

**新增 `POST /v1/auth/login` 端点：**
- 接受 `{ username, password }` 请求
- 先尝试管理员凭证匹配（保持 hmac.Equal 常量时间比较）
- 若管理员匹配：签发 `role=admin, sub=admin`
- 若管理员不匹配：查 `users` 表，bcrypt 验证 `password_hash`，签发 `role=user, uid=<user_id>, sub=<username>`
- 统一返回 `{ token, expires_in, role }`

**修改现有文件：`admin_auth.go` -> 重命名为 `auth_middleware.go`**

```go
// AuthMiddleware 校验 JWT 并注入 claims 到 context
func AuthMiddleware(secret []byte) func(http.Handler) http.Handler

// RequireRole 从 context 读取 claims，检查角色
func RequireRole(roles ...string) func(http.Handler) http.Handler

// GetClaims 从 context 获取 AppClaims
func GetClaims(ctx context.Context) (*AppClaims, bool)
```

**路由层变更（`router.go`）：**

| 路由前缀 | 中间件 | 说明 |
|----------|--------|------|
| `/v1/admin/*` | `AuthMiddleware` + `RequireRole("admin")` | 保持现有行为，仅管理员 |
| `/v1/me/*` | `AuthMiddleware` + `RequireRole("user")` | 新增，用户自助 API |
| `/v1/auth/login` | 无 | 新增，统一登录 |
| `/v1/bootstrap/*` | 无 | 保持现有 |
| `/entry/{shortId}` | 无 | 保持现有 |

**迁移路径：** `POST /v1/admin/login` 保留为 `/v1/auth/login` 的别名，前端切换后可废弃。

**数据库变更：** `users` 表已有 `password_hash` 列。当前管理后台创建用户时会写入 bcrypt hash。无需 schema 变更。

### 置信度：HIGH

JWT role claim 是 golang-jwt/jwt/v5 的标准用法，无第三方依赖。

---

## 2. React 角色路由守卫

### 方案

**使用 TanStack Router 的 `beforeLoad` + Router Context 实现 RBAC。** 官方文档已有 [RBAC 指南](https://tanstack.com/router/v1/docs/framework/react/how-to/setup-rbac)。

### 具体变更

**修改 `lib/auth.ts`：**

```typescript
interface AuthState {
  token: string | null
  role: 'admin' | 'user' | null
  userId: string | null
}

// 解码 JWT payload（不验签，前端仅做 UI 展示用）
function parseJwtPayload(token: string): { role: string; uid?: string; sub: string }

export function getAuth(): AuthState
export function setAuth(token: string): void
export function clearAuth(): void
export function hasRole(role: string): boolean
```

**修改 `routes/__root.tsx`：** 在 Router Context 注入 auth 状态

```typescript
export const Route = createRootRouteWithContext<{ auth: AuthState }>()({
  component: RootLayout,
})
```

**路由结构重组：**

```
routes/
  __root.tsx              # 注入 auth context
  login.tsx               # 统一登录页（替换 /v1/admin/login）
  _dashboard.tsx          # beforeLoad: requireAuth()
  _dashboard/
    index.tsx             # 按 role 重定向
    _admin.tsx            # beforeLoad: requireRole('admin')
    _admin/
      users/              # 现有管理员页面
      hosts/
      egress-ips/
      events/
      tasks/
      claude-accounts/    # 新增：Claude 账号管理
    _user.tsx             # beforeLoad: requireRole('user')
    _user/
      index.tsx           # 用户仪表盘
      hosts/              # 我的主机
      vnc/{hostId}.tsx    # KasmVNC 入口
```

**关键模式 — `_admin.tsx` 的 `beforeLoad`：**

```typescript
export const Route = createFileRoute('/_dashboard/_admin')({
  beforeLoad: ({ context }) => {
    if (context.auth.role !== 'admin') {
      throw redirect({ to: '/' })
    }
  },
  component: AdminLayout,
})
```

**API 层变更（`lib/api.ts`）：**

新增 `userApiFetch`，基础路径改为 `/v1/me`。现有 `apiFetch` 保持 `/v1/admin` 前缀。共享 token 存储和 401 处理逻辑。

### 置信度：HIGH

TanStack Router 官方文档明确支持此模式。

---

## 3. KasmVNC WebSocket 代理

### 当前状态

`AdminVNCProxyHandler` 已完整实现了 HTTP 反向代理 + WebSocket hijack 转发到容器 `:6080`。位于 `/v1/admin/hosts/{hostID}/vnc/{path...}`，受 `adminAuth` 中间件保护。

### 方案

**复用现有 `AdminVNCProxyHandler`，为用户面新增路由，加用户权限校验。**

### 具体变更

**新增路由（`router.go`）：**

```go
// 用户面 VNC 代理
mux.Handle("/v1/me/hosts/{hostID}/vnc/{path...}",
    authMiddleware(requireRole("user")(userVNCProxy)))
```

**新增 `UserVNCProxyHandler`（`user_vnc_proxy.go`）：**

与 `AdminVNCProxyHandler` 逻辑几乎一致，唯一区别是**权限校验**：从 context 取出 `uid`，验证该 host 确实属于该用户。可以抽取公共的 VNC 代理逻辑到 `vncProxyCore`，管理员和用户 handler 各自做权限校验后委托。

**Nginx 变更（`nginx.conf`）：**

```nginx
# 现有 /v1/ 规则已能覆盖 /v1/me/hosts/{id}/vnc/ 路径
# 但需要添加 WebSocket 升级支持
location /v1/ {
    proxy_pass http://control-plane:8080;
    proxy_set_header Host $host;
    proxy_set_header X-Real-IP $remote_addr;
    proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;

    # WebSocket 支持（KasmVNC）
    proxy_http_version 1.1;
    proxy_set_header Upgrade $http_upgrade;
    proxy_set_header Connection $connection_upgrade;
    proxy_read_timeout 1800s;
    proxy_buffering off;
}
```

需要在 `http` 块或 `server` 块顶部加：

```nginx
map $http_upgrade $connection_upgrade {
    default upgrade;
    '' close;
}
```

**前端 VNC 页面（`_user/vnc/$hostId.tsx`）：**

嵌入 iframe 指向 `/v1/me/hosts/{hostId}/vnc/?token=<jwt>`，KasmVNC 的 web 客户端由容器内的 `:6080` 服务。现有 `AdminVNCProxyHandler` 已通过 query param `token` 设置 cookie 来支持子资源加载。

### 置信度：HIGH

现有代码已验证此模式可行。

---

## 4. Claude 账号数据模型

### 方案

新建 `claude_accounts` 表，建立 `用户 -> Claude 账号 -> 主机` 的一对多对一关系。

### 数据模型

```sql
-- Migration: 0006_claude_accounts.sql

CREATE TABLE claude_accounts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    host_id UUID REFERENCES hosts (id) ON DELETE SET NULL,
    label TEXT NOT NULL DEFAULT '',           -- 显示名称，如 "工作号"、"个人号"
    email TEXT NOT NULL,                      -- Claude 账号邮箱
    account_type TEXT NOT NULL DEFAULT 'pro', -- pro / team / max 等
    status TEXT NOT NULL DEFAULT 'active',    -- active / suspended / expired
    notes TEXT NOT NULL DEFAULT '',           -- 管理员备注
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- 每个 Claude 账号最多绑定一台主机
CREATE UNIQUE INDEX idx_claude_accounts_host_id ON claude_accounts (host_id) WHERE host_id IS NOT NULL;

-- 按用户快速查询
CREATE INDEX idx_claude_accounts_user_id ON claude_accounts (user_id);
```

### 关系图

```
users (1) ---< claude_accounts (N) >--- hosts (0..1)
  |                                        |
  +---- short_id, entry_password           +---- 容器、出口 IP 绑定
  |                                        |
  +---- password_hash (for web login)      +---- VNC, SSH 接入
```

**关键设计决策：**

- **一个用户可以有多个 Claude 账号**（需求明确：一个用户拥有多个 Claude 账号，每个账号对应一台独立主机）
- **一个 Claude 账号最多绑定一台主机**（UNIQUE INDEX on host_id WHERE NOT NULL）
- **host_id 可为 NULL**（账号创建后，主机可以稍后绑定或被管理员回收）
- **删除用户级联删除其所有 Claude 账号**
- **删除主机时 host_id 置 NULL**（账号保留，可重新绑定）

### API 端点

**管理员（`/v1/admin/claude-accounts`）：**

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/v1/admin/claude-accounts` | 列表（支持 ?user_id= 过滤） |
| POST | `/v1/admin/claude-accounts` | 创建 |
| GET | `/v1/admin/claude-accounts/{id}` | 详情 |
| PUT | `/v1/admin/claude-accounts/{id}` | 更新 |
| DELETE | `/v1/admin/claude-accounts/{id}` | 删除 |
| POST | `/v1/admin/claude-accounts/{id}/bind-host` | 绑定主机 |
| POST | `/v1/admin/claude-accounts/{id}/unbind-host` | 解绑主机 |

**用户（`/v1/me/claude-accounts`）：**

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/v1/me/claude-accounts` | 我的 Claude 账号列表 |

### 新增 Go 文件

- `internal/controlplane/http/admin_claude_accounts.go` — 管理员 CRUD handler
- `internal/controlplane/http/user_me.go` — 用户自助面板 handler（含 Claude 账号、主机列表等）
- `internal/store/repository/` 中新增 Claude 账号相关查询方法

### 置信度：HIGH

标准关系模型，无技术风险。

---

## 5. Bootstrap URL 重设计

### 当前路径

- `GET /entry/{shortId}` — 返回 shell 脚本
- `POST /v1/entry/{shortId}/auth` — 认证

### 目标路径

- `GET /{short_id}` — 返回 shell 脚本（短 URL，如 `curl domain/abc123 | bash`）

### 方案

**在 Nginx 层用 try_files + 正则拦截短 ID，转发到控制面。** 不在 Go router 中注册 `/{short_id}`，避免和其他顶级路由冲突。

### 具体变更

**Nginx（`nginx.conf`）：**

```nginx
# 短 ID 格式：6-10 位字母数字
location ~ "^/([a-zA-Z0-9]{6,10})$" {
    proxy_pass http://control-plane:8080/entry/$1;
    proxy_set_header Host $host;
    proxy_set_header X-Real-IP $remote_addr;
    proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
}

# API
location /v1/ {
    proxy_pass http://control-plane:8080;
    # ... WebSocket 等配置
}

# SPA — 放最后
location / {
    try_files $uri $uri/ /index.html;
}
```

**Go router（保持不变）：**

`GET /entry/{shortId}` 端点保持原样，Nginx 负责 URL 重写。这样 Go 侧不需要处理顶级路由歧义。

**认证 API 路径调整：**

shell 脚本中的 auth URL 从 `/v1/entry/{shortId}/auth` 改为 `/v1/entry/{shortId}/auth`（不变），因为 shell 脚本内部直接调用完整路径。

### 置信度：HIGH

纯 Nginx 重写，零代码风险。

---

## 6. Bootstrap 实时状态：SSE

### 方案

**使用 Server-Sent Events (SSE)** 替代当前的轮询（`GET /v1/bootstrap/tasks/{taskID}`）。

SSE 优于 WebSocket 的理由：
- Bootstrap 是纯服务器到客户端的单向推送
- 在 shell 脚本中用 `curl` 即可消费 SSE（`curl -N`），无需 WebSocket 客户端库
- Go `net/http` 原生支持 SSE（Flusher 接口）
- 自动重连是协议内置的

### 具体变更

**新增 `internal/controlplane/http/bootstrap_sse.go`：**

```go
func (h *bootstrapSSEHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
    taskID := r.PathValue("taskID")

    w.Header().Set("Content-Type", "text/event-stream")
    w.Header().Set("Cache-Control", "no-cache")
    w.Header().Set("Connection", "keep-alive")
    w.Header().Set("X-Accel-Buffering", "no") // 告诉 Nginx 不要缓冲

    flusher, ok := w.(http.Flusher)
    if !ok {
        http.Error(w, "streaming not supported", http.StatusInternalServerError)
        return
    }

    ticker := time.NewTicker(1 * time.Second)
    defer ticker.Stop()

    for {
        select {
        case <-r.Context().Done():
            return
        case <-ticker.C:
            stage := resolveCurrentStage(r.Context(), taskID)
            fmt.Fprintf(w, "event: stage\ndata: %s\n\n", stageJSON)
            flusher.Flush()

            if stage.IsTerminal() {
                return
            }
        }
    }
}
```

**路由：**

```go
mux.Handle("GET /v1/bootstrap/tasks/{taskID}/stream", bootstrapSSEHandler)
```

**Nginx SSE 配置（已被上面 WebSocket 配置覆盖）：**

`proxy_buffering off` 和 `X-Accel-Buffering: no` 确保 Nginx 不缓冲 SSE。

**Shell 脚本消费 SSE：**

```bash
# 使用 curl -N（不缓冲）消费 SSE
curl -sN "$BASE_URL/v1/bootstrap/tasks/$TASK_ID/stream" | while IFS= read -r line; do
    case "$line" in
        data:*)
            json="${line#data: }"
            stage_text=$(echo "$json" | grep -o '"stage_text":"[^"]*"' | cut -d'"' -f4)
            printf "\r\033[K%s" "$stage_text"
            ;;
    esac
done
```

**保留现有轮询端点：** `GET /v1/bootstrap/tasks/{taskID}` 保持原样作为回退。SSE 端点是增量新增，不破坏任何现有功能。

### Nginx 对 SSE 的额外注意

在 `/v1/` location 中已经配置了 `proxy_buffering off`，这对 SSE 至关重要。如果 Nginx 缓冲了响应，SSE 事件将不会实时到达客户端。

### 置信度：HIGH

Go `net/http` + `http.Flusher` 是标准 SSE 实现方式，无需第三方库。

---

## 组件边界总览

### 新增组件

| 组件 | 文件 | 依赖 |
|------|------|------|
| `AppClaims` + 统一登录 | `user_auth.go` | `golang-jwt/jwt/v5`（已有） |
| `AuthMiddleware` + `RequireRole` | `auth_middleware.go` | 替换 `admin_auth.go` |
| 用户自助 API | `user_me.go` | Repository |
| Claude 账号管理 API | `admin_claude_accounts.go` | Repository |
| 用户 VNC 代理 | `user_vnc_proxy.go` | 复用 VNC 代理核心 |
| Bootstrap SSE | `bootstrap_sse.go` | Go 标准库 |
| DB Migration | `0006_claude_accounts.sql` | — |

### 修改组件

| 组件 | 变更 | 影响 |
|------|------|------|
| `router.go` | 新增路由组、中间件链 | 中等 |
| `nginx.conf` | WebSocket 升级 + 短 URL 重写 + SSE 不缓冲 | 低 |
| `lib/auth.ts` | 解析 role、存储 AuthState | 低 |
| `_dashboard.tsx` | 注入 auth context | 低 |
| 路由文件结构 | 拆分 `_admin/` 和 `_user/` 布局 | 中等 |
| `lib/api.ts` | 新增 `userApiFetch` | 低 |

### 不变组件

| 组件 | 原因 |
|------|------|
| Host Agent / Unix Socket | 不涉及用户面 |
| Network Provider (WireGuard/sing-box) | 不涉及 |
| SSH Proxy | 不涉及（继续用 short_id + entry_password） |
| Expiry Scanner / Reconciler | 不涉及 |
| 现有管理员 API 全部端点 | 仅中间件链调整，handler 逻辑不变 |

---

## 数据流变更

### 用户登录流

```
Browser -> POST /v1/auth/login { username, password }
        -> Go: 尝试 admin 凭证 -> 不匹配 -> 查 users 表 -> bcrypt 验证
        -> 签发 JWT { role: "user", uid: "<uuid>", sub: "<username>" }
        -> Browser 存储 token + role
        -> TanStack Router beforeLoad 按 role 路由
```

### 用户查看 VNC

```
Browser -> GET /v1/me/hosts/{hostId}/vnc/?token=<jwt>
        -> AuthMiddleware: 验证 JWT
        -> RequireRole("user"): 检查 role=user
        -> UserVNCProxyHandler: 检查 host 属于 uid
        -> WebSocket Upgrade / HTTP 反向代理 -> 容器 :6080
```

### Bootstrap SSE 流

```
Terminal -> curl domain/abc123 | bash
         -> Shell 脚本执行
         -> read -sp password
         -> POST /v1/entry/abc123/auth -> { task_id, ... }
         -> curl -sN /v1/bootstrap/tasks/{taskID}/stream
         -> SSE: event:stage, data:{stage_text:"主机启动中"}
         -> SSE: event:stage, data:{stage_text:"网络配置中"}
         -> SSE: event:stage, data:{stage_text:"SSH 就绪"}
         -> SSE: event:complete, data:{ssh_host,ssh_port,ssh_user}
         -> exec ssh ...
```

---

## 建议构建顺序

依赖分析决定的最优顺序：

### Phase 1: 认证基础设施

1. DB Migration `0006_claude_accounts.sql`
2. `AppClaims` 类型 + 统一登录端点 `POST /v1/auth/login`
3. `AuthMiddleware` + `RequireRole` 中间件
4. 将现有 `/v1/admin/*` 路由切换到新中间件链（行为等价，管理员照常工作）

**理由：** 所有后续功能都依赖角色认证。先做这一步，可以立即在现有管理后台验证新中间件不破坏任何东西。

### Phase 2: 用户自助 API + 前端路由

1. 用户自助 API（`/v1/me/hosts`、`/v1/me/claude-accounts`）
2. 前端 `lib/auth.ts` 改造 + 统一登录页
3. TanStack Router 路由拆分（`_admin/` + `_user/`）
4. 用户仪表盘页面

**理由：** 有了认证基础设施后，用户面 API 和路由可以平行推进。

### Phase 3: Claude 账号管理

1. 管理员 Claude 账号 CRUD API
2. 管理后台 Claude 账号管理页面
3. 用户面展示 Claude 账号信息

**理由：** 独立的 CRUD 功能，不阻塞其他功能。

### Phase 4: KasmVNC 用户面

1. VNC 代理核心抽取 + 用户 VNC handler
2. Nginx WebSocket 配置
3. 用户面 VNC 页面

**理由：** 依赖 Phase 2 的用户路由和权限。

### Phase 5: Bootstrap 重设计

1. Nginx 短 URL 重写
2. Bootstrap SSE 端点
3. 新 shell 脚本（欢迎艺术字 + 密码输入 + SSE 消费 + 自动 SSH）

**理由：** 相对独立，但排在最后是因为改动用户入口体验，需要前面的功能稳定后再动。

---

## Sources

- [Role-based Access Control in Golang with jwt-go](https://dev.to/bensonmacharia/role-based-access-control-in-golang-with-jwt-go-ijn)
- [golang-jwt/jwt/v5 官方文档](https://pkg.go.dev/github.com/golang-jwt/jwt/v5)
- [TanStack Router RBAC 指南](https://tanstack.com/router/v1/docs/framework/react/how-to/setup-rbac)
- [TanStack Router 认证路由](https://tanstack.com/router/v1/docs/framework/react/guide/authenticated-routes)
- [Proxying noVNC with nginx](https://github.com/novnc/noVNC/wiki/Proxying-with-nginx)
- [KasmVNC 反向代理文档](https://kasmweb.com/kasmvnc/docs/master/how_to/reverse_proxy.html)
- [Kasm Workspaces 反向代理配置](https://docs.kasm.com/docs/how-to/reverse_proxy/index.html)
- 项目现有代码：`admin_auth.go`、`admin_vnc_proxy.go`、`entry.go`、`router.go`、`bootstrap_status.go`
