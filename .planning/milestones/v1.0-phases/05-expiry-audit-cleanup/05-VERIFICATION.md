---
phase: 05
status: passed
verified: 2026-03-27
score: 4/4
---

# Phase 05 Verification

**Phase Goal:** 实现到期治理，并补齐日常运营所需的事件记录和运行状态清理能力。

## Success Criteria

| # | Criterion | Status | Evidence |
|---|-----------|--------|----------|
| 1 | 已过期用户无法再发起新的启动会话 | PASS | `bootstrap_auth.go:94-96` — `expired` 状态拦截返回 `account_expired` 错误；`scheduler/expiry.go:55` — `UpdateUserStatus(ctx, user.ID, "expired")` 将到期用户标记为 expired |
| 2 | 属于已过期用户的运行中主机会按照策略被处理 | PASS | `scheduler/expiry.go:80` — `QueueHostAction(ctx, host.ID, agentapi.ActionStopHost, "system:expiry")` 停止过期用户的 running 主机；`expiry.go:85-97` — 记录 `host.stop.expired` 事件 |
| 3 | 认证、启动、生命周期和到期事件都会被记录并可查看 | PASS | 事件注入：`bootstrap_auth.go` (`auth.success`/`auth.failed`)、`admin_users.go` (`admin.user.created`/`updated`/`deleted`/`password_rotated`)、`admin_hosts.go` (`admin.host.action`)、`admin_bindings.go` (`admin.binding.created`/`deleted`)、`expiry.go` (`user.expired`/`host.stop.expired`)、`reconciler.go` (`reconcile.host.drift`/`reconcile.task.stale`)；查询 API：`admin_events.go` + `router.go:199` (`GET /v1/admin/events`)；前端：`events/index.tsx` 事件日志页面 + `index.tsx` 仪表板最近事件卡片 |
| 4 | 陈旧任务和运行时漂移状态可以在不手工 SSH 排障的前提下完成对账 | PASS | `reconciler.go:55-103` — 漂移检测通过 `InspectContainer` 查询 Docker 状态，不一致时 `UpdateHostStatus` 修正并记录 `reconcile.host.drift` 事件；`reconciler.go:105-133` — `MarkStaleTasks` 标记超时任务为 failed 并记录 `reconcile.task.stale` 事件；`app.go:115-117` — reconcile job 在 scheduler 中注册 |

## Requirements Coverage

| Requirement | Description | Status | Evidence |
|-------------|-------------|--------|----------|
| LIFE-04 | 管理员可以为用户账号设置到期时间 | PASS | 后端：`admin_users.go` `UpdateExpiry()` handler + `router.go:170` `PUT /v1/admin/users/{userID}/expiry`；Repository：`queries.go:854` `UpdateUserExpiry`；前端：`$userId.tsx` 到期时间 Dialog（设置/修改/清除） |
| LIFE-05 | 已过期用户不能再开启新会话，后台明确显示已过期状态 | PASS | 拦截：`bootstrap_auth.go:94-96` expired 状态返回 `account_expired`；定时器：`scheduler/expiry.go:33-51` 每 60s 扫描并标记到期用户；前端：`users/index.tsx` expired Badge (destructive variant, "已过期")；`$userId.tsx` 到期时间信息行 + expired 状态 Badge |
| ADMN-04 | 管理员操作和启动结果会被记录为运维事件 | PASS | 全部 D-08 定义的 13 种事件类型均有 RecordEvent 调用点：auth (2), user (4), binding (2), host (1), expiry (2), reconcile (2)；`GET /v1/admin/events` 支持 type/user_id/host_id/since/until 筛选和 limit/offset 分页；事件日志页面 + 仪表板摘要卡片提供可视化查看 |

## Must-Haves Check

### Plan 05-01 Must-Haves

| Truth | Status | Evidence |
|-------|--------|----------|
| users 表包含 expires_at TIMESTAMPTZ 列且可为 NULL | PASS | `0003_expiry_audit.sql:2` — `ALTER TABLE users ADD COLUMN expires_at TIMESTAMPTZ` |
| events 表包含 user_id UUID 列并有索引 | PASS | `0003_expiry_audit.sql:5` — `ALTER TABLE events ADD COLUMN user_id UUID`；`0003_expiry_audit.sql:10` — `idx_events_user_id_created_at` |
| 到期定时器每 60 秒扫描过期用户并标记为 expired | PASS | `app.go:111-113` — 默认 60s 间隔；`expiry.go:34` — `ListExpiredActiveUsers`；`expiry.go:55` — `UpdateUserStatus expired` |
| 已过期用户的 running 主机通过 QueueHostAction 被自动停止 | PASS | `expiry.go:74-82` — `ListRunningHostsByUserID` + `QueueHostAction(ActionStopHost)` |
| 管理 API 支持设置、修改和清除用户到期时间 | PASS | `admin_users.go` `UpdateExpiry()`；`router.go:170` — `PUT /v1/admin/users/{userID}/expiry` |
| 管理 API 允许将 expired 用户重新激活为 active | PASS | `admin_users.go` `UpdateStatus()` — 目标状态 active/disabled 允许 expired→active |

| Artifact | Status | Evidence |
|----------|--------|----------|
| `internal/store/migrations/0003_expiry_audit.sql` | VERIFIED | 存在，包含 expires_at、user_id、索引 |
| `internal/controlplane/scheduler/scheduler.go` | VERIFIED | 导出 Scheduler/Job/New，Run 使用 WaitGroup |
| `internal/controlplane/scheduler/expiry.go` | VERIFIED | 导出 ExpiryScanner/NewExpiryScanner，Scan 方法完整 |

| Key Link | Status | Evidence |
|----------|--------|----------|
| app.go → scheduler.go via `sched.Run` | WIRED | `app.go:123` — `sched.Run(schedCtx)` |
| expiry.go → queries.go via `ListExpiredActiveUsers` | WIRED | `expiry.go:34` — `s.store.ListExpiredActiveUsers(ctx)` |
| admin_users.go → queries.go via `UpdateUserExpiry` | WIRED | `admin_users.go` `UpdateExpiry()` 调用 `h.store.UpdateUserExpiry` |

### Plan 05-02 Must-Haves

| Truth | Status | Evidence |
|-------|--------|----------|
| 管理员创建/修改/删除用户时都会生成事件记录 | PASS | `admin_users.go:124-131` (created), `admin_users.go:170-177` (updated/status), `admin_users.go:259-266` (deleted) |
| 管理员轮换密码时会生成 admin.user.password_rotated 事件 | PASS | `admin_users.go:309-316` — Type: `admin.user.password_rotated` |
| 管理员创建/删除绑定时会生成事件记录 | PASS | `admin_bindings.go:74-81` (created), `admin_bindings.go:127-134` (deleted) |
| 管理员发起主机生命周期操作时会生成 admin.host.action 事件 | PASS | `admin_hosts.go:98-105` — Type: `admin.host.action` |
| Bootstrap 认证成功/失败时会生成事件 | PASS | `bootstrap_auth.go:124-131` (auth.success), `bootstrap_auth.go:147-154` (auth.failed) |
| GET /v1/admin/events 返回支持筛选的分页事件列表 | PASS | `admin_events.go` List() — 支持 type/user_id/host_id/since/until/limit/offset；`router.go:199` 注册路由 |
| 对账定时器检测 DB 与 Docker 状态漂移 | PASS | `reconciler.go:55-103` — ListRunningHosts + InspectContainer；通信失败只记日志不修改 DB（Pitfall 4 防护） |
| 对账定时器标记陈旧任务为 failed | PASS | `reconciler.go:105-133` — MarkStaleTasks + reconcile.task.stale 事件记录 |

| Artifact | Status | Evidence |
|----------|--------|----------|
| `internal/controlplane/http/admin_events.go` | VERIFIED | 导出 AdminEventsHandler/NewAdminEventsHandler |
| `internal/controlplane/scheduler/reconciler.go` | VERIFIED | 导出 Reconciler/NewReconciler |

| Key Link | Status | Evidence |
|----------|--------|----------|
| admin_users.go → queries.go via RecordEvent | WIRED | 5 处 RecordEvent 调用 |
| admin_events.go → queries.go via ListEvents | WIRED | `admin_events.go:72` — `h.store.ListEvents` |
| reconciler.go → agentapi/client.go via InspectContainer | WIRED | `reconciler.go:64` — `r.inspector.InspectContainer` |

### Plan 05-03 Must-Haves

| Truth | Status | Evidence |
|-------|--------|----------|
| 用户列表页展示到期时间列和 expired 状态 Badge | PASS | `users/index.tsx` — 到期时间列 + destructive Badge "已过期" |
| 用户详情页展示到期时间并提供设置/修改/清除操作 | PASS | `$userId.tsx` — 到期时间信息行 + Dialog (设置/修改/清除) |
| 用户详情页允许将已过期用户重新激活为 active | PASS | `$userId.tsx` — expired → active 重新激活 |
| 侧栏包含事件日志导航入口 | PASS | `sidebar.tsx:20` — `{ label: "事件日志", to: "/events", icon: ScrollText }` |
| 事件日志页面展示全局事件时间线并支持筛选 | PASS | `events/index.tsx` — createFileRoute("/_dashboard/events/")，含类型筛选和分页 |
| 仪表板概览包含最近事件摘要卡片 | PASS | `_dashboard/index.tsx` — `useEvents({ limit: 5 })` + "最近事件" CardTitle + "查看全部" Link |

| Artifact | Status | Evidence |
|----------|--------|----------|
| `web/admin/src/hooks/use-events.ts` | VERIFIED | 导出 useEvents/eventTypeLabel/EventItem |
| `web/admin/src/routes/_dashboard/events/index.tsx` | VERIFIED | 事件日志页面完整实现 |
| `web/admin/src/components/layout/sidebar.tsx` | VERIFIED | 包含事件日志入口 |

| Key Link | Status | Evidence |
|----------|--------|----------|
| use-events.ts → GET /v1/admin/events via apiFetch | WIRED | `use-events.ts:51+` — `apiFetch<EventsResponse>("/events...")` |
| events/index.tsx → use-events.ts via useEvents | WIRED | events/index.tsx 导入并使用 useEvents hook |
| $userId.tsx → PUT /users/{id}/expiry via useUpdateUserExpiry | WIRED | `$userId.tsx:5,50` — 导入并使用 useUpdateUserExpiry |

## Anti-Pattern Scan

| Pattern | Files Scanned | Results |
|---------|---------------|---------|
| TODO/FIXME/XXX/HACK | scheduler/*, admin_events.go, use-events.ts, events/index.tsx | None found |
| Placeholder content | scheduler/* | None found |
| Empty returns | — | N/A (all functions have real logic) |
| Log-only functions | — | N/A |

## Human Verification

| Test | What to Do | Expected Result | Why Manual |
|------|------------|-----------------|------------|
| 到期流程端到端 | 创建用户 → 设置到期时间为当前时间后 2 分钟 → 等待定时器触发 → 尝试 bootstrap 认证 | 用户被标记为 expired，认证返回 `account_expired`，running 主机被停止 | 需要真实 DB + 定时器运行 + Docker 环境 |
| 事件日志 UI 交互 | 管理后台 → 事件日志 → 按类型筛选 → 展开行查看 metadata → 分页切换 | 筛选正确，metadata 展示正确，分页正常 | 需要浏览器交互验证 |
| 仪表板事件卡片 | 管理后台 → 仪表板 → 查看最近事件卡片 → 点击"查看全部" | 显示最近 5 条事件，点击跳转到事件日志页面 | 视觉验证和导航交互 |
| 对账漂移检测 | 手动停止一个 running 状态的容器（docker stop） → 等待对账定时器运行 | DB 中主机状态更新为 stopped，生成 reconcile.host.drift 事件 | 需要 Docker 环境 + 定时器运行 |
| 过期用户重新激活 | 管理后台 → 找到 expired 用户 → 点击重新激活 → 重新设置到期时间 | 状态恢复为 active，可再次启动会话 | UI 交互验证 |

## Gaps

无。所有 4 项成功标准均通过自动化验证，所有 3 个需求 ID 均已覆盖，所有 PLAN must-haves 均已满足。

---
*Verified: 2026-03-27*
*Phase: 05-expiry-audit-cleanup*
