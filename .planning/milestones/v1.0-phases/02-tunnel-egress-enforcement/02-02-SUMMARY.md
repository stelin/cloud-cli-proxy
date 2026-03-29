---
phase: 02-tunnel-egress-enforcement
plan: "02"
subsystem: network
tags: [wireguard, nftables, netns, veth, dns, tunnel, firewall]

requires:
  - phase: 02-tunnel-egress-enforcement
    provides: "TunnelSpec, EgressConfig, HostNetworkSpec types, NetworkError taxonomy, ValidateEgressBinding, --network=none"
provides:
  - "GetContainerNetNS — retrieve container network namespace fd from Docker inspect"
  - "InjectManagementVeth — veth pair for SSH management without egress bypass"
  - "InjectWireGuard — birthplace namespace model WireGuard tunnel injection"
  - "GenerateWireGuardKeys — WireGuard key pair generation"
  - "ApplyFirewallRules — nftables default-deny with IPv4/IPv6 in container netns"
  - "ConfigureContainerDNS — tunnel-only resolv.conf via /proc/<pid>/root"
  - "TunnelProvider — full wiring pipeline replacing NoopProvider"
affects: [02-03-leak-verification, 03-ssh-handoff]

tech-stack:
  added: [vishvananda/netns v0.0.5, vishvananda/netlink v1.3.1, wgctrl, google/nftables v0.3.0]
  patterns: [birthplace namespace WireGuard model, nftables default-deny in container netns, //go:build linux platform separation]

key-files:
  created:
    - internal/network/namespace.go
    - internal/network/wireguard.go
    - internal/network/firewall.go
    - internal/network/dns.go
    - internal/network/tunnel_provider_linux.go
  modified:
    - internal/network/provider.go
    - cmd/host-agent/main.go
    - go.mod
    - go.sum

key-decisions:
  - "Split Provider interface (cross-platform) from TunnelProvider implementation (Linux-only) using //go:build linux"
  - "Management veth uses /30 subnet derived from hostID hash to avoid address collision"
  - "Firewall rules use interface index matching (not name) for robustness"
  - "DNS config via /proc/<pid>/root path avoids nsenter dependency"
  - "CleanupHost logs errors but never returns them to prevent blocking rebuild"

patterns-established:
  - "//go:build linux platform separation: all Linux netns/netlink/nftables code gated behind build tag"
  - "birthplace namespace WireGuard: create wg in host ns, configure, move to container ns"
  - "nftables rule composition: helper functions per rule pattern (oif accept, iif ct established, etc.)"

requirements-completed: [NET-02, NET-03, NET-04]

duration: 7min
completed: 2026-03-27
---

# Phase 02 Plan 02: Tunnel Wiring Pipeline Summary

**WireGuard birthplace-namespace 隧道注入、nftables 默认拒绝防火墙、管理 veth 和隧道 DNS 配置，TunnelProvider 替换 NoopProvider**

## Performance

- **Duration:** 7 min
- **Started:** 2026-03-27T06:35:33Z
- **Completed:** 2026-03-27T06:43:04Z
- **Tasks:** 2
- **Files modified:** 9

## Accomplishments
- 实现完整的容器网络接线管线：getNetNS → veth → WireGuard → DNS → firewall 五步顺序执行
- WireGuard 接口在宿主机 netns 创建后移入容器 netns，加密 UDP socket 留在宿主机侧（birthplace namespace 模型）
- 管理 veth 对不配默认网关，nftables 限制 veth 只允许 SSH 入站和 established/related 回包
- nftables OUTPUT/INPUT chain 默认拒绝，IPv6 出站也被默认拒绝
- DNS resolv.conf 只写隧道侧 DNS 服务器地址，不使用宿主机或 Docker 默认解析器
- TunnelProvider 完全替换 NoopProvider，全项目编译通过（macOS + Linux 交叉编译）

## Task Commits

Each task was committed atomically:

1. **Task 1: 实现 netns 操作和 WireGuard 隧道注入模块** - `1bbb3b1` (feat)
2. **Task 2: 实现 nftables 默认拒绝、DNS 受控路径和 TunnelProvider** - `2a36560` (feat)

## Files Created/Modified
- `internal/network/namespace.go` - GetContainerNetNS, InjectManagementVeth with unique /30 subnet per host
- `internal/network/wireguard.go` - GenerateWireGuardKeys, InjectWireGuard with birthplace namespace model
- `internal/network/firewall.go` - ApplyFirewallRules with IPv4 default-deny + IPv6 full block
- `internal/network/dns.go` - ConfigureContainerDNS writing tunnel-only resolv.conf
- `internal/network/tunnel_provider_linux.go` - TunnelProvider implementing full PrepareHost/CleanupHost pipeline
- `internal/network/provider.go` - Reduced to Provider interface only (cross-platform)
- `cmd/host-agent/main.go` - Switched from NoopProvider to TunnelProvider, added //go:build linux
- `go.mod` / `go.sum` - Added netns, netlink, wgctrl, nftables dependencies

## Decisions Made
- 将 Provider 接口（跨平台）与 TunnelProvider 实现（仅 Linux）分离，使用 `//go:build linux` 构建标签，确保 macOS 开发和 Linux 部署都能编译通过
- 管理 veth 使用从 hostID 派生的 /30 子网索引，避免多容器并发接线时地址冲突
- 防火墙规则使用接口索引匹配（而非名称），更健壮
- DNS 配置通过 `/proc/<pid>/root/etc/resolv.conf` 路径直接写入容器文件系统，无需 nsenter
- CleanupHost 只记录错误不返回，防止清理失败阻塞 rebuild 操作
- cmd/host-agent/main.go 添加 `//go:build linux` 标签，因为 host-agent 仅在 Linux 上运行

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] Added //go:build linux constraints for cross-platform compilation**
- **Found during:** Task 2
- **Issue:** nftables、netlink、netns 等库依赖 Linux 内核特有的系统调用和常量，在 macOS 上无法编译
- **Fix:** 将所有 Linux 特有的网络模块添加 `//go:build linux` 构建标签，将 Provider 接口与 TunnelProvider 实现分离到不同文件
- **Files modified:** namespace.go, wireguard.go, firewall.go, dns.go, provider.go (split), tunnel_provider_linux.go (new), cmd/host-agent/main.go
- **Verification:** `go build ./...`（macOS）和 `GOOS=linux go build ./...` 均通过
- **Committed in:** 2a36560 (Task 2 commit)

**2. [Rule 1 - Bug] Fixed nftables API constant names**
- **Found during:** Task 2
- **Issue:** 使用了不存在的 `expr.MetaKeyOIFINDEX` 和 `unix.NF_CT_STATE_BIT` 常量
- **Fix:** 改为正确的 `expr.MetaKeyOIF`/`expr.MetaKeyIIF` 和 `expr.CtStateBitESTABLISHED`/`expr.CtStateBitRELATED`，使用 `binaryutil.NativeEndian` 编码
- **Files modified:** internal/network/firewall.go
- **Verification:** `GOOS=linux go build ./...` 编译通过
- **Committed in:** 2a36560 (Task 2 commit)

---

**Total deviations:** 2 auto-fixed (1 blocking, 1 bug)
**Impact on plan:** Both fixes necessary for compilation correctness. No scope creep.

## Issues Encountered
None

## User Setup Required
None - no external service configuration required.

## Known Stubs
None - all functions implement real logic, no placeholder or mock data.

## Next Phase Readiness
- 完整的网络接线管线已就位，02-03 可直接实现三重校验（出口 IP、DNS、泄漏阻断）
- TunnelProvider.PrepareHost 在 docker start 后执行完整管线，为就绪前校验提供了检测基础
- CleanupHost 已实现宿主机侧接口清理，支持 rebuild 场景

## Self-Check: PASSED

- [x] internal/network/namespace.go exists
- [x] internal/network/wireguard.go exists
- [x] internal/network/firewall.go exists
- [x] internal/network/dns.go exists
- [x] internal/network/tunnel_provider_linux.go exists
- [x] internal/network/provider.go exists (interface only)
- [x] Commit 1bbb3b1 exists
- [x] Commit 2a36560 exists
- [x] `go build ./...` passes (macOS)
- [x] `GOOS=linux go build ./...` passes

---
*Phase: 02-tunnel-egress-enforcement*
*Completed: 2026-03-27*
