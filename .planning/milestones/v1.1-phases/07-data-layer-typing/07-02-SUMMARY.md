---
phase: 07-data-layer-typing
plan: 02
subsystem: network
tags: [wireguard, proxy, sing-box, egress, validation, types]

requires:
  - phase: 02-tunnel-egress
    provides: TunnelSpec, EgressConfig, ValidateEgressBinding, TunnelProvider

provides:
  - TunnelTypeWireGuard/TunnelTypeProxy constants for tunnel mode discrimination
  - ProxySpec struct for sing-box outbound config
  - Dual-mode EgressConfig (TunnelType + Tunnel/Proxy pointers)
  - validateProxyBinding with JSON validation
  - validateWireGuardBinding extracted from monolithic function
  - TunnelProvider nil-guard for proxy-mode safety

affects: [08-singbox-provider, 09-provider-factory, frontend-egress-form]

tech-stack:
  added: []
  patterns:
    - "TunnelType switch dispatch in ValidateEgressBinding"
    - "Pointer-typed optional config fields (Tunnel *TunnelSpec, Proxy *ProxySpec)"

key-files:
  created: []
  modified:
    - internal/network/types.go
    - internal/network/validate.go
    - internal/network/validate_test.go
    - internal/network/tunnel_provider_linux.go
    - internal/network/verify.go
    - internal/network/verify_test.go

key-decisions:
  - "Tunnel field changed from value TunnelSpec to pointer *TunnelSpec for nil-ability in proxy mode"
  - "EgressIPRecord extended with TunnelType and ProxyConfig fields at network layer"

patterns-established:
  - "TunnelType switch: default branch handles wireguard, explicit case for proxy"
  - "Pointer config fields: nil indicates mode not active (Tunnel=nil for proxy, Proxy=nil for wireguard)"

requirements-completed: [DATA-03]

duration: 6min
completed: 2026-03-28
---

# Phase 07 Plan 02: Network Types & Validation Summary

**EgressConfig 扩展为 wireguard/proxy 双模式，ValidateEgressBinding 按 TunnelType 分支校验，新增 ProxySpec 和 3 个 proxy 测试用例**

## Performance

- **Duration:** 6 min
- **Started:** 2026-03-28T06:18:28Z
- **Completed:** 2026-03-28T06:23:59Z
- **Tasks:** 2
- **Files modified:** 6

## Accomplishments

- EgressConfig 支持 wireguard 和 proxy 双模式，通过 TunnelType 字段区分
- ValidateEgressBinding 按 tunnel_type 分支，WireGuard 路径行为完全不变
- TunnelProvider.PrepareHost 添加 Tunnel nil guard，proxy 模式不会 panic
- 7 个 ValidateEgressBinding 测试全部通过（4 个 WireGuard + 3 个 proxy）

## Task Commits

Each task was committed atomically:

1. **Task 1: 扩展 network 类型和 ValidateEgressBinding 分支逻辑** - `ff76b5e` (feat)
2. **Task 2: 扩展 ValidateEgressBinding 测试覆盖 proxy 分支** - `16d7215` (test)

## Files Created/Modified

- `internal/network/types.go` — TunnelType 常量、ProxySpec 结构体、EgressConfig 重构
- `internal/network/validate.go` — EgressIPRecord 扩展、ValidateEgressBinding switch 分支、validateProxyBinding
- `internal/network/validate_test.go` — 3 个 proxy 测试 + Success 测试增强断言
- `internal/network/tunnel_provider_linux.go` — Tunnel nil guard + 指针解引用修复
- `internal/network/verify.go` — Tunnel 指针 nil-safe 访问（Deviation）
- `internal/network/verify_test.go` — Tunnel 字面量改为指针（Deviation）

## Decisions Made

- Tunnel 字段从值类型 TunnelSpec 改为指针 *TunnelSpec，使 proxy 模式下可为 nil
- EgressIPRecord 在 network 层扩展 TunnelType/ProxyConfig，保持 store 包解耦

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] verify.go / verify_test.go Tunnel 指针兼容**
- **Found during:** Task 1（types.go Tunnel 改为指针后）
- **Issue:** verify.go 中 `expected.Tunnel.DNSServer` 和 `expected.Tunnel.PeerEndpoint` 在 Tunnel 为 nil 时会 panic；verify_test.go 的 EgressConfig 字面量使用值类型不再编译
- **Fix:** verify.go 添加 nil guard，verify_test.go 改为指针字面量
- **Files modified:** internal/network/verify.go, internal/network/verify_test.go
- **Verification:** 所有 firstNetworkError 测试通过
- **Committed in:** ff76b5e (Task 1 commit)

**2. [Rule 3 - Blocking] tunnel_provider_linux.go InjectWireGuard 调用解引用**
- **Found during:** Task 1（Tunnel 改为指针后）
- **Issue:** InjectWireGuard 接受 TunnelSpec 值类型，但 spec.Egress.Tunnel 现在是 *TunnelSpec
- **Fix:** 调用处添加解引用 `*spec.Egress.Tunnel`
- **Files modified:** internal/network/tunnel_provider_linux.go
- **Verification:** go build 通过
- **Committed in:** ff76b5e (Task 1 commit)

---

**Total deviations:** 2 auto-fixed (2 blocking)
**Impact on plan:** 两项修复都是 Tunnel 类型改为指针后的必要兼容修正，无范围膨胀。

## Issues Encountered

None

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- network 类型系统已支持 wireguard/proxy 双模式
- Phase 8 的 SingBoxProvider 可直接使用 ProxySpec.OutboundConfig 构建 sing-box 配置
- Provider 工厂可通过 EgressConfig.TunnelType 选择 WireGuard 或 sing-box 路径

## Self-Check: PASSED

- [x] `internal/network/types.go` exists
- [x] `internal/network/validate.go` exists
- [x] `internal/network/validate_test.go` exists
- [x] `internal/network/tunnel_provider_linux.go` exists
- [x] Commit ff76b5e exists
- [x] Commit 16d7215 exists

---
*Phase: 07-data-layer-typing*
*Completed: 2026-03-28*
