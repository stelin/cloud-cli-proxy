---
phase: 03-ssh
plan: "01"
subsystem: auth
tags: [bcrypt, bootstrap, curl, bash, ssh-entry]

requires:
  - phase: 01-foundation-control-plane-host-agent
    provides: QueueHostAction 异步任务编排、HTTP 路由与依赖注入模式
  - phase: 02-tunnel-egress-enforcement
    provides: 出口 IP 绑定与网络隔离（容器启动前置条件）
provides:
  - POST /v1/bootstrap/sessions 认证与 start_host 任务入队 API
  - GET /v1/bootstrap/script 受管 bootstrap 脚本分发
  - GetBootstrapUserByUsername 与 GetPrimaryHostByUserID 仓库查询
  - deploy/bootstrap/cloud-bootstrap.sh 终端交互脚本
  - 稳定错误码（auth_invalid/account_disabled/account_expired）与非零退出码映射
affects: [03-02, 03-03, 05-admin]

tech-stack:
  added: [golang.org/x/crypto/bcrypt]
  patterns: [bootstrap-auth-handler, writeBootstrapError, script-thin-layer]

key-files:
  created:
    - internal/controlplane/http/bootstrap_auth.go
    - internal/controlplane/http/bootstrap_auth_test.go
    - internal/controlplane/http/bootstrap_script.go
    - internal/controlplane/http/bootstrap_script_test.go
    - deploy/bootstrap/cloud-bootstrap.sh
  modified:
    - internal/controlplane/http/router.go
    - internal/store/repository/models.go
    - internal/store/repository/queries.go

key-decisions:
  - "bcrypt 校验密码，沿用 golang.org/x/crypto（已在 go.mod indirect）"
  - "账号状态仅通过 users.status 字段判断（active/disabled/expired），Phase 5 再补 expires_at 自动流转"
  - "认证成功后调用现有 QueueHostAction(ActionStartHost)，不新增绕过任务系统的启动路径"
  - "脚本错误码映射为固定退出码（10=auth_invalid, 11=disabled, 12=expired, 13=host_not_found, 14=start_failed）"

patterns-established:
  - "writeBootstrapError: 统一的 bootstrap 错误响应格式（error_code + message）"
  - "脚本薄层原则: bootstrap 脚本只做输入采集和 API 调用，不包含业务逻辑"

requirements-completed: [ACCS-01, ACCS-04]

duration: 6min
completed: 2026-03-27
---

# Phase 03 Plan 01: Bootstrap 认证入口与脚本分发 Summary

**bcrypt 密码认证 + 异步 start_host 任务入队 + 受管 bootstrap 脚本（密码不回显 + 稳定退出码）**

## Performance

- **Duration:** 6 min
- **Started:** 2026-03-27T08:55:55Z
- **Completed:** 2026-03-27T09:02:15Z
- **Tasks:** 2
- **Files modified:** 8

## Accomplishments
- POST /v1/bootstrap/sessions 认证入口可用：bcrypt 密码校验 + 账号状态检查（active/disabled/expired）
- 认证成功后通过现有 QueueHostAction(ActionStartHost) 入队，不新增绕过任务系统的启动路径
- 受管 bootstrap 脚本支持密码不回显输入（read -r -s），通过 GET /v1/bootstrap/script 分发
- 7 个自动化测试全覆盖：5 个认证场景 + 2 个脚本分发场景

## Task Commits

Each task was committed atomically:

1. **Task 1: 实现 bootstrap 认证入口与任务入队** - `7442eff` (test) + `a33ae38` (feat)
2. **Task 2: 发布受管 bootstrap 脚本入口** - `d026847` (test) + `4b5c60f` (feat)

_Note: TDD tasks have separate test → feat commits_

## Files Created/Modified
- `internal/controlplane/http/bootstrap_auth.go` - POST /v1/bootstrap/sessions 认证 handler
- `internal/controlplane/http/bootstrap_auth_test.go` - 认证 handler 表驱动测试（5 场景）
- `internal/controlplane/http/bootstrap_script.go` - GET /v1/bootstrap/script 脚本分发 handler
- `internal/controlplane/http/bootstrap_script_test.go` - 脚本分发测试（2 场景）
- `deploy/bootstrap/cloud-bootstrap.sh` - 受管终端 bootstrap 脚本
- `internal/controlplane/http/router.go` - 挂载 bootstrap 路由，增加 BootstrapUsers/BootstrapHosts 依赖
- `internal/store/repository/models.go` - 新增 BootstrapUserAuth 结构体
- `internal/store/repository/queries.go` - 新增 GetBootstrapUserByUsername、GetPrimaryHostByUserID 查询

## Decisions Made
- 使用 bcrypt.CompareHashAndPassword 校验密码，沿用项目已有的 golang.org/x/crypto
- 账号过期判断先通过 users.status 字段值实现（Phase 5 再补 expires_at 自动计算）
- 认证成功一律通过 QueueHostAction(ActionStartHost) 入队，保持 D-04 约束
- 脚本退出码映射为固定数值（10-14），满足 D-13 自动化检查需求

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered
None

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness
- 认证入口已就绪，可衔接 03-02 的进度轮询与状态映射
- 脚本交互与退出码框架已建立，03-03 可扩展完整失败分类
- router.go 已预留 BootstrapUsers / BootstrapHosts 注入点，后续只需在 main.go 接入 repository 实例

## Self-Check: PASSED

All 8 files verified present. All 4 commits verified in git log.

---
*Phase: 03-ssh*
*Completed: 2026-03-27*
