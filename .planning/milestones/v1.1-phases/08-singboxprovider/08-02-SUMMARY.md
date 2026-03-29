---
phase: 08-singboxprovider
plan: 02
subsystem: network
tags: [nftables, iptables, firewall, masquerade, proxy, tun]

requires:
  - phase: 07-data-layer-typing
    provides: "EgressConfig.TunnelType / ProxySpec 类型定义"
  - phase: 08-singboxprovider plan 01
    provides: "SingBoxProvider 核心结构和配置生成"
provides:
  - "ApplyProxyFirewallRules — proxy 模式容器内 nftables 防火墙"
  - "ensureIPForwarding — 宿主机 IPv4 转发启用"
  - "ensureHostMasquerade — 管理 veth 子网 NAT 规则"
  - "addOifDstPortAcceptRule — nftables OIF+dst+port 白名单 helper"
affects: [08-singboxprovider plan 03, singbox_provider_linux]

tech-stack:
  added: []
  patterns:
    - "proxy 模式防火墙与 WireGuard 防火墙并行，独立函数不影响原有逻辑"
    - "宿主机侧用 iptables 设置 masquerade，避免与 Docker iptables 规则冲突"

key-files:
  created:
    - internal/network/firewall_proxy.go
    - internal/network/host_forwarding_linux.go
  modified: []

key-decisions:
  - "proxy 模式独立函数 ApplyProxyFirewallRules，不修改现有 ApplyFirewallRules"
  - "宿主机 masquerade 使用 iptables CLI 而非 nftables，避免与 Docker Engine 冲突"
  - "masquerade 幂等检查使用 iptables -C，存在则跳过"

patterns-established:
  - "addOifDstPortAcceptRule: 匹配 OIF + L4 proto + IPv4 dst addr + dst port 的 nftables 规则模式"

requirements-completed: [SING-03, SING-01]

duration: 1min
completed: 2026-03-28
---

# Phase 08 Plan 02: Proxy 防火墙与宿主机转发 Summary

**proxy 模式 nftables 防火墙规则（tun0/proxy server 白名单）和宿主机 IP 转发 + masquerade**

## Performance

- **Duration:** 1 min
- **Started:** 2026-03-28T07:11:49Z
- **Completed:** 2026-03-28T07:13:16Z
- **Tasks:** 2
- **Files modified:** 2

## Accomplishments
- 实现 proxy 模式容器内 nftables 防火墙：tun0 全放行、mgmt0 到 proxy server 的 TCP/UDP 白名单、默认 DROP
- 实现宿主机 IP 转发和幂等 masquerade 规则，使代理流量可通过 mgmt0 → 宿主机 → 外网

## Task Commits

Each task was committed atomically:

1. **Task 1: Proxy 模式 nftables 防火墙规则** - `9b98462` (feat)
2. **Task 2: 宿主机 IP 转发和 masquerade 规则** - `cc9d172` (feat)

## Files Created/Modified
- `internal/network/firewall_proxy.go` — ApplyProxyFirewallRules + applyProxyIPv4Rules + addOifDstPortAcceptRule
- `internal/network/host_forwarding_linux.go` — ensureIPForwarding + ensureHostMasquerade

## Decisions Made
- proxy 模式使用独立函数 ApplyProxyFirewallRules，不修改现有 WireGuard 防火墙逻辑
- 宿主机侧 masquerade 使用 iptables CLI 而非 nftables Go 库，避免与 Docker Engine 28.x 的 iptables 规则冲突
- masquerade 规则通过 `iptables -C` 幂等检查，每次 PrepareHost 调用时验证

## Deviations from Plan

None — plan executed exactly as written.

## Issues Encountered

None

## User Setup Required

None — no external service configuration required.

## Known Stubs

None — 所有函数实现完整，无占位符。

## Next Phase Readiness
- firewall_proxy.go 的 ApplyProxyFirewallRules 可直接被 SingBoxProvider.PrepareHost 调用
- host_forwarding_linux.go 的 ensureIPForwarding/ensureHostMasquerade 可被 SingBoxProvider.PrepareHost 在配置管理 veth 后调用
- Plan 03 可继续实现 RoutingProvider 工厂和 SingBoxProvider 核心流水线

---
*Phase: 08-singboxprovider*
*Completed: 2026-03-28*
