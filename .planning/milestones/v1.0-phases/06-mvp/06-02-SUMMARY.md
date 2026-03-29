---
phase: 06-mvp
plan: 02
subsystem: testing
tags: [go-test, bats, mock, scheduler, bootstrap, expiry, reconciler]

requires:
  - phase: 05-expiry-audit-cleanup
    provides: ExpiryScanner 和 Reconciler 调度器实现
  - phase: 03-bootstrap-ssh
    provides: bootstrap 脚本和 BootstrapErrorEntries 错误码契约

provides:
  - 到期扫描器 ExpiryScanner 的 5 个 mock 单元测试
  - 对账器 Reconciler 的 6 个 mock 单元测试
  - bootstrap 脚本 7 个 BATS 契约测试（覆盖错误码映射和连接失败场景）

affects: [06-mvp]

tech-stack:
  added: [bats@1.13.0]
  patterns: [mock struct + table-driven Go tests, BATS with Python mock HTTP server]

key-files:
  created:
    - internal/controlplane/scheduler/expiry_test.go
    - internal/controlplane/scheduler/reconciler_test.go
  modified:
    - tests/smoke/bootstrap.bats
    - tests/smoke/test_helper/common.bash
    - deploy/bootstrap/cloud-bootstrap.sh
    - package.json

key-decisions:
  - "mock struct 直接实现接口方法并记录调用参数，与 bootstrap_auth_test.go 的 stub 模式一致"
  - "BATS 测试使用 Python3 http.server 作为 mock HTTP 服务器，通过动态端口避免冲突"

patterns-established:
  - "scheduler 测试模式: mock store + mock queuer/inspector，验证调用参数而非返回值"
  - "BATS 脚本测试模式: start_mock_server + get_free_port + kill_mock_server 生命周期管理"

requirements-completed: [ACCS-01, ACCS-03, NET-05]

duration: 12min
completed: 2026-03-27
---

# Phase 06 Plan 02: 调度器与 Bootstrap 自动化测试 Summary

**ExpiryScanner/Reconciler 11 个 mock 单元测试 + bootstrap 脚本 7 个 BATS 错误码契约测试，修复脚本 set -eo pipefail 下两个退出码 bug**

## Performance

- **Duration:** 12 min
- **Started:** 2026-03-27T17:36:29Z
- **Completed:** 2026-03-27T17:48:34Z
- **Tasks:** 2
- **Files modified:** 6

## Accomplishments
- ExpiryScanner 5 个测试覆盖：空列表、标记过期、停止主机、store 错误、多用户继续处理
- Reconciler 6 个测试覆盖：全部健康、容器停止、容器不存在、inspect 错误跳过（Pitfall 4 验证）、陈旧任务、空主机列表
- BATS 7 个测试覆盖 auth_invalid(10)、account_disabled(11)、account_expired(12)、host_not_found(13)、unknown(1)、connection refused(2)、HTTP 500(2)
- 修复 bootstrap 脚本在非终端 stdin 下退出码被 EXIT trap 覆盖的 bug
- 修复 bootstrap 脚本在 set -eo pipefail 下 grep 无匹配导致脚本中断的 bug

## Task Commits

Each task was committed atomically:

1. **Task 1: 到期扫描器与对账器单元测试** - `789d96f` (test)
2. **Task 2: BATS bootstrap 脚本契约测试** - `7dc1980` (test + fix)

## Files Created/Modified
- `internal/controlplane/scheduler/expiry_test.go` - ExpiryScanner 5 个 mock 测试
- `internal/controlplane/scheduler/reconciler_test.go` - Reconciler 6 个 mock 测试
- `tests/smoke/bootstrap.bats` - 7 个错误码契约测试
- `tests/smoke/test_helper/common.bash` - BATS 测试辅助函数（mock HTTP server）
- `deploy/bootstrap/cloud-bootstrap.sh` - 修复 EXIT trap 和 grep pipefail bug
- `package.json` - 添加 bats devDependency

## Decisions Made
- mock struct 直接实现接口方法并记录调用参数，与 bootstrap_auth_test.go 的 stub 模式一致
- BATS 测试使用 Python3 http.server 作为 mock HTTP 服务器，通过动态端口避免冲突
- bootstrap 脚本 bug 修复作为测试过程中的自动修复（Deviation Rule 1），不改变任何预期行为

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] EXIT trap 中 stty echo 失败导致退出码被覆盖**
- **Found during:** Task 2 (BATS 测试)
- **Issue:** macOS bash 3.2 下 `set -e` 在 EXIT trap 中生效，`stty echo` 因非终端 stdin 返回 1，覆盖脚本的 `exit 2` 退出码
- **Fix:** 将 `stty echo 2>/dev/null` 改为 `stty echo 2>/dev/null || true`
- **Files modified:** deploy/bootstrap/cloud-bootstrap.sh
- **Verification:** 连接失败场景正确返回 exit 2
- **Committed in:** 7dc1980

**2. [Rule 1 - Bug] grep -o 无匹配时在 set -eo pipefail 下中断脚本**
- **Found during:** Task 2 (BATS 测试 HTTP 500 场景)
- **Issue:** 响应体不含 `error_code` 时，`grep -o` 返回 1，`pipefail` 将其传播，`set -e` 中断脚本
- **Fix:** 所有 `grep | cut` 管道末尾添加 `|| true`
- **Files modified:** deploy/bootstrap/cloud-bootstrap.sh
- **Verification:** HTTP 500 无 error_code 场景正确返回 exit 2，正常成功流程不受影响
- **Committed in:** 7dc1980

---

**Total deviations:** 2 auto-fixed (2 bugs)
**Impact on plan:** 两个修复都是 bootstrap 脚本中已有的 bug，修复后不改变预期行为，仅确保退出码契约正确传播。

## Issues Encountered
None

## User Setup Required
None - no external service configuration required.

## Next Phase Readiness
- 调度器和 bootstrap 脚本的回归测试基线已建立
- `go test ./internal/controlplane/scheduler/ -count=1` 和 `npx bats tests/smoke/bootstrap.bats` 均可作为 CI 检查点

## Self-Check: PASSED

## Self-Check: PASSED

- All 6 files verified present on disk
- Commit 789d96f verified in git log
- Commit 7a7a0a3 verified in git log
- `go test ./internal/controlplane/scheduler/ -count=1` exits 0 (11 tests pass)
- `npx bats tests/smoke/bootstrap.bats` exits 0 (7 tests pass)

---
*Phase: 06-mvp*
*Completed: 2026-03-27*
