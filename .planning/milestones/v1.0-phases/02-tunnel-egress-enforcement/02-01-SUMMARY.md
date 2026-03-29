---
phase: 02-tunnel-egress-enforcement
plan: "01"
subsystem: network
tags: [wireguard, egress, docker, postgres, migration]

requires:
  - phase: 01-control-plane-foundation
    provides: "Worker, Provider interface, repository, agent server, initial schema"
provides:
  - "EgressConfig / TunnelSpec / HostNetworkSpec types for tunnel wiring"
  - "6-type NetworkError taxonomy for event recording (D-12)"
  - "ValidateEgressBinding pre-start gate"
  - "0002_egress_tunnel.sql migration adding WireGuard fields"
  - "GetEgressIP / GetEgressIPByHost / GetHostWgKeys / SetHostWgKeys queries"
  - "--network=none container creation mode"
affects: [02-02-tunnel-provider, 02-03-leak-verification]

tech-stack:
  added: []
  patterns: [repoValidator adapter, structured NetworkError event recording]

key-files:
  created:
    - internal/network/types.go
    - internal/network/errors.go
    - internal/network/errors_test.go
    - internal/network/validate.go
    - internal/network/validate_test.go
    - internal/store/migrations/0002_egress_tunnel.sql
  modified:
    - internal/network/provider.go
    - internal/store/repository/models.go
    - internal/store/repository/queries.go
    - internal/runtime/tasks/worker.go
    - internal/agent/server.go

key-decisions:
  - "HostNetworkSpec moved from provider.go to types.go and expanded with ContainerPID/Egress"
  - "network package defines its own EgressIPRecord to avoid importing store — repoValidator adapter bridges the gap"
  - "ValidateEgressBinding does not generate WireGuard keys — deferred to 02-02 TunnelProvider"
  - "Binding validation runs before docker start; PrepareHost runs after docker start (D-06)"

patterns-established:
  - "repoValidator adapter: Worker adapts its repo interface to network.EgressValidator without coupling packages"
  - "NetworkError event recording: structured error → RecordEvent with typed event and metadata"

requirements-completed: [NET-01]

duration: 4min
completed: 2026-03-27
---

# Phase 02 Plan 01: Egress Binding Validation Summary

**WireGuard 隧道类型建模、6 类网络错误体系、启动前绑定校验门禁和 --network=none 容器隔离**

## Performance

- **Duration:** 4 min
- **Started:** 2026-03-27T06:28:56Z
- **Completed:** 2026-03-27T06:32:59Z
- **Tasks:** 2
- **Files modified:** 11

## Accomplishments
- 建立了 TunnelSpec / EgressConfig / HostNetworkSpec 类型体系，为后续 WireGuard 接线提供数据基础
- 实现 6 种 NetworkErrorType（D-12），每种都可生成结构化事件用于审计
- 容器创建使用 --network=none，彻底隔离 Docker 默认网络
- startHost / rebuildHost 在启动前执行绑定校验，缺少出口 IP 绑定立即失败并记录事件
- PrepareHost 移至 docker start 之后执行（D-06），为后续隧道接线预留正确执行时机

## Task Commits

Each task was committed atomically:

1. **Task 1: 创建网络类型、错误类型、数据库迁移和仓储查询** - `7fe73ae` (feat)
2. **Task 2: 实现启动前绑定校验、--network=none 和 worker 流程重构** - `f5e3714` (feat)

## Files Created/Modified
- `internal/network/types.go` - TunnelSpec, EgressConfig, expanded HostNetworkSpec
- `internal/network/errors.go` - NetworkErrorType enum and NetworkError struct with event helpers
- `internal/network/errors_test.go` - 6-type error tests covering Error(), EventType(), EventMetadata()
- `internal/network/validate.go` - EgressValidator interface and ValidateEgressBinding function
- `internal/network/validate_test.go` - binding missing, incomplete config, success, keys error scenarios
- `internal/network/provider.go` - removed HostNetworkSpec (moved to types.go)
- `internal/store/migrations/0002_egress_tunnel.sql` - WireGuard columns on egress_ips and hosts
- `internal/store/repository/models.go` - added EgressIP model
- `internal/store/repository/queries.go` - GetEgressIP, GetEgressIPByHost, GetHostWgKeys, SetHostWgKeys
- `internal/runtime/tasks/worker.go` - expanded WorkerRepo, repoValidator adapter, validateAndPrepare, --network=none, lifecycle refactor
- `internal/agent/server.go` - simplified to accept WorkerRepo interface directly

## Decisions Made
- HostNetworkSpec 从 provider.go 移到 types.go 以集中类型定义
- network 包通过 EgressIPRecord 独立类型避免导入 store 包，使用 repoValidator adapter 桥接
- ValidateEgressBinding 不负责生成 WireGuard 密钥，留给 02-02 的 TunnelProvider
- 绑定校验在 docker start 之前执行，PrepareHost 在 docker start 之后执行（符合 D-06）

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] Removed duplicate HostNetworkSpec definition**
- **Found during:** Task 1
- **Issue:** Plan specified HostNetworkSpec in types.go, but it already existed in provider.go
- **Fix:** Moved HostNetworkSpec from provider.go to types.go and expanded with new fields
- **Files modified:** internal/network/provider.go, internal/network/types.go
- **Verification:** `go build ./...` passes
- **Committed in:** 7fe73ae (Task 1 commit)

**2. [Rule 3 - Blocking] Removed unused repository import in server.go**
- **Found during:** Task 2
- **Issue:** After switching NewServer to accept WorkerRepo, the repository import became unused
- **Fix:** Removed the import
- **Files modified:** internal/agent/server.go
- **Verification:** `go build ./...` passes
- **Committed in:** f5e3714 (Task 2 commit)

---

**Total deviations:** 2 auto-fixed (2 blocking)
**Impact on plan:** Both fixes necessary for compilation. No scope creep.

## Issues Encountered
None

## User Setup Required
None - no external service configuration required.

## Next Phase Readiness
- 类型体系和绑定校验已就位，02-02 可直接实现 TunnelProvider（WireGuard namespace 接线）
- --network=none 确保容器无默认网络，02-02 需要在 PrepareHost 中完成隧道建立
- 02-03 泄漏验证可利用 NetworkError 类型体系记录检测结果

---
*Phase: 02-tunnel-egress-enforcement*
*Completed: 2026-03-27*
