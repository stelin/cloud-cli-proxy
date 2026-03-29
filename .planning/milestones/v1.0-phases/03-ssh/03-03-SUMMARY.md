---
phase: 03-ssh
plan: "03"
subsystem: handoff
tags: [ssh-handoff, bootstrap-errors, exec-ssh, exit-codes, e2e]

requires:
  - phase: 03-ssh
    provides: WaitForSSHReady SSH readiness gate 与 ssh.ready 事件
  - phase: 03-ssh
    provides: POST /v1/bootstrap/sessions 认证入口与 GET /v1/bootstrap/tasks/{taskID} 状态查询
provides:
  - GET /v1/bootstrap/tasks/{taskID}/handoff SSH 交接载荷 API
  - BuildSSHHandoffMetadata host-agent 侧 handoff 元数据生成
  - DeriveManagementSSHAccess 管理网段 SSH 接入参数推导
  - 稳定 error_code → 中文提示 → 非零退出码映射
  - 完整 bootstrap 脚本闭环（poll → handoff → exec ssh）
affects: [05-admin, deployment]

tech-stack:
  added: []
  patterns: [ssh-handoff-event, error-code-mapping, exec-ssh-handoff]

key-files:
  created:
    - internal/network/ssh_access.go
    - internal/runtime/tasks/ssh_handoff.go
    - internal/runtime/tasks/ssh_handoff_test.go
    - internal/controlplane/http/bootstrap_errors.go
    - internal/controlplane/http/bootstrap_handoff.go
    - internal/controlplane/http/bootstrap_handoff_test.go
    - test/bootstrap/e2e_bootstrap_ssh.sh
  modified:
    - internal/runtime/tasks/worker.go
    - internal/controlplane/http/router.go
    - deploy/bootstrap/cloud-bootstrap.sh

key-decisions:
  - "管理网段 SSH 接入参数通过 DeriveManagementSSHAccess 集中推导，避免多处复制地址算法"
  - "worker 在 ssh.ready 后写入 ssh.handoff.ready 事件，元数据包含 ssh_host/ssh_port/ssh_user/host_id"
  - "error_code 映射由 BootstrapErrorEntries 统一管理，API 与脚本共享同一来源"
  - "脚本失败路径只给显式重试建议（请重试命令 …），不做自动重试"

patterns-established:
  - "ssh.handoff.ready event: worker 在 SSH 就绪后追加 handoff 元数据事件，控制面通过事件查询提供给 API"
  - "BootstrapErrorEntries: error_code → message → exit_code 的单一真相来源"
  - "exec ssh handoff: 脚本在获取到 handoff 载荷后 exec ssh 完成会话接管"

requirements-completed: [ACCS-03, ACCS-04]

duration: 5min
completed: 2026-03-27
---

# Phase 03 Plan 03: SSH Handoff、失败码映射与脚本接入闭环 Summary

**host-agent ssh.handoff.ready 元数据 + GET handoff API + 稳定 error_code/exit_code 映射 + bootstrap 脚本 poll→handoff→exec ssh 完整闭环**

## Performance

- **Duration:** 5 min
- **Started:** 2026-03-27T09:19:53Z
- **Completed:** 2026-03-27T09:24:50Z
- **Tasks:** 2
- **Files modified:** 10

## Accomplishments
- DeriveManagementSSHAccess 集中推导容器管理网段 SSH 接入参数（host/port），避免多处复制 /30 子网算法
- BuildSSHHandoffMetadata 生成标准 ssh_host/ssh_port/ssh_user/host_id 元数据
- worker 在 ssh.ready 成功后追加 ssh.handoff.ready 事件，将 handoff 元数据写入事件流
- GET /v1/bootstrap/tasks/{taskID}/handoff 仅在任务 succeeded 且存在 ssh.handoff.ready 事件时返回交接载荷
- BootstrapErrorEntries 统一管理 7 个 error_code 的中文提示和非零退出码映射
- bootstrap 脚本完整闭环：轮询状态 → 拉取 handoff → exec ssh 自动接入
- 脚本不含自动重试循环（D-11）和 Web Terminal fallback（D-09）
- 15 个自动化测试全覆盖：4 个 handoff metadata + 4 个 handoff handler + 6 个 error mapping + 7 个 e2e 脚本验证

## Task Commits

Each task was committed atomically:

1. **Task 1: SSH handoff 元数据与事件** - `0c43fcd` (test) + `e5d045a` (feat)
2. **Task 2: Handoff API、失败码映射与脚本闭环** - `1f94185` (test) + `43c8375` (feat)

_Note: TDD tasks have separate test → feat commits_

## Files Created/Modified
- `internal/network/ssh_access.go` - DeriveManagementSSHAccess 管理网段 SSH 参数推导
- `internal/runtime/tasks/ssh_handoff.go` - BuildSSHHandoffMetadata 元数据生成函数
- `internal/runtime/tasks/ssh_handoff_test.go` - 4 场景 metadata 测试
- `internal/runtime/tasks/worker.go` - ssh.ready 后追加 ssh.handoff.ready 事件
- `internal/controlplane/http/bootstrap_errors.go` - 7 error_code 稳定映射
- `internal/controlplane/http/bootstrap_handoff.go` - GET /v1/bootstrap/tasks/{taskID}/handoff handler
- `internal/controlplane/http/bootstrap_handoff_test.go` - 4 handoff + 6 error mapping 测试
- `internal/controlplane/http/router.go` - 挂载 handoff 路由
- `deploy/bootstrap/cloud-bootstrap.sh` - 完整闭环：poll → handoff → exec ssh
- `test/bootstrap/e2e_bootstrap_ssh.sh` - 7 个 e2e 脚本结构验证

## Decisions Made
- 管理网段 SSH 参数通过 DeriveManagementSSHAccess 集中推导，与 namespace.go 使用相同 /30 子网算法
- ssh.handoff.ready 事件在 ssh.ready 后自动写入，包含完整 SSH 接入元数据
- BootstrapErrorEntries 作为 error_code 映射的唯一来源，API 和脚本共享
- 脚本失败路径仅输出"请重试命令 …"的显式建议，不做自动重试或隐式恢复

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered
None

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness
- Phase 03 所有三个 Plan 全部完成
- curl → 认证 → 启动提示 → SSH 闭环已可用
- 错误码、退出码和中文提示框架已建立，Phase 05 后台管理可扩展
- handoff 事件模型可支持后续 session_entering 等阶段扩展

## Self-Check: PASSED

All 10 files verified present. All 4 commits verified in git log.

---
*Phase: 03-ssh*
*Completed: 2026-03-27*
