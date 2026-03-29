# Phase 4: 后台管理界面 - Context

**Gathered:** 2026-03-27
**Status:** Ready for planning

<domain>
## Phase Boundary

本阶段交付后台管理界面的完整能力：管理员认证登录、用户 CRUD 与凭证轮换、出口 IP 资源 CRUD 与绑定管理、主机生命周期操作（启动/停止/重建）以及状态看板。前端使用 React + Vite 从零搭建，后端补齐缺失的管理 API。到期治理、审计事件展示和计费流程不在本阶段范围。

</domain>

<decisions>
## Implementation Decisions

### 后台认证与权限模型
- **D-01:** v1 采用单管理员账号 + JWT 认证。管理员凭证通过环境变量或首次配置设定，登录后签发 JWT，前端所有 API 请求携带 Bearer Token。
- **D-02:** v1 不做多角色 RBAC，后台所有功能对已认证管理员全部开放。
- **D-03:** JWT 应设置合理过期时间，过期后前端引导重新登录。

### 界面布局与状态看板
- **D-04:** 采用侧边栏导航 + 顶栏布局。侧边栏分为：仪表板概览、用户管理、出口 IP 管理、主机管理、任务列表。
- **D-05:** 仪表板概览展示关键计数（活跃用户数、运行中主机数、可用出口 IP 数）和最近任务状态摘要。
- **D-06:** 前端使用 React 19.2 + Vite 8.x 搭建，与后端 Go API 分离部署（开发阶段 Vite dev server 代理到后端）。

### 用户与凭证管理
- **D-07:** 创建用户时管理员手动设定初始密码，后端使用 bcrypt 哈希存储（沿用 Phase 3 决策）。
- **D-08:** 删除用户时弹窗确认 + 要求输入用户名二次确认，防止误操作。删除会级联清理关联主机与绑定。
- **D-09:** 密码轮换：管理员点击按钮系统自动生成随机强密码，界面展示一次供复制，不支持找回。
- **D-10:** 禁用用户后，该用户无法通过 bootstrap 流程认证，但不影响已运行主机的即时状态（已运行主机需手动停止）。

### 出口 IP 资源与绑定操作
- **D-11:** 出口 IP 的创建和编辑使用右侧抽屉表单，不离开列表页面。字段包括标签、IP 地址、提供商、WireGuard 配置参数等。
- **D-12:** 绑定操作在用户详情页完成：管理员在用户/主机视图中选择可用出口 IP 进行绑定。
- **D-13:** 运行中主机不允许直接换绑出口 IP，必须先停机再操作（符合 Phase 2 D-03 "禁止运行时隐式换 IP"）。
- **D-14:** 解绑出口 IP 时如果主机正在运行，界面应给出警告并阻止操作。

### 主机生命周期操作
- **D-15:** 启动/停止/重建操作均通过现有异步任务链路（Phase 1 D-13），后台界面提交后显示任务状态。
- **D-16:** 重建操作时提供模式选择：默认"保留主目录并重置系统层"，可选"工厂重置"（Phase 1 D-11）。
- **D-17:** 主机当前状态（pending/running/stopped/failed）在列表和详情页实时可见。

### Claude's Discretion
- 具体的 UI 组件库选择（可用 Tailwind CSS + headless 组件，也可用其他轻量方案）。
- 前端路由组织方式和状态管理策略。
- JWT 签名算法、过期时间和刷新机制的具体实现细节。
- 仪表板概览的数据刷新策略（轮询间隔、手动刷新按钮等）。
- 表格分页、排序、筛选的具体交互细节。

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### 阶段范围与验收标准
- `.planning/ROADMAP.md` — Phase 4 的目标、成功标准与 04-01/04-02/04-03 计划拆分。
- `.planning/REQUIREMENTS.md` — `USER-01`、`USER-02`、`LIFE-01`~`LIFE-03`、`ADMN-01`~`ADMN-03` 的正式定义。
- `.planning/PROJECT.md` — 产品边界、约束与核心价值。

### 前置阶段锁定决策
- `.planning/phases/01-foundation-control-plane-host-agent/01-CONTEXT.md` — 控制面/host-agent 边界、任务模型、重建双模式、失败策略。
- `.planning/phases/02-tunnel-egress-enforcement/02-CONTEXT.md` — 出口 IP 绑定语义、禁止运行时换 IP、校验策略。
- `.planning/phases/03-ssh/03-CONTEXT.md` — bcrypt 密码、bootstrap 认证流程、错误分类。

### 技术栈
- `CLAUDE.md` — 推荐技术栈：React 19.2、Vite 8.x、Node.js 24 LTS，以及 Go 后端技术选型。

### 现有 API 与数据层
- `internal/controlplane/http/router.go` — 现有 API 路由结构和依赖注入模式。
- `internal/controlplane/http/hosts.go` — 主机生命周期 action handler 实现模式。
- `internal/store/repository/models.go` — 数据模型定义：User、Host、EgressIP、HostBinding、Task、Event。
- `internal/store/repository/queries.go` — 现有查询能力与 repository 模式。
- `internal/store/migrations/0001_initial.sql` — 核心数据库 schema。
- `internal/store/migrations/0002_egress_tunnel.sql` — WireGuard 隧道字段扩展。

### 架构
- `.planning/codebase/ARCHITECTURE.md` — 控制面/host-agent/用户容器的职责分段。

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- `internal/controlplane/http/router.go`：已有路由注册模式和 `writeJSON` 工具函数，新增管理 API 可直接沿用。
- `internal/controlplane/http/hosts.go`：`HostActionsHandler` 已封装 create/start/stop/rebuild 操作，后台界面可直接调用。
- `internal/store/repository/queries.go`：已有 `ListUsers`、`ListHosts`、`ListHostBindings`、`GetEgressIP`、`UpsertHost`、`CreateTask` 等查询，后台管理 API 可复用大量读写逻辑。
- `internal/store/repository/models.go`：`User`、`Host`、`EgressIP`、`HostBinding`、`Task`、`Event` 模型已完整定义。

### Established Patterns
- Go 控制面使用 `net/http` + JSON 响应，统一错误格式。
- 生命周期操作走异步任务状态机，结果通过任务状态和事件记录暴露。
- 高权限操作封装在 host-agent 中，控制面只负责入队和查询。
- 前端项目从零搭建，没有既有前端代码可继承。

### Integration Points
- 后端需补齐管理 API：用户 CRUD、密码轮换、出口 IP CRUD、绑定管理、管理员认证。
- 前端 Vite dev server 可代理 `/v1/*` 请求到 Go 后端（开发环境）。
- 部署时后端可同时 serve 前端静态文件，也可独立 Nginx 代理。

</code_context>

<specifics>
## Specific Ideas

- 删除用户时要求输入用户名进行二次确认，是为了防止管理员误操作导致数据丢失。
- 密码轮换采用"自动生成 + 展示一次"模式，简单且安全，适合管理员直接告知用户新密码的场景。
- 运行中主机不能换绑出口 IP 的限制，直接继承自 Phase 2 的核心安全约束（出口 IP 变化会触发 claude 账号风控）。

</specifics>

<deferred>
## Deferred Ideas

None — discussion stayed within phase scope.

</deferred>

---

*Phase: 04-admin-ui*
*Context gathered: 2026-03-27*
