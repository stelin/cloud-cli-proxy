---
phase: 08-singboxprovider
plan: 03
subsystem: network
tags: [sing-box, tun, proxy, routing-provider, provider-factory, nsenter]

requires:
  - phase: 08-singboxprovider plan 01
    provides: "buildSingBoxConfig / extractProxyServer 配置生成函数"
  - phase: 08-singboxprovider plan 02
    provides: "ApplyProxyFirewallRules / ensureIPForwarding / ensureHostMasquerade"
provides:
  - "SingBoxProvider PrepareHost 15 步流水线（netns → veth → host route → forwarding → sing-box 配置启动 → DNS → 防火墙 → 三重校验）"
  - "SingBoxProvider CleanupHost（mgmt veth 删除 + sing-box 进程兜底清理）"
  - "RoutingProvider 工厂按 TunnelType 委托 TunnelProvider 或 SingBoxProvider"
  - "host-agent 注入 RoutingProvider，替代直接注入 TunnelProvider"
affects: [09-frontend-proxy, proxy-test-api]

tech-stack:
  added: []
  patterns:
    - "RoutingProvider 工厂模式：单一 Provider 接口注入，内部按 TunnelType switch 委托"
    - "SingBoxProvider 流水线：与 TunnelProvider 高度对称，差异在 WireGuard 注入 → sing-box 配置写入 + nsenter 进程启动"

key-files:
  created:
    - internal/network/singbox_provider_linux.go
    - internal/network/routing_provider_linux.go
  modified: []

key-decisions:
  - "SingBoxProvider PrepareHost 流水线与 TunnelProvider 保持高度对称，便于维护"
  - "RoutingProvider.CleanupHost 防御性调用两个 provider，防止崩溃后不确定哪个 provider 被使用"
  - "sing-box 进程通过 nsenter 在容器 PID 命名空间内启动，容器停止时自动终止"
  - "tun0 就绪轮询间隔 200ms，超时 10s"
  - "host-agent main.go 已在前序执行中切换为 NewRoutingProvider"

patterns-established:
  - "Provider 工厂委托模式：RoutingProvider 按 TunnelType 路由到具体实现"
  - "sing-box 进程管理：nsenter 后台启动 + waitForTun0 轮询就绪 + pkill 兜底清理"

requirements-completed: [SING-01, SING-02, SING-04, SING-06, SING-07]

duration: 2min
completed: 2026-03-28
---

# Phase 08 Plan 03: SingBoxProvider 核心流水线与 RoutingProvider 工厂 Summary

**SingBoxProvider 15 步 PrepareHost 流水线（tun 模式全流量代理）和 RoutingProvider 工厂按 TunnelType 自动路由到 WireGuard/sing-box**

## Performance

- **Duration:** 2 min
- **Started:** 2026-03-28T07:15:46Z
- **Completed:** 2026-03-28T07:17:34Z
- **Tasks:** 2
- **Files modified:** 2

## Accomplishments
- SingBoxProvider 完整实现 15 步 PrepareHost 流水线：容器 netns 获取 → mgmt veth 注入 → proxy server host route → 宿主机 IP 转发 + masquerade → sing-box 配置生成/写入/启动/等待 tun0 → DNS 配置 → nftables 防火墙 → 三重校验
- RoutingProvider 工厂实现 Provider 接口，按 TunnelType 委托到 TunnelProvider（wireguard）或 SingBoxProvider（proxy）
- CleanupHost 包含 mgmt veth 删除和 nsenter pkill sing-box 兜底清理
- WireGuard 路径通过 RoutingProvider 委托完全不受影响

## Task Commits

Each task was committed atomically:

1. **Task 1: SingBoxProvider PrepareHost/CleanupHost 完整实现** - `10f3912` (feat)
2. **Task 2: RoutingProvider 工厂** - `6cd7555` (feat)

## Files Created/Modified
- `internal/network/singbox_provider_linux.go` — SingBoxProvider struct + PrepareHost 15 步流水线 + CleanupHost + 5 个私有 helper 函数
- `internal/network/routing_provider_linux.go` — RoutingProvider struct + PrepareHost TunnelType switch + 防御性 CleanupHost

## Decisions Made
- SingBoxProvider 流水线与 TunnelProvider 保持高度对称结构，便于后续维护和理解
- RoutingProvider.CleanupHost 防御性调用两个 provider（tunnel + singbox），应对崩溃后不确定哪个 provider 被使用的场景
- sing-box 进程通过 nsenter -n -m -p 在容器命名空间内后台启动，cmd.Wait() 在 goroutine 中执行防止僵尸进程
- tun0 就绪检测使用 200ms 间隔轮询 + 10s 超时，平衡响应速度和系统负载
- host-agent main.go 已在前序执行中完成 NewRoutingProvider 注入（本次无需修改）

## Deviations from Plan

None — plan executed exactly as written.

## Issues Encountered

None

## User Setup Required

None — no external service configuration required.

## Known Stubs

None — 所有函数实现完整，无占位符。

## Next Phase Readiness
- Phase 08 全部 3 个 plan 完成，sing-box tun 模式全流量代理运行时功能就绪
- RoutingProvider 已注入 host-agent，代理类型主机创建时将自动走 SingBoxProvider 路径
- Phase 9（前端代理表单 + 代理测试 API）可继续推进

---
*Phase: 08-singboxprovider*
*Completed: 2026-03-28*
