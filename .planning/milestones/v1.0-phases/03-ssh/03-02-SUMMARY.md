---
phase: 03-ssh
plan: "02"
subsystem: runtime
tags: [ssh-readiness, bootstrap-status, stage-mapping, polling, tdd]

requires:
  - phase: 03-ssh
    provides: POST /v1/bootstrap/sessions 认证入口与 QueueHostAction 启动任务入队
  - phase: 02-tunnel-egress-enforcement
    provides: PrepareHost 网络就绪校验与 net.ready 事件
provides:
  - WaitForSSHReady SSH readiness gate（阻止假就绪接入）
  - GET /v1/bootstrap/tasks/{taskID} 阶段化进度查询接口
  - GetTaskByID 和 ListEventsByTaskID 仓储查询
  - D-06 固定阶段映射（host_starting → runtime_validating → ssh_ready）
affects: [03-03, 05-admin]

tech-stack:
  added: []
  patterns: [ssh-readiness-gate, stage-mapping-by-event, injectable-checker]

key-files:
  created:
    - internal/runtime/tasks/ssh_ready.go
    - internal/runtime/tasks/ssh_ready_test.go
    - internal/controlplane/http/bootstrap_status.go
    - internal/controlplane/http/bootstrap_status_test.go
  modified:
    - internal/runtime/tasks/worker.go
    - internal/store/repository/queries.go
    - internal/controlplane/http/router.go

key-decisions:
  - "WaitForSSHReady 使用可注入的 SSHReadyConfig.Check 函数，生产用 docker exec TCP 探测，测试用 mock"
  - "SSH readiness 失败返回 SSHNotReadyError，Execute() 映射为 ssh_not_ready error_code"
  - "阶段映射通过事件流反向扫描确定最高到达阶段，不依赖额外状态字段"
  - "startHost 和 rebuildHost 均在 PrepareHost 之后调用 WaitForSSHReady，防止假就绪"

patterns-established:
  - "SSHReadyConfig injectable checker: 生产与测试共享同一 WaitForSSHReady 逻辑，仅替换 Check 函数"
  - "resolveStage event-walk: 反向遍历事件流确定阶段，新阶段只需在 stagesByEventType 注册"

requirements-completed: [ACCS-02, RUNT-03]

duration: 5min
completed: 2026-03-27
---

# Phase 03 Plan 02: 启动进度展示与 SSH 就绪门槛 Summary

**SSH readiness gate 阻止假就绪接入 + GET /v1/bootstrap/tasks/{taskID} 阶段化进度轮询（D-06 固定映射）**

## Performance

- **Duration:** 5 min
- **Started:** 2026-03-27T09:07:50Z
- **Completed:** 2026-03-27T09:13:01Z
- **Tasks:** 2
- **Files modified:** 7

## Accomplishments
- WaitForSSHReady 在 startHost/rebuildHost 的 PrepareHost 之后作为门槛，只有 SSH 端口可达才允许任务成功
- GET /v1/bootstrap/tasks/{taskID} 返回 task_status + stage_code + stage_text，覆盖 pending/running/succeeded/failed 全状态
- D-06 固定阶段映射：host_starting → runtime_validating → ssh_ready，通过事件流自动推断
- 9 个自动化测试全覆盖：4 个 SSH readiness 场景 + 5 个 status handler 场景

## Task Commits

Each task was committed atomically:

1. **Task 1: SSH readiness gate** - `ce9ebee` (test) + `14155a9` (feat)
2. **Task 2: Bootstrap status handler** - `e1f69e0` (test) + `0cdc50e` (feat)

_Note: TDD tasks have separate test → feat commits_

## Files Created/Modified
- `internal/runtime/tasks/ssh_ready.go` - WaitForSSHReady 轮询逻辑与 SSHNotReadyError 类型
- `internal/runtime/tasks/ssh_ready_test.go` - 成功/重试/超时/取消 4 场景测试
- `internal/runtime/tasks/worker.go` - startHost/rebuildHost 集成 SSH readiness gate + 事件记录
- `internal/controlplane/http/bootstrap_status.go` - GET /v1/bootstrap/tasks/{taskID} handler
- `internal/controlplane/http/bootstrap_status_test.go` - 5 场景阶段映射测试
- `internal/store/repository/queries.go` - 新增 GetTaskByID、ListEventsByTaskID 查询
- `internal/controlplane/http/router.go` - 挂载 bootstrap status 路由 + BootstrapTasks/Events 依赖

## Decisions Made
- WaitForSSHReady 使用 SSHReadyConfig 可注入 checker，生产用 `docker exec bash -c "</dev/tcp/127.0.0.1/22"`，测试用 mock
- SSHNotReadyError 独立类型，Execute() 通过 errors.As 判断并设置 ssh_not_ready error_code
- 阶段映射采用事件流反向扫描（ssh.ready > runtime.validating > net.ready），不引入额外状态字段
- 失败响应包含 error_code + error_message + retryable=false，满足终端直接展示需求

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered
None

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness
- SSH readiness gate 已生效，03-03 可在此基础上实现 session_entering 阶段与 SSH handoff
- bootstrap status 接口已挂载，脚本可轮询进度并在 ssh_ready 后执行 exec ssh
- router.go 已预留 BootstrapTasks / BootstrapEvents 注入点

## Self-Check: PASSED

All 7 files verified present. All 4 commits verified in git log.

---
*Phase: 03-ssh*
*Completed: 2026-03-27*
