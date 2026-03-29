---
phase: 02-tunnel-egress-enforcement
verified: 2026-03-27T15:02:00Z
status: passed
score: 13/13 must-haves verified
---

# Phase 02: Tunnel Egress Enforcement Verification Report

**Phase Goal:** 保证每个可运行用户容器都绑定出口 IP，并且只能通过受控隧道出网。
**Verified:** 2026-03-27T15:02:00Z
**Status:** passed
**Re-verification:** No — initial verification

## Goal Achievement

### Observable Truths

| # | Truth | Status | Evidence |
|---|-------|--------|----------|
| 1 | 没有出口 IP 绑定的主机在启动阶段立即失败，不做自动重试 | ✓ VERIFIED | `ValidateEgressBinding` 在 `worker.go:243` 调用；失败返回 `ErrBindingMissing` NetworkError；`startHost` 调用 `validateAndPrepare` 在 docker start 之前 (line 126)；无重试逻辑 |
| 2 | 出口 IP 的隧道配置可以通过 egress_ips 表持久化存储 | ✓ VERIFIED | `0002_egress_tunnel.sql` 添加 `wg_endpoint`, `wg_public_key`, `wg_preshared_key`, `wg_allowed_ips`, `wg_dns_server`, `wg_peer_address` 列；`EgressIP` 模型在 `models.go:63-77` 映射所有字段；`GetEgressIP`/`GetEgressIPByHost` 查询在 `queries.go:210-271` |
| 3 | 网络失败类型按 D-12 要求至少区分六种 | ✓ VERIFIED | `errors.go:8-15` 定义了六种：`ErrBindingMissing`, `ErrEgressIPMismatch`, `ErrDNSLeak`, `ErrLeakNotBlocked`, `ErrEgressUnreachable`, `ErrTunnelSetupFailed`；`errors_test.go` 覆盖全部六种类型的 `Error()`/`EventType()`/`EventMetadata()` |
| 4 | 容器使用 --network=none 创建，不接入任何 Docker 默认网络 | ✓ VERIFIED | `worker.go:105`: `"--network", "none"` 在 docker create args 中 |
| 5 | 网络准备在 docker start 之后、就绪标记之前执行（D-06） | ✓ VERIFIED | `startHost` 流程：`validateAndPrepare` (L126) → `createHost` (L134) → `docker start` (L139) → `buildEgressConfig` (L143) → `PrepareHost` (L152) → `RecordEvent net.ready` (L157)；`rebuildHost` 同一模式 (L206-234) |
| 6 | 用户容器通过 --network=none 启动后，只拥有 WireGuard 隧道接口、管理 veth 和 loopback | ✓ VERIFIED | `tunnel_provider_linux.go:PrepareHost` 按顺序注入：`InjectManagementVeth` (L69) → `InjectWireGuard` (L76)；容器以 `--network=none` 创建，无 Docker 默认网络 |
| 7 | 容器内所有出站流量的默认路由指向 WireGuard 隧道接口 | ✓ VERIFIED | `wireguard.go:162-171`: `RouteAdd` 添加 `0.0.0.0/0` 默认路由指向 wg 接口 index |
| 8 | 容器内 DNS 解析器配置为隧道侧 DNS 服务器 | ✓ VERIFIED | `dns.go:14-30`: `ConfigureContainerDNS` 写入 `/proc/<pid>/root/etc/resolv.conf`，内容仅包含 `nameserver <dnsServer>`；不含 `127.0.0.53`（grep 确认） |
| 9 | 容器 netns 内 nftables OUTPUT chain 默认拒绝 | ✓ VERIFIED | `firewall.go:46,59`: `policyDrop := nftables.ChainPolicyDrop`，OUTPUT chain policy 设为 DROP；只允许 loopback、WireGuard 和管理 veth established/related |
| 10 | 管理 veth 对提供宿主机到容器的 SSH 管理路径，但不允许容器通过 veth 出网 | ✓ VERIFIED | `namespace.go:57-171`: `InjectManagementVeth` 不为容器侧 mgmt0 配默认网关；`firewall.go:64`: OUTPUT chain 中 mgmt 接口只允许 established/related (非新连接)；`firewall.go:77-78`: INPUT chain 中 mgmt 只允许 SSH (dport 22) + established/related |
| 11 | 主机在被标记为可用前，必须通过出口 IP 匹配、DNS 路径正确、非隧道直连被阻断的三重校验 | ✓ VERIFIED | `tunnel_provider_linux.go:100-123`: `VerifyNetworkIntegrity` 在 PrepareHost 步骤 8 执行；`verify.go:37-56` 运行三项检查：`verifyEgressIP` (curl api.ipify.org)、`verifyDNS` (cat resolv.conf)、`verifyLeakBlocked` (bash /dev/tcp 直连测试)；全部通过才返回 nil |
| 12 | 任一校验失败时主机立即标记为失败，并按 D-12 要求的类型细分记录事件 | ✓ VERIFIED | `verify.go:118-163`: `firstNetworkError` 返回类型化 NetworkError（ErrEgressIPMismatch / ErrEgressUnreachable / ErrDNSLeak / ErrLeakNotBlocked）；`worker.go:152-154/223-225`: PrepareHost 失败时调用 `recordNetworkError`；`worker.go:262-272`: 解析 NetworkError 后通过 `RecordEvent` 记录到 events 表 |
| 13 | 宿主机预检脚本检查 WireGuard 内核模块、nsenter 和 nft 工具的可用性 | ✓ VERIFIED | `host-preflight.sh:21-28`: `modprobe -n wireguard` 或 `ip link add wg-test type wireguard`；`host-preflight.sh:31`: `require_cmd nsenter`；`host-preflight.sh:34`: `require_cmd nft`；`host-preflight.sh:37`: `require_cmd curl`；`bash -n` 语法检查通过 |

**Score:** 13/13 truths verified

### Required Artifacts

| Artifact | Expected | Status | Details |
|----------|----------|--------|---------|
| `internal/network/types.go` | EgressConfig, TunnelSpec, HostNetworkSpec 共享类型 | ✓ VERIFIED | 29 行；导出 TunnelSpec (6 fields)、EgressConfig (3 fields)、HostNetworkSpec (3 fields) |
| `internal/network/errors.go` | 按类型细分的网络错误体系 | ✓ VERIFIED | 43 行；NetworkErrorType enum (6 consts)、NetworkError struct with Error()/EventType()/EventMetadata() |
| `internal/network/errors_test.go` | 六种错误类型测试 | ✓ VERIFIED | 72 行；覆盖 6 种类型 + metadata merge + nil metadata |
| `internal/network/validate.go` | 启动前绑定校验逻辑 | ✓ VERIFIED | 103 行；EgressValidator 接口、EgressIPRecord、ValidateEgressBinding 函数 |
| `internal/network/validate_test.go` | 绑定校验单元测试 | ✓ VERIFIED | 140 行；4 场景：missing binding、incomplete config、success、keys error |
| `internal/store/migrations/0002_egress_tunnel.sql` | egress_ips 隧道字段扩展和 hosts WireGuard 密钥列 | ✓ VERIFIED | 11 行；6 列加到 egress_ips + 2 列加到 hosts |
| `internal/store/repository/models.go` | EgressIP 模型 | ✓ VERIFIED | EgressIP struct (lines 63-77) 含 WgEndpoint、WgPublicKey 等 |
| `internal/store/repository/queries.go` | GetEgressIP 和 GetEgressIPByHost 查询 | ✓ VERIFIED | GetEgressIP (L210)、GetEgressIPByHost (L240)、GetHostWgKeys (L273)、SetHostWgKeys (L292) |
| `internal/network/namespace.go` | 容器 netns 获取、veth 对创建与注入 | ✓ VERIFIED | 183 行；`//go:build linux`；GetContainerNetNS、InjectManagementVeth with runtime.LockOSThread |
| `internal/network/wireguard.go` | WireGuard 接口创建、配置和 netns 注入 | ✓ VERIFIED | 174 行；`//go:build linux`；GenerateWireGuardKeys、InjectWireGuard with LinkSetNsFd + RouteAdd |
| `internal/network/firewall.go` | nftables 默认拒绝策略 | ✓ VERIFIED | 222 行；`//go:build linux`；ApplyFirewallRules with ChainPolicyDrop (IPv4 + IPv6)；辅助函数 per rule pattern |
| `internal/network/dns.go` | 容器 DNS 配置 | ✓ VERIFIED | 30 行；`//go:build linux`；ConfigureContainerDNS via /proc/<pid>/root/etc/resolv.conf |
| `internal/network/provider.go` | Provider 接口（跨平台） | ✓ VERIFIED | 10 行；仅接口定义，无 NoopProvider（grep 确认 NoopProvider 不存在于 internal/） |
| `internal/network/tunnel_provider_linux.go` | TunnelProvider 替代 NoopProvider | ✓ VERIFIED | 174 行；`//go:build linux`；完整 PrepareHost 管线 8 步 + CleanupHost |
| `internal/network/verify.go` | 出口 IP、DNS 和泄漏三重校验 | ✓ VERIFIED | 163 行；VerifyResult struct、VerifyNetworkIntegrity、nsenter prefix pattern、api.ipify.org / resolv.conf / 1.1.1.1:80 |
| `internal/network/verify_test.go` | 校验结果判定的单元测试 | ✓ VERIFIED | 112 行；AllPassed 8 种组合 + firstNetworkError 4 种优先级场景 |
| `deploy/scripts/host-preflight.sh` | 扩展后的宿主机预检脚本 | ✓ VERIFIED | 40 行；wireguard module + nsenter + nft + curl 检查；bash -n 通过 |
| `cmd/host-agent/main.go` | 使用 TunnelProvider 替代 NoopProvider | ✓ VERIFIED | `//go:build linux`；line 41: `network.NewTunnelProvider(logger)` |

### Key Link Verification

| From | To | Via | Status | Details |
|------|----|-----|--------|---------|
| `worker.go` | `validate.go` | startHost 调用 ValidateEgressBinding | ✓ WIRED | `validateAndPrepare` (L243) 和 `buildEgressConfig` (L254) 均调用 `network.ValidateEgressBinding` |
| `worker.go` | docker create | --network=none 参数 | ✓ WIRED | `worker.go:105`: `"--network", "none"` 在 args 列表中 |
| `tunnel_provider_linux.go` | `namespace.go` | PrepareHost 调用 GetContainerNetNS + InjectManagementVeth | ✓ WIRED | L52: `GetContainerNetNS`、L69: `InjectManagementVeth` |
| `tunnel_provider_linux.go` | `wireguard.go` | PrepareHost 调用 InjectWireGuard | ✓ WIRED | L76: `InjectWireGuard(hostNS, nsHandle, spec.Egress.Tunnel)` |
| `tunnel_provider_linux.go` | `firewall.go` | PrepareHost 调用 ApplyFirewallRules | ✓ WIRED | L95: `ApplyFirewallRules(nsHandle, wgIfIndex, loIfIndex, mgmtIfIndex)` |
| `tunnel_provider_linux.go` | `verify.go` | PrepareHost 在管线末尾调用 VerifyNetworkIntegrity | ✓ WIRED | L101: `VerifyNetworkIntegrity(ctx, spec.ContainerPID, *spec.Egress)` |
| `worker.go` | `errors.go` | startHost 中 PrepareHost 失败时解析 NetworkError 并记录分类事件 | ✓ WIRED | `recordNetworkError` (L262-272): `errors.As(err, &netErr)` → `RecordEvent` with `netErr.EventType()` |

### Data-Flow Trace (Level 4)

Not applicable — Phase 02 artifacts are system-level network modules (kernel netns/nftables/WireGuard operations), not UI components rendering dynamic data.

### Behavioral Spot-Checks

| Behavior | Command | Result | Status |
|----------|---------|--------|--------|
| 全项目编译通过 (macOS) | `go build ./...` | exit 0 | ✓ PASS |
| 全项目编译通过 (Linux 交叉编译) | `GOOS=linux go build ./...` | exit 0 | ✓ PASS |
| 网络包测试全绿 | `go test ./internal/network/... -count=1 -short` | ok (0.851s) | ✓ PASS |
| 预检脚本语法正确 | `bash -n deploy/scripts/host-preflight.sh` | exit 0 | ✓ PASS |
| NoopProvider 已全部替换 | `grep -r NoopProvider internal/` | 0 matches | ✓ PASS |
| 所有 netns 操作锁定 OS 线程 | `grep LockOSThread internal/network/` | 4 files matched | ✓ PASS |

### Requirements Coverage

| Requirement | Source Plan | Description | Status | Evidence |
|-------------|------------|-------------|--------|----------|
| NET-01 | 02-01 | 每个可运行容器都必须至少绑定一个出口 IP 资源；如果未绑定，启动必须失败 | ✓ SATISFIED | `ValidateEgressBinding` 返回 `ErrBindingMissing`；`startHost` 在 `validateAndPrepare` 失败时中止；`rebuildHost` 同理 |
| NET-02 | 02-02 | 用户容器中的所有出站流量都必须被强制导向指定的全隧道路由路径 | ✓ SATISFIED | WireGuard 接口注入容器 netns；默认路由 `0.0.0.0/0` 指向 wg 接口；nftables OUTPUT 默认 DROP，只允许 wg/lo/mgmt-established |
| NET-03 | 02-02 | 用户容器中的 DNS 解析也必须走受控路径 | ✓ SATISFIED | `ConfigureContainerDNS` 写入隧道侧 DNS 到 `/proc/<pid>/root/etc/resolv.conf`；不含宿主机 DNS 或 `127.0.0.53`；三重校验中 DNS check 验证一致性 |
| NET-04 | 02-02 | 对用户容器而言，任何未走隧道的出站流量都必须被默认阻断 | ✓ SATISFIED | nftables OUTPUT chain policy DROP (IPv4)；IPv6 table 同样 policy DROP (仅允许 loopback)；管理 veth OUTPUT 仅允许 established/related 回包 |
| NET-05 | 02-03 | 在主机被标记为可接入前，系统会验证出口 IP 和 DNS 路径都符合预期 | ✓ SATISFIED | `VerifyNetworkIntegrity` 作为 PrepareHost step 8 在管线末尾执行；三项检查全通过才返回 nil → 才记录 `net.ready` 事件 |

### Anti-Patterns Found

| File | Line | Pattern | Severity | Impact |
|------|------|---------|----------|--------|
| — | — | — | — | — |

无 TODO、FIXME、placeholder、空实现或硬编码空数据。全部模块实现真实逻辑。

### Human Verification Required

### 1. WireGuard 隧道端到端连通性

**Test:** 在 Linux 宿主机上启动一个绑定了出口 IP（含完整 WireGuard 配置）的容器，进入容器后执行 `curl https://api.ipify.org`
**Expected:** 返回的 IP 与 egress_ips 表中绑定的 ip_address 一致
**Why human:** 需要真实 WireGuard 远端节点和 Linux 内核环境，无法在 macOS CI 中自动化

### 2. nftables 泄漏阻断验证

**Test:** 在容器内尝试 `curl --interface mgmt0 http://example.com` 或 `ping -I mgmt0 8.8.8.8`
**Expected:** 连接被拒绝（nftables DROP）
**Why human:** 需要完整 Linux netns 和 nftables 运行环境

### 3. DNS 隧道路径验证

**Test:** 在容器内执行 `nslookup example.com`，同时在宿主机上抓包观察 DNS 流量路径
**Expected:** DNS 请求通过 WireGuard 隧道发出，宿主机网卡上看不到明文 DNS 查询
**Why human:** 需要 tcpdump 配合真实隧道流量分析

### 4. 管理 veth SSH 接入

**Test:** 从宿主机通过管理 veth 的宿主机侧 IP ssh 到容器的 mgmt0 IP
**Expected:** SSH 连接成功；从容器内无法通过 mgmt0 主动连接外网
**Why human:** 需要容器内 OpenSSH 运行中的完整环境

### Gaps Summary

无差距。所有 13 项 must-have truths 均通过自动化验证。5 项需求 (NET-01 至 NET-05) 全部在代码中有实现覆盖。所有产出物存在、有实质性代码、已接入调用链路。

4 项需要人工验证的是端到端集成场景——需要真实 Linux 内核 + WireGuard 远端节点环境，无法在普通 CI 中自动化。

---

_Verified: 2026-03-27T15:02:00Z_
_Verifier: Claude (gsd-verifier)_
