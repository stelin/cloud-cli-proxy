# Phase 12: 用户自助 API 与前端路由 - Context

**Gathered:** 2026-03-29 (auto mode)
**Status:** Ready for planning

<domain>
## Phase Boundary

用户可以在自助面板中查看自己的主机状态、出口 IP 并执行主机重建。覆盖 PANEL-01、PANEL-02、PANEL-03。用户面板与管理员面板共存于同一 React 应用，通过角色路由守卫隔离。

</domain>

<decisions>
## Implementation Decisions

### API 路由设计
- **D-01:** 用户 API 使用 `/v1/user/` 前缀，与管理员 `/v1/admin/` 对称
- **D-02:** 用户路由守卫 `userGuard = authMw(RequireRole("user", "admin"))`，管理员也能访问用户端点
- **D-03:** 每个用户端点内部从 `UserIDFromContext(r.Context())` 获取 user_id，查询时自动过滤只返回属于该用户的资源
- **D-04:** 新增端点：`GET /v1/user/hosts`（我的主机列表）、`GET /v1/user/hosts/{hostID}`（主机详情含出口 IP）、`POST /v1/user/hosts/{hostID}/rebuild`（触发重建）

### 面板布局与信息展示
- **D-05:** 用户主页 `/portal` 展示主机列表，每台主机以卡片形式显示：主机名（hostname）、运行状态（status badge）、出口 IP 地址和隧道类型、创建时间
- **D-06:** 复用现有 `_portal.tsx` 布局（Topbar + 主内容区），不需要侧边栏（用户功能少，Topbar 导航即可）
- **D-07:** 前端路由结构：`/portal`（主机列表首页）、`/portal/hosts/$hostId`（主机详情页）

### 主机重建交互
- **D-08:** 重建按钮点击后弹出确认对话框，明确警告"重建将重置容器环境，home 目录数据保留"
- **D-09:** 确认后调用 `POST /v1/user/hosts/{hostID}/rebuild`，后端通过 `QueueHostAction` 排队任务
- **D-10:** 重建触发后前端显示任务进度状态（可通过轮询或直接更新主机状态实现），重建期间主机状态变为 rebuilding

### 出口 IP 展示
- **D-11:** 出口 IP 信息在主机详情页展示，只读模式，显示 IP 地址和隧道类型（wireguard/proxy）
- **D-12:** 在主机列表卡片中也简要展示出口 IP 地址（不展示隧道类型细节）

### 资源归属校验
- **D-13:** 用户请求的 hostID 必须通过 user_id 归属校验，不匹配返回 403
- **D-14:** 管理员角色访问用户端点时跳过归属校验（管理员可查看任意用户资源）

### Claude's Discretion
- 主机卡片的具体样式（阴影、圆角、间距）
- 状态 badge 的颜色映射（running=green, stopped=gray, rebuilding=yellow 等）
- 空状态展示（无主机时的提示文案和样式）
- 主机详情页的具体布局
- 重建进度的轮询间隔

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### 需求定义
- `.planning/REQUIREMENTS.md` — PANEL-01, PANEL-02, PANEL-03 验收标准
- `.planning/ROADMAP.md` §Phase 12 — Success Criteria 4 条

### 认证基础设施（Phase 11 产出）
- `internal/controlplane/http/auth_middleware.go` — AuthMiddleware、RequireRole、UserIDFromContext、RoleFromContext
- `internal/controlplane/http/auth_handler.go` — UnifiedLoginHandler、AuthClaims（含 user_id + role）
- `web/admin/src/lib/auth.ts` — 前端 JWT 解析、getRole()、isAdmin()、parseTokenPayload()

### 现有管理员 API（可参考的实现模式）
- `internal/controlplane/http/admin_hosts.go` — 管理员主机 CRUD handler 实现模式
- `internal/controlplane/http/router.go` — 路由注册、中间件链和 adminGuard 模式

### 数据层
- `internal/store/repository/queries.go` — ListHostsByUserID、GetEgressIPByHost、QueueHostAction 等现有查询
- `internal/store/repository/models.go` — Host、EgressIP 结构体定义

### 前端路由与布局
- `web/admin/src/routes/_portal.tsx` — 用户面板布局（Topbar + main）
- `web/admin/src/routes/_portal/index.tsx` — 当前骨架页面（需替换为主机列表）
- `web/admin/src/routes/_dashboard.tsx` — 管理员布局（含角色路由守卫参考）

### 先前阶段决策
- `.planning/phases/11-auth-infra-migration/11-CONTEXT.md` — D-11~D-14 统一登录与角色路由决策

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- `AuthMiddleware` + `RequireRole` + `UserIDFromContext`：认证和归属校验的完整基础设施
- `ListHostsByUserID(ctx, userID)`：按 user_id 查主机列表的现成查询
- `GetEgressIPByHost(ctx, hostID)`：按 host_id 查绑定出口 IP 的现成查询
- `QueueHostAction(ctx, hostID, action, userID)`：排队主机操作的现成接口
- `AdminHostsHandler`：管理员主机 handler 可作为用户 handler 的实现参考
- `_portal.tsx` 布局组件：前端用户面板框架已就位
- TanStack Router 文件路由 + React Query 数据获取模式已建立

### Established Patterns
- Go handler 模式：`type XxxHandler struct` + 依赖注入 + `func (h *XxxHandler) List() nethttp.Handler`
- 路由守卫链：`authMw(RequireRole(...))(handler)`
- 前端数据获取：React Query + `getToken()` 构造 Authorization header
- 数据库查询：pgx 手写 SQL + `Repository` 方法

### Integration Points
- `router.go` 中注册 `/v1/user/` 端点，与 `/v1/admin/` 平级
- `Dependencies` struct 可能需要新增 `UserHosts`、`UserEgressIPs` 等接口（或复用已有的）
- 前端在 `_portal/` 目录下新增路由文件

</code_context>

<specifics>
## Specific Ideas

No specific requirements — open to standard approaches

</specifics>

<deferred>
## Deferred Ideas

- Claude 账号展示 — Phase 13
- KasmVNC 远程桌面访问入口 — Phase 14
- 用户面板的更多个性化设置 — 未来考虑

</deferred>

---

*Phase: 12-api*
*Context gathered: 2026-03-29*
