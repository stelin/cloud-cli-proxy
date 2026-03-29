---
phase: 05-expiry-audit-cleanup
plan: 02
subsystem: api, scheduler
tags: [events, audit, reconciliation, docker, agentapi]

requires:
  - phase: 05-01
    provides: "events table, RecordEvent, ListEvents, scheduler framework, expiry scanner"
provides:
  - "EventRecorder interface for all admin handlers"
  - "Admin event recording on user/host/binding CRUD and auth"
  - "GET /v1/admin/events API with multi-filter pagination"
  - "Container inspect endpoint on host-agent"
  - "Runtime reconciler for host drift and stale task detection"
affects: [06-frontend-admin]

tech-stack:
  added: []
  patterns:
    - "fire-and-forget event recording with nil-guard and error logging"
    - "ReconcileStore/ContainerInspector interface separation for reconciler"
    - "communication failure vs container absence distinction in reconciliation"

key-files:
  created:
    - internal/controlplane/http/admin_events.go
    - internal/controlplane/scheduler/reconciler.go
  modified:
    - internal/controlplane/http/admin_users.go
    - internal/controlplane/http/admin_hosts.go
    - internal/controlplane/http/admin_bindings.go
    - internal/controlplane/http/bootstrap_auth.go
    - internal/controlplane/http/router.go
    - internal/controlplane/app/app.go
    - internal/agentapi/contracts.go
    - internal/agentapi/client.go
    - internal/agent/server.go

key-decisions:
  - "EventRecorder interface defined in router.go, all handlers receive it via constructor injection"
  - "Event recording is fire-and-forget: failure only logs, never blocks the main operation response"
  - "Reconciler strictly distinguishes communication failure (skip) from container absence (update DB)"

patterns-established:
  - "nil-guard event recording: if h.events != nil { h.events.RecordEvent(...) }"
  - "Reconciler Pitfall 4 safeguard: InspectContainer error → log + continue, never modify DB"

requirements-completed: [ADMN-04]

duration: 12min
completed: 2026-03-27
---

# Phase 05 Plan 02: 管理操作事件审计与运行时对账 Summary

**为所有管理 handler 注入事件记录、新增事件查询 API、实现 host-agent 容器 inspect 端点和 DB/Docker 运行时对账定时器**

## Performance

- **Duration:** 12 min
- **Started:** 2026-03-27T10:00:00Z
- **Completed:** 2026-03-27T10:12:00Z
- **Tasks:** 2
- **Files modified:** 11

## Accomplishments
- 所有管理员操作（用户 CRUD、密码轮换、到期时间、绑定管理、主机生命周期）成功后都会生成审计事件
- Bootstrap 认证的成功和失败（含原因区分）都会记录事件
- 新增 `GET /v1/admin/events` API，支持 type/user_id/host_id/since/until 筛选和 limit/offset 分页
- host-agent 新增 `GET /v1/containers/{name}/status` 端点，通过 docker inspect 查询容器状态
- 运行时对账定时器检测 DB/Docker 状态漂移，自动修正漂移主机并标记陈旧任务

## Task Commits

Each task was committed atomically:

1. **Task 1: 管理操作事件记录注入与事件查询 API** - `4cfcf6e` (feat)
2. **Task 2: 运行时对账与 host-agent inspect 端点** - `370f3ad` (feat)

## Files Created/Modified
- `internal/controlplane/http/router.go` - EventRecorder/AdminEventStore 接口定义和 Dependencies 扩展
- `internal/controlplane/http/admin_users.go` - 用户 CRUD 和密码轮换事件注入
- `internal/controlplane/http/admin_hosts.go` - 主机生命周期操作事件注入
- `internal/controlplane/http/admin_bindings.go` - 绑定管理事件注入
- `internal/controlplane/http/bootstrap_auth.go` - 认证成功/失败事件注入
- `internal/controlplane/http/admin_events.go` - 事件查询 API handler
- `internal/agentapi/contracts.go` - ContainerStatusResponse 类型
- `internal/agentapi/client.go` - InspectContainer 客户端方法
- `internal/agent/server.go` - 容器状态查询端点
- `internal/controlplane/scheduler/reconciler.go` - 运行时对账逻辑
- `internal/controlplane/app/app.go` - reconciler 注入和 job 注册

## Decisions Made
- EventRecorder 接口统一定义在 router.go，所有 handler 通过构造函数注入
- 事件记录采用 fire-and-forget 模式，失败只记日志不影响主操作
- Reconciler 严格区分通信失败和容器不存在：通信失败只记日志不修改 DB（Pitfall 4 防护）

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered
None

## User Setup Required
None - no external service configuration required.

## Next Phase Readiness
- 事件审计和对账基础设施完成，后端 API 已齐全
- 可进入前端管理面板开发阶段

---
*Phase: 05-expiry-audit-cleanup*
*Completed: 2026-03-27*
