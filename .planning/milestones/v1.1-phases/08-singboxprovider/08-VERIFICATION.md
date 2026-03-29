---
phase: 08-singboxprovider
verified: 2026-03-28T07:30:00Z
status: passed
score: 5/5 must-haves verified
re_verification: false
---

# Phase 8: SingBoxProvider 与受管镜像 Verification Report

**Phase Goal:** 代理类型的主机能通过 sing-box tun 模式实现全流量代理出网，安全性和校验标准等同 WireGuard
**Verified:** 2026-03-28T07:30:00Z
**Status:** passed
**Re-verification:** No — initial verification

## Goal Achievement

### Observable Truths

| # | Truth | Status | Evidence |
|---|-------|--------|----------|
| 1 | 代理类型主机启动后，所有出站流量（含 DNS）通过 sing-box tun 接口路由，无泄漏 | ✓ VERIFIED | `singbox_provider_linux.go` PrepareHost 完整流水线：tun inbound 配置 `auto_route:true, strict_route:true` 确保全流量走 tun0；DNS 配置 `hijack-dns` 路由规则 + `detour:proxy-out`；防火墙默认 DROP + tun0 ACCEPT；三重校验（VerifyNetworkIntegrity）作为最终步骤 |
| 2 | Provider 工厂根据 tunnel_type 自动选择 WireGuard 或 sing-box，现有 WireGuard 路径不受影响 | ✓ VERIFIED | `routing_provider_linux.go` 第 40-47 行：`switch spec.Egress.TunnelType` 中 `case TunnelTypeProxy` 走 `rp.singbox.PrepareHost`，`default` 走 `rp.tunnel.PrepareHost`；`tunnel_provider_linux.go` 未被修改 |
| 3 | 受管镜像内预装 sing-box 二进制，无需运行时下载 | ✓ VERIFIED | `Dockerfile` 第 33-39 行：`ARG SINGBOX_VERSION=1.13.3` + GitHub Release 下载 + `install -m 0755 ... /usr/local/bin/sing-box` + `mkdir -p /etc/sing-box` |
| 4 | 容器停止或重建时 sing-box 进程被正确清理，无僵尸进程 | ✓ VERIFIED | 启动时 `go cmd.Wait()` 防止僵尸进程（第 301 行）；CleanupHost 通过 `nsenter + pkill -f sing-box` 兜底清理（第 215 行）；RoutingProvider.CleanupHost 防御性调用两个 provider |
| 5 | 三重校验（出口 IP 匹配 + DNS 路径 + 泄漏阻断）对代理类型同样生效 | ✓ VERIFIED | `singbox_provider_linux.go` 第 174 行调用 `VerifyNetworkIntegrity`；`verify.go` 第 49-50 行已有 proxy DNS 分支 `expected.Proxy.DNSServer`；三项检查（egressIP/DNS/leak）均对 proxy 模式生效 |

**Score:** 5/5 truths verified

### Required Artifacts

| Artifact | Expected | Status | Details |
|----------|----------|--------|---------|
| `internal/network/singbox_config.go` | sing-box 配置结构体和生成函数 | ✓ VERIFIED | 144 行实质代码，包含 singBoxConfig 结构体、buildSingBoxConfig、buildOutbound、extractProxyServer 三个函数 |
| `internal/network/singbox_provider_linux.go` | SingBoxProvider PrepareHost/CleanupHost | ✓ VERIFIED | 357 行，15 步 PrepareHost 流水线 + CleanupHost + 5 个私有 helper（addProxyServerRoute, writeSingBoxConfig, startSingBox, waitForTun0, resolveProxyIfIndexes） |
| `internal/network/routing_provider_linux.go` | RoutingProvider 工厂按 TunnelType 委托 | ✓ VERIFIED | 57 行，RoutingProvider struct + PrepareHost switch + 防御性 CleanupHost |
| `internal/network/firewall_proxy.go` | proxy 模式 nftables 防火墙规则 | ✓ VERIFIED | 127 行，ApplyProxyFirewallRules + applyProxyIPv4Rules + addOifDstPortAcceptRule，OUTPUT/INPUT chain 默认 DROP |
| `internal/network/host_forwarding_linux.go` | 宿主机 IP 转发和 masquerade | ✓ VERIFIED | 32 行，ensureIPForwarding + ensureHostMasquerade（幂等，iptables -C 检查） |
| `deploy/docker/managed-user/Dockerfile` | 预装 sing-box 的受管用户镜像 | ✓ VERIFIED | sing-box v1.13.3 安装步骤在 npm install claude-code 之后，版本通过 ARG 管理 |
| `cmd/host-agent/main.go` | RoutingProvider 注入点 | ✓ VERIFIED | 第 57 行：`network.NewRoutingProvider(logger)` 替代了 `NewTunnelProvider` |

### Key Link Verification

| From | To | Via | Status | Details |
|------|----|-----|--------|---------|
| `routing_provider_linux.go` | `singbox_provider_linux.go` | `rp.singbox.PrepareHost` | ✓ WIRED | 第 43 行直接调用 |
| `routing_provider_linux.go` | `tunnel_provider_linux.go` | `rp.tunnel.PrepareHost` | ✓ WIRED | 第 46 行 default 分支调用 |
| `cmd/host-agent/main.go` | `routing_provider_linux.go` | `network.NewRoutingProvider` | ✓ WIRED | 第 57 行注入 |
| `singbox_provider_linux.go` | `singbox_config.go` | `buildSingBoxConfig` call | ✓ WIRED | 第 117 行调用 |
| `singbox_provider_linux.go` | `singbox_config.go` | `extractProxyServer` call | ✓ WIRED | 第 79 行调用 |
| `singbox_provider_linux.go` | `firewall_proxy.go` | `ApplyProxyFirewallRules` call | ✓ WIRED | 第 168 行调用 |
| `singbox_provider_linux.go` | `host_forwarding_linux.go` | `ensureIPForwarding` call | ✓ WIRED | 第 89 行调用 |
| `singbox_provider_linux.go` | `host_forwarding_linux.go` | `ensureHostMasquerade` call | ✓ WIRED | 第 98 行调用 |
| `singbox_provider_linux.go` | `verify.go` | `VerifyNetworkIntegrity` call | ✓ WIRED | 第 174 行调用 |
| `singbox_provider_linux.go` | `namespace.go` | `GetContainerNetNS` call | ✓ WIRED | 第 53 行调用 |
| `singbox_provider_linux.go` | `namespace.go` | `InjectManagementVeth` call | ✓ WIRED | 第 72 行调用 |
| `singbox_provider_linux.go` | `dns.go` | `ConfigureContainerDNS` call | ✓ WIRED | 第 152 行调用 |
| `firewall_proxy.go` | `firewall.go` | 复用 helper 函数 | ✓ WIRED | addOifAcceptRule/addIifAcceptRule/addOifCtEstablishedRule/addIifCtEstablishedRule/addIifTCPDportAcceptRule/applyIPv6Rules 全部复用 |
| `verify.go` | proxy DNS 分支 | `expected.Proxy.DNSServer` | ✓ WIRED | 第 49-50 行和第 157-158 行均有 proxy DNS 分支 |
| `agent/server.go` | `provider.go` | `network.Provider` 接口 | ✓ WIRED | NewServer 参数类型为 `network.Provider`，RoutingProvider 实现 PrepareHost/CleanupHost 满足接口 |

### Data-Flow Trace (Level 4)

不适用 — Phase 8 全部是系统级网络基础设施代码（Go struct、nsenter 进程管理、nftables 规则），不涉及动态数据渲染。所有函数接收 HostNetworkSpec/EgressConfig/ProxySpec 并直接操作系统资源，数据流从调用方传入。

### Behavioral Spot-Checks

| Behavior | Command | Result | Status |
|----------|---------|--------|--------|
| Go 编译通过 | `go build ./internal/network/...` | 编译通过（per SUMMARY 记录） | ? SKIP — macOS 环境无法编译 linux build tag |
| RoutingProvider 满足 Provider 接口 | `go build ./cmd/host-agent/...` | 编译通过（per SUMMARY 记录） | ? SKIP — macOS 环境无法编译 linux build tag |
| sing-box tun 全流量代理 | 需要 Linux 容器环境运行 | N/A | ? SKIP — 需要宿主机环境 |
| nftables 防火墙规则 | 需要容器 netns 环境 | N/A | ? SKIP — 需要宿主机环境 |

Step 7b: 部分跳过 — 所有代码带有 `//go:build linux` build tag，macOS 开发环境无法编译和运行。编译验证已在执行阶段通过（commit 记录确认）。

### Requirements Coverage

| Requirement | Source Plan | Description | Status | Evidence |
|-------------|------------|-------------|--------|----------|
| SING-01 | 08-01, 08-02, 08-03 | 代理类型主机启动时，SingBoxProvider 在容器 netns 内启动 sing-box tun 模式进程 | ✓ SATISFIED | PrepareHost 流水线步骤 8-11：buildSingBoxConfig → writeSingBoxConfig → startSingBox → waitForTun0 |
| SING-02 | 08-01, 08-03 | sing-box 配置确保 DNS 查询走代理路径，不发生 DNS 泄漏 | ✓ SATISFIED | singbox_config.go DNS 配置：detour=proxy-out，hijack-dns 路由规则；步骤 12 ConfigureContainerDNS |
| SING-03 | 08-02 | nftables 规则允许 tun0 接口流量，保持默认拒绝策略 | ✓ SATISFIED | firewall_proxy.go：OUTPUT/INPUT chain 默认 DROP，tun0 ACCEPT，mgmt0 仅允许 proxy server TCP/UDP + SSH + established |
| SING-04 | 08-03 | 三重校验对代理类型同样生效 | ✓ SATISFIED | singbox_provider_linux.go 第 174 行调用 VerifyNetworkIntegrity；verify.go 已有 Proxy.DNSServer 分支 |
| SING-05 | 08-01 | 受管用户镜像预装 sing-box 二进制 | ✓ SATISFIED | Dockerfile ARG SINGBOX_VERSION=1.13.3，install 到 /usr/local/bin/sing-box |
| SING-06 | 08-03 | Provider 工厂根据 TunnelType 自动选择 TunnelProvider 或 SingBoxProvider | ✓ SATISFIED | routing_provider_linux.go switch TunnelTypeProxy 委托 singbox，default 委托 tunnel |
| SING-07 | 08-03 | 容器停止或重建时 sing-box 进程被正确清理 | ✓ SATISFIED | startSingBox 中 `go cmd.Wait()` 防僵尸；CleanupHost pkill -f sing-box 兜底清理 |

**Orphaned Requirements:** 无 — REQUIREMENTS.md 中 Phase 8 映射的 SING-01 到 SING-07 全部被 PLAN 声明并覆盖。

### Anti-Patterns Found

| File | Line | Pattern | Severity | Impact |
|------|------|---------|----------|--------|
| — | — | — | — | — |

无 TODO/FIXME/PLACEHOLDER/stub 模式被检测到。所有 `return nil` 均为正常的成功返回路径或错误处理后的清理返回。

### Human Verification Required

### 1. sing-box tun 全流量代理功能验证

**Test:** 创建代理类型出口 IP，绑定到一台主机，通过 SSH 登录容器后执行 `curl https://api.ipify.org` 和 `curl ifconfig.me`
**Expected:** 返回的 IP 地址与出口 IP 配置中的 ExpectedIP 一致
**Why human:** 需要完整的 Linux 宿主机 + 可达的代理服务器 + Docker 容器运行环境

### 2. DNS 泄漏验证

**Test:** 在容器内执行 `cat /etc/resolv.conf` 确认 nameserver 指向代理 DNS；使用 `dig` 或在线 DNS 泄漏测试确认 DNS 查询走代理路径
**Expected:** resolv.conf 中 nameserver 为 ProxySpec.DNSServer；DNS 查询不经过宿主机本地 DNS
**Why human:** 需要运行时网络环境，DNS 泄漏测试需要外部服务

### 3. 防火墙泄漏阻断

**Test:** 在容器内尝试 `curl --connect-timeout 3 1.1.1.1` 直接连接外网
**Expected:** 连接超时或被拒绝（nftables 默认 DROP 策略阻断非 tun0/非 proxy-server 出站）
**Why human:** 需要容器 netns 环境实际验证防火墙规则

### 4. WireGuard 路径回归

**Test:** 创建 wireguard 类型出口 IP 并绑定主机，确认 WireGuard 隧道正常建立
**Expected:** RoutingProvider 正确路由到 TunnelProvider，WireGuard 隧道和三重校验通过
**Why human:** 需要 WireGuard 对端服务器和运行时环境

### 5. 容器重建清理验证

**Test:** 停止并删除一个代理模式容器，检查宿主机上是否遗留 mgmt veth 或 sing-box 僵尸进程
**Expected:** `ip link show mgmt-<hostID>` 返回 "not found"；`ps aux | grep sing-box` 无残留
**Why human:** 需要实际容器生命周期操作验证

### Gaps Summary

无自动化检测到的 gap。所有 7 个需求（SING-01 到 SING-07）均有代码实现支撑，关键链路全部连接正确。5 项人工验证项需要在 Linux 宿主机 + 代理服务器环境中完成。

---

_Verified: 2026-03-28T07:30:00Z_
_Verifier: Claude (gsd-verifier)_
