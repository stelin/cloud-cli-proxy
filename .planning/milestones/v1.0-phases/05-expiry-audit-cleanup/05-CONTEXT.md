# Phase 5: 到期、审计与清理 - Context

**Gathered:** 2026-03-27
**Status:** Ready for planning

<domain>
## Phase Boundary

本阶段交付到期治理、运维事件记录和运行状态清理能力。具体包括：为用户账号引入到期时间与自动执行策略，让已过期用户无法发起新会话并按策略处理其运行中主机；补齐认证、启动、生命周期和到期事件的结构化记录与后台可查展示；增加陈旧任务清理和 DB/Docker 运行时漂移对账能力。计费、自助续期、多宿主机调度和 Web Terminal 不在本阶段范围。

</domain>

<decisions>
## Implementation Decisions

### 到期模型与策略
- **D-01:** 在 `users` 表新增 `expires_at TIMESTAMPTZ NULL` 字段；NULL 表示永不过期，非 NULL 值表示账号到期时间点。
- **D-02:** 控制面启动后台定时 goroutine（如每 60 秒一次），扫描 `expires_at <= now() AND status = 'active'` 的用户，将其 `status` 设为 `expired`。该定时器与控制面主进程同生命周期。
- **D-03:** 管理 API 和后台 UI 支持设置、修改和清除用户到期时间（满足 LIFE-04）。管理员也可以将已过期用户重新激活（设置新到期时间并恢复 `active` 状态）。
- **D-04:** Bootstrap 认证入口已在 Phase 3 实现 `expired` 状态拦截（`bootstrap_auth.go`），本阶段无需修改认证拦截逻辑，只需确保到期定时器正确触发状态变更。

### 过期主机处理策略
- **D-05:** 到期定时器在将用户标记为 `expired` 的同时，检查该用户是否有 `status = 'running'` 的主机，如有则通过现有 `QueueHostAction(ActionStopHost)` 下发停止动作。
- **D-06:** v1 不提供宽限期（grace period），到期即执行。这与项目"宁可失败，不可打穿"的原则一致。
- **D-07:** 过期主机停止事件使用专用事件类型（`host.stop.expired`），与管理员手动停止区分开，便于审计追溯。

### 事件记录范围与分类
- **D-08:** 新增以下事件类型约定，覆盖 ADMN-04 要求的认证、启动、生命周期和到期事件：
  - `auth.success` — 用户认证成功
  - `auth.failed` — 用户认证失败（含原因：凭证错误/账号禁用/账号过期）
  - `user.expired` — 用户被到期定时器标记为过期
  - `host.stop.expired` — 过期用户的主机被自动停止
  - `admin.user.created` — 管理员创建用户
  - `admin.user.updated` — 管理员修改用户（含状态变更、到期时间修改）
  - `admin.user.deleted` — 管理员删除用户
  - `admin.user.password_rotated` — 管理员轮换用户密码
  - `admin.binding.created` — 管理员创建出口 IP 绑定
  - `admin.binding.deleted` — 管理员删除出口 IP 绑定
  - `admin.host.action` — 管理员发起主机生命周期操作（start/stop/rebuild）
  - `reconcile.host.drift` — 对账发现主机状态漂移
  - `reconcile.task.stale` — 对账发现陈旧任务
- **D-09:** 沿用现有 `RecordEvent` + JSONB `metadata` 模式。事件统一携带 `user_id`（如适用）和 `operator`（`system` / `admin` / `bootstrap`），按需附加 `host_id`、`reason`、`action` 等上下文字段。
- **D-10:** 现有 worker 中已有的 `net.ready`、`ssh.ready` 等事件保持不变，新增事件类型不影响既有记录行为。

### 后台审计展示
- **D-11:** 侧栏新增「事件日志」入口，展示全局事件时间线列表。独立于现有「任务列表」页面。
- **D-12:** 事件列表支持按事件类型、用户、主机和时间范围筛选。默认按时间倒序展示。
- **D-13:** 事件详情展示 metadata 中的上下文字段，便于排障和支持。
- **D-14:** 后台仪表板概览中增加"最近事件"摘要卡片，展示最近 N 条关键事件。

### 对账与漂移检测
- **D-15:** 控制面定时扫描 DB 中 `hosts.status = 'running'` 的主机列表，通过 host-agent 查询 Docker 实际容器状态，发现不一致时记录 `reconcile.host.drift` 事件并修正 DB 状态（如容器已不存在则标记主机为 `stopped`）。
- **D-16:** 超过可配置阈值（默认 10 分钟）仍处于 `pending` 或 `running` 状态的任务，由对账定时器标记为 `failed` 并记录 `reconcile.task.stale` 事件，携带原始任务信息作为 metadata。
- **D-17:** 对账定时器与到期定时器共享同一后台调度框架（控制面启动时统一注册），但执行间隔独立可配。
- **D-18:** 对账操作通过 host-agent 查询 Docker 状态，不在控制面直接持有 Docker 特权（遵循 Phase 1 D-01 特权边界）。

### Claude's Discretion
- 定时器的具体间隔参数和配置方式（环境变量 / 配置文件）。
- 事件列表页的分页策略和批量加载实现细节。
- 对账扫描的并发策略和批量大小。
- 事件 metadata 中各类型的具体字段命名约定。
- 仪表板"最近事件"卡片的展示条数和刷新策略。

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### 阶段范围与验收标准
- `.planning/ROADMAP.md` — Phase 5 的目标、成功标准与 05-01/05-02/05-03 计划拆分。
- `.planning/REQUIREMENTS.md` — `LIFE-04`、`LIFE-05`、`ADMN-04` 的正式定义。
- `.planning/PROJECT.md` — 产品边界、约束与核心价值。

### 前置阶段锁定决策
- `.planning/phases/01-foundation-control-plane-host-agent/01-CONTEXT.md` — 异步任务模型（D-13~15）、失败策略、重建双模式。
- `.planning/phases/02-tunnel-egress-enforcement/02-CONTEXT.md` — 事件按类型细分记录（D-12~13）、异常时主机标记失败（D-10）。
- `.planning/phases/03-ssh/03-CONTEXT.md` — Bootstrap 认证含 `expired` 状态拦截（D-03）、失败不自动重试（D-11）。
- `.planning/phases/04-admin-ui/04-CONTEXT.md` — 禁用用户不影响已运行主机（D-10）、管理 API 与 JWT 认证模式。

### 现有代码锚点
- `internal/controlplane/http/bootstrap_auth.go` — 已实现 `expired` 状态拦截和 `account_expired` 错误返回。
- `internal/controlplane/http/bootstrap_errors.go` — `account_expired` 等启动错误码与终端文案定义。
- `internal/controlplane/http/admin_users.go` — 用户管理 API，当前只允许 `active`/`disabled` 状态切换。
- `internal/controlplane/http/router.go` — 现有路由结构，需扩展事件查询 API。
- `internal/store/repository/queries.go` — `RecordEvent`、`ListEventsByTaskID`、`ListPendingTasks`（未使用）、`UpdateUserStatus`。
- `internal/store/repository/models.go` — `User`、`Host`、`Task`、`Event` 模型定义。
- `internal/store/migrations/0001_initial.sql` — `users`、`events`、`tasks` 表结构。
- `internal/runtime/runtime_service.go` — `QueueHostAction` 入口，可复用于到期主机停止。
- `internal/runtime/tasks/worker.go` — 任务执行与事件记录模式。
- `internal/controlplane/app/app.go` — 控制面启动入口，需增加后台定时器注册。

### 前端锚点
- `web/admin/src/lib/api.ts` — 前端 API 调用基础。
- `web/admin/src/hooks/use-tasks.ts` — TanStack Query + 轮询模式，可作为事件列表 hook 参考。
- `web/admin/src/routes/_dashboard/tasks/index.tsx` — 任务列表页实现，可作为事件列表页参考。

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- `RecordEvent` + `Event` 模型：事件记录基础设施已完备，新增事件类型只需统一 `type` 约定即可。
- `ListPendingTasks`：已定义但全仓库无调用方，可直接用于陈旧任务扫描。
- `QueueHostAction` + `ActionStopHost`：到期主机停止可直接复用现有主机停止链路。
- `bootstrap_auth.go` 中 `expired` 状态拦截：认证层已就绪，只需确保到期定时器正确设置用户状态。
- 前端 TanStack Query + `apiFetch` 模式：事件列表页可沿用用户/任务列表的数据获取和轮询模式。

### Established Patterns
- 控制面使用 `net/http` + JSON 响应，统一错误格式，新增 API 继续沿用。
- 生命周期操作走异步任务状态机（`pending/running/succeeded/failed/canceled`），对账清理遵循同一模型。
- 高权限操作封装在 host-agent 中，对账查询 Docker 状态也走 host-agent 通道。
- 事件类型为自由文本 `type`，metadata 为 JSONB，已建立灵活的扩展模式。

### Integration Points
- `internal/controlplane/app/app.go`：需增加后台定时器（到期扫描、对账扫描）的注册与生命周期管理。
- `internal/store/migrations/`：需新增迁移文件，添加 `users.expires_at` 字段和 `events` 表按 `host_id`/`created_at` 的索引。
- `internal/controlplane/http/router.go`：需注册事件查询 API（`GET /v1/admin/events`）。
- `internal/controlplane/http/admin_users.go`：需扩展用户更新 API 支持 `expires_at` 字段设置。
- 前端侧栏与路由：需新增事件日志页面和仪表板事件摘要卡片。

</code_context>

<specifics>
## Specific Ideas

- 到期执行遵循"宁可失败，不可打穿"原则：过期用户的主机必须被停止，不能留下可继续使用的运行态主机。
- 事件日志应覆盖"运维排障"场景：管理员看到一个失败，应该能从事件时间线中快速定位原因。
- 对账的核心价值是"不需要 SSH 到宿主机上排查"：DB 状态与 Docker 实际状态的偏差应在后台可见。

</specifics>

<deferred>
## Deferred Ideas

None — discussion stayed within phase scope.

</deferred>

---

*Phase: 05-expiry-audit-cleanup*
*Context gathered: 2026-03-27*
