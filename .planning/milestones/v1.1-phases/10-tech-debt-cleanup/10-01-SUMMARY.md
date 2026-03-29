---
phase: 10-tech-debt-cleanup
plan: 01
subsystem: runtime, api
tags: [docker, network, sing-box, cleanup, proxy-test]

requires:
  - phase: 08-singbox-provider
    provides: SingBoxProvider 和 RoutingProvider CleanupHost 实现
  - phase: 09-frontend-proxy-test
    provides: 代理测试 API 中 vmess/ss/trojan 路径的 startLocalSingBox 调用

provides:
  - stopHost 路径与 rebuildHost 对齐的 CleanupHost 调用
  - vmess/ss/trojan 代理测试的 sing-box LookPath 预检和中文错误提示

affects: [runtime, controlplane, proxy-test]

tech-stack:
  added: []
  patterns: [defensive-cleanup-on-stop, lookpath-precheck]

key-files:
  created: []
  modified:
    - internal/runtime/tasks/worker.go
    - internal/controlplane/http/admin_egress_ip_probe.go

key-decisions:
  - "stopHost 使用与 rebuildHost 一致的 CleanupHost 调用模式，保持行为对称"
  - "LookPath 预检返回中文错误，而非依赖 cmd.Start 的底层英文错误"

patterns-established:
  - "停机路径必须清理网络资源：所有停止容器的路径都应调用 CleanupHost"
  - "外部二进制依赖预检：调用外部工具前先用 LookPath 检查可用性"

requirements-completed: [SC-1, SC-2]

duration: 1min
completed: 2026-03-28
---

# Phase 10 Plan 01: 后端技术债务修复 Summary

**stopHost 追加 CleanupHost 消除 mgmt veth 残留 + vmess/ss/trojan 代理测试添加 sing-box LookPath 预检返回中文提示**

## Performance

- **Duration:** 1 min
- **Started:** 2026-03-28T10:08:45Z
- **Completed:** 2026-03-28T10:09:35Z
- **Tasks:** 2
- **Files modified:** 2

## Accomplishments
- stopHost 在 docker stop 成功后调用 CleanupHost，与 rebuildHost 行为对齐，消除宿主机侧 mgmt veth 残留
- vmess/ss/trojan 代理测试在进入 startLocalSingBox 之前即检查 sing-box 是否可用，未安装时返回明确中文错误

## Task Commits

Each task was committed atomically:

1. **Task 1: stopHost 追加 CleanupHost 调用** - `72e795b` (fix)
2. **Task 2: getProxyDialer 添加 sing-box LookPath 预检** - `6da18e0` (fix)

## Files Created/Modified
- `internal/runtime/tasks/worker.go` - stopHost 函数追加 CleanupHost 调用
- `internal/controlplane/http/admin_egress_ip_probe.go` - vmess/ss/trojan 分支添加 LookPath 预检

## Decisions Made
- stopHost 使用与 rebuildHost 一致的 CleanupHost 调用模式，保持行为对称
- LookPath 预检返回中文错误提示（"sing-box 未安装，无法测试 %s 协议（需在控制面环境安装 sing-box）"），优于 cmd.Start 的底层英文错误

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered
None

## User Setup Required
None - no external service configuration required.

## Next Phase Readiness
- 后端技术债务 Plan 01 完成，可继续 Plan 02 的前端技术债务清理
- stopHost 和 rebuildHost 的网络清理行为现已对齐

---
*Phase: 10-tech-debt-cleanup*
*Completed: 2026-03-28*
