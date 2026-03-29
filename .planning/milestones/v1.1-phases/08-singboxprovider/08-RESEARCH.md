# Phase 8: SingBoxProvider 与受管镜像 - Research

**Researched:** 2026-03-28
**Domain:** sing-box TUN 模式全流量代理、容器网络编排、nftables 防火墙适配
**Confidence:** HIGH

## Summary

Phase 8 的核心是在容器网络命名空间内，使用 sing-box 的 TUN 模式作为全流量代理出口——与 Phase 2 中 WireGuard 的角色等价。sing-box 在容器内创建 tun0 虚拟网卡，通过 `auto_route: true` 接管内核路由，所有用户流量（包括 DNS）都经由代理协议出网。代理连接本身通过管理 veth (`mgmt0`) 绑定 (`bind_interface`) 发出，避免被 tun0 回环截获。

技术上，SingBoxProvider 与 TunnelProvider 高度对称：差异在于 WireGuard 注入替换为 sing-box 配置写入 + 进程启动。nftables 规则需适配 tun0 接口名并新增 mgmt0 到代理服务器的出站白名单。受管镜像在构建时预装 sing-box 二进制，运行时由 Provider 写入配置。RoutingProvider 作为工厂层根据 `tunnel_type` 委托到对应实现。

**Primary recommendation:** 使用 sing-box v1.13.3 稳定版，TUN inbound (`auto_route: true`) + proxy outbound (`bind_interface: "mgmt0"`) + DNS hijack 路由规则的配置模式。配置以 Go 结构体序列化为 JSON，通过 `/proc/<pid>/root/` 写入容器文件系统。

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions

**Provider 工厂架构：**
- D-01: 创建 `RoutingProvider` 实现 `Provider` 接口，内部持有 `TunnelProvider` 和 `SingBoxProvider` 两个实例
- D-02: `RoutingProvider.PrepareHost` 读取 `HostNetworkSpec.Egress.TunnelType`，委托给对应实现；`CleanupHost` 同理
- D-03: `cmd/host-agent/main.go` 注入 `RoutingProvider`（替代直接注入 `TunnelProvider`），对外仍是单一 `Provider` 接口
- D-04: 现有 WireGuard 路径完全不受影响——`RoutingProvider` 在 `TunnelType == "wireguard"` 时直接委托给 `TunnelProvider`

**sing-box 配置生成：**
- D-05: 在 SingBoxProvider 内定义 Go 结构体表示 sing-box 配置模型
- D-06: `ProxySpec.OutboundConfig`（用户提供的 sing-box outbound JSON）直接嵌入 outbound 数组，不做二次解析
- D-07: 固定模板部分：inbound 为 tun 模式（`auto_route: true, strict_route: true`），DNS server 使用 `ProxySpec.DNSServer`，route 为默认全流量
- D-08: 配置序列化为 JSON 后通过 `/proc/<pid>/root/etc/sing-box/config.json` 写入容器文件系统

**sing-box 出网路径：**
- D-09: sing-box 在容器 netns 内运行，其 proxy outbound 通过管理 veth（mgmt0）→ 宿主机出网
- D-10: 容器 netns 内添加 host route：`<proxy_server_ip>/32 via <mgmt_gateway> dev mgmt0`
- D-11: sing-box outbound 配置使用 `bind_interface: "mgmt0"` 确保代理连接绑定到管理 veth
- D-12: 宿主机侧需为管理 veth 启用 IP 转发 + masquerade
- D-13: nftables 规则需为 proxy 模式扩展：允许 mgmt0 上 OUTPUT 到 proxy_server_ip:port 的连接

**sing-box 进程管理：**
- D-14: 使用 `nsenter -t <pid> -n -m -p` 进入容器的 net/mount/pid 命名空间，后台启动 sing-box
- D-15: sing-box 进程在容器 PID 命名空间内，容器停止时自动被终止
- D-16: `CleanupHost` 增加兜底清理：通过 nsenter 检查并 kill 残留的 sing-box 进程
- D-17: `PrepareHost` 启动 sing-box 后需等待 tun0 接口出现（轮询 `ip link show tun0`），超时则报错
- D-18: tun 接口使用固定名 `tun0`——每个容器有独立 netns，不会冲突

**受管镜像预装：**
- D-19: Dockerfile 中从 sing-box GitHub Release 下载 Linux amd64 预编译二进制
- D-20: 创建配置目录 `/etc/sing-box/`，运行时由 SingBoxProvider 写入配置文件
- D-21: sing-box 版本以 Dockerfile ARG 形式管理

**nftables 适配：**
- D-22: `ApplyFirewallRules` 需支持 proxy 模式变体：接受 tun0 ifindex + proxy server 信息
- D-23: proxy 模式 OUTPUT chain 增加规则：允许 mgmt0 到 proxy_server_ip:port 的 TCP/UDP 出站
- D-24: 其余规则保持一致：默认 DROP、lo 允许、IPv6 全部 DROP

**三重校验：**
- D-25: `VerifyNetworkIntegrity` 已在 Phase 7 适配 proxy DNS 路径，本阶段确保端到端验证通过
- D-26: 泄漏检测在 proxy 模式下同样生效

### Claude's Discretion

- sing-box 进程启动后 tun0 就绪的等待策略（轮询间隔、最大超时）
- sing-box 配置中 tun 的具体参数（inet4_address、mtu、stack 选择 system/gvisor/mixed）
- masquerade 规则的具体实现方式（iptables vs nftables on host side）
- sing-box 启动失败时的错误恢复和回滚策略
- sing-box 二进制的具体版本选择

### Deferred Ideas (OUT OF SCOPE)

None — discussion stayed within phase scope

</user_constraints>

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| SING-01 | 代理类型主机启动时，SingBoxProvider 在容器 netns 内启动 sing-box tun 模式进程 | sing-box TUN 配置模型、nsenter 进程管理、bind_interface 路由回避 |
| SING-02 | sing-box 配置确保 DNS 查询走代理路径，不使用系统 DNS | DNS hijack 路由规则、DNS server detour 配置 |
| SING-03 | nftables 规则允许 tun0 接口流量，保持默认拒绝策略 | ApplyFirewallRules 扩展方案、proxy server 白名单规则 |
| SING-04 | 三重校验对代理类型同样生效 | VerifyNetworkIntegrity 已支持 proxy DNS 路径（Phase 7 完成） |
| SING-05 | 受管用户镜像预装 sing-box 二进制 | sing-box v1.13.3 release asset、Dockerfile ARG 模式 |
| SING-06 | Provider 工厂根据 TunnelType 自动选择 TunnelProvider 或 SingBoxProvider | RoutingProvider 委托模式，对外仍是单一 Provider 接口 |
| SING-07 | 容器停止或重建时 sing-box 进程被正确清理 | PID namespace 自动终止 + CleanupHost 兜底 kill |

</phase_requirements>

## Standard Stack

### Core

| Library/Tool | Version | Purpose | Why Standard |
|-------------|---------|---------|--------------|
| sing-box | v1.13.3 | TUN 模式全流量代理进程 | 2026-03-15 发布的最新稳定版，支持 tun inbound + auto_route + strict_route + bind_interface，是代理协议全流量接管的标准方案 |
| Go `os/exec` | Go 1.25.7 | nsenter 调用、sing-box 进程管理 | 标准库，与项目现有 `exec.Command` 模式一致（verify.go、namespace.go） |
| Go `encoding/json` | Go 1.25.7 | sing-box 配置 JSON 序列化 | 标准库，用于将 Go 结构体序列化为 sing-box 配置文件 |
| `google/nftables` | v0.3.0 | 容器内防火墙规则（扩展支持 tun0 + proxy server 白名单） | 项目已有依赖，go.mod 已锁定版本 |
| `vishvananda/netlink` | v1.3.1 | 容器内接口探测（tun0 ifindex 解析） | 项目已有依赖，与 resolveContainerIfIndexes 模式一致 |
| `vishvananda/netns` | v0.0.5 | 容器网络命名空间操作 | 项目已有依赖，GetContainerNetNS/InjectManagementVeth 等均已使用 |

### Supporting

| Library/Tool | Version | Purpose | When to Use |
|-------------|---------|---------|-------------|
| `iptables` CLI | 宿主机稳定版 | 宿主机侧 masquerade 规则（代理流量 NAT） | 为 sing-box proxy 连接设置宿主机出网 NAT |
| `nsenter` CLI | 宿主机稳定版 | 进入容器命名空间启动/清理 sing-box 进程 | PrepareHost 启动 sing-box、CleanupHost 清理残留 |

### Alternatives Considered

| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| `iptables` CLI (宿主机 masquerade) | `google/nftables` Go 库 | nftables 更现代，但 Docker Engine 28.x 默认使用 iptables，混用可能导致规则冲突；容器内部可用 nftables（无 Docker 干扰），宿主机用 iptables 更安全 |
| nsenter CLI 启动 sing-box | Go 代码直接操作 namespace + exec | 直接操作更精细，但 nsenter 是标准做法，更易理解和调试；与 VerifyNetworkIntegrity 的 nsenter 模式一致 |
| TUN `system` stack | `mixed` (system TCP + gvisor UDP) 或 `gvisor` | `mixed` 是预编译二进制的默认值，对 UDP 处理更完善；`system` 更轻量，在受控容器环境中足够 |

**Version verification:**

- sing-box v1.13.3: 2026-03-15 发布，GitHub Releases 页面确认 (https://github.com/SagerNet/sing-box/releases/tag/v1.13.3)
- 其余依赖版本与 `go.mod` 一致，无需额外安装

## Architecture Patterns

### Recommended Project Structure

```
internal/network/
├── provider.go                     # Provider 接口（不变）
├── routing_provider_linux.go       # [新增] RoutingProvider — 按 TunnelType 委托
├── singbox_provider_linux.go       # [新增] SingBoxProvider — sing-box 配置生成 + 进程管理
├── singbox_config.go               # [新增] sing-box 配置结构体和序列化
├── tunnel_provider_linux.go        # TunnelProvider — WireGuard（不变）
├── firewall.go                     # ApplyFirewallRules（扩展签名）
├── firewall_proxy.go               # [新增] ApplyProxyFirewallRules — proxy 模式防火墙
├── host_forwarding_linux.go        # [新增] 宿主机 IP 转发 + masquerade
├── namespace.go                    # GetContainerNetNS, InjectManagementVeth（不变）
├── verify.go                       # VerifyNetworkIntegrity（不变，已支持 proxy）
├── dns.go                          # ConfigureContainerDNS（复用）
├── types.go                        # 类型定义（不变，Phase 7 已扩展）
├── validate.go                     # ValidateEgressBinding（不变，Phase 7 已扩展）
└── errors.go                       # NetworkError 类型（不变）
```

### Pattern 1: RoutingProvider 委托模式

**What:** RoutingProvider 实现 Provider 接口，根据 `EgressConfig.TunnelType` 将请求委托给 TunnelProvider 或 SingBoxProvider。

**When to use:** 需要在运行时根据配置选择不同的网络 Provider 实现。

**Example:**

```go
// Source: 基于项目现有 Provider 接口模式
type RoutingProvider struct {
    tunnel  *TunnelProvider
    singbox *SingBoxProvider
    logger  *slog.Logger
}

func NewRoutingProvider(logger *slog.Logger) *RoutingProvider {
    return &RoutingProvider{
        tunnel:  NewTunnelProvider(logger),
        singbox: NewSingBoxProvider(logger),
        logger:  logger,
    }
}

func (rp *RoutingProvider) PrepareHost(ctx context.Context, spec HostNetworkSpec) error {
    if spec.Egress == nil {
        return &NetworkError{Type: ErrBindingMissing, Message: "PrepareHost called without egress config", HostID: spec.HostID}
    }
    switch spec.Egress.TunnelType {
    case TunnelTypeProxy:
        return rp.singbox.PrepareHost(ctx, spec)
    default:
        return rp.tunnel.PrepareHost(ctx, spec)
    }
}

func (rp *RoutingProvider) CleanupHost(ctx context.Context, spec HostNetworkSpec) error {
    // CleanupHost 需同时清理两种类型的残留
    rp.tunnel.CleanupHost(ctx, spec)
    rp.singbox.CleanupHost(ctx, spec)
    return nil
}
```

### Pattern 2: sing-box 配置序列化

**What:** 用 Go 结构体建模 sing-box 完整配置，合并用户提供的 outbound JSON 和固定模板，序列化为 JSON 写入容器。

**When to use:** SingBoxProvider.PrepareHost 生成配置文件时。

**Example:**

```go
// Source: sing-box 官方文档 https://sing-box.sagernet.org/configuration/
type singBoxConfig struct {
    Log       singBoxLog        `json:"log"`
    DNS       singBoxDNS        `json:"dns"`
    Inbounds  []json.RawMessage `json:"inbounds"`
    Outbounds []json.RawMessage `json:"outbounds"`
    Route     singBoxRoute      `json:"route"`
}

type singBoxLog struct {
    Level string `json:"level"`
}

type singBoxDNS struct {
    Servers  []singBoxDNSServer `json:"servers"`
    Strategy string             `json:"strategy,omitempty"`
}

type singBoxDNSServer struct {
    Tag    string `json:"tag"`
    Type   string `json:"type"`
    Server string `json:"server"`
    Detour string `json:"detour,omitempty"`
}

type singBoxRoute struct {
    Rules            []singBoxRouteRule `json:"rules"`
    DefaultInterface string             `json:"default_interface,omitempty"`
}

type singBoxRouteRule struct {
    Action   string `json:"action,omitempty"`
    Protocol string `json:"protocol,omitempty"`
}
```

### Pattern 3: nsenter 进程管理

**What:** 使用 nsenter 进入容器的 net/mount/pid 命名空间后台启动 sing-box，利用 PID namespace 实现容器停止时自动清理。

**When to use:** SingBoxProvider.PrepareHost 启动 sing-box 进程。

**Example:**

```go
// Source: 基于 nsenter(1) 手册和项目现有 nsenter 模式 (verify.go)
func startSingBox(ctx context.Context, containerPID uint32) error {
    pidStr := strconv.FormatUint(uint64(containerPID), 10)
    cmd := exec.CommandContext(ctx,
        "nsenter", "-t", pidStr, "-n", "-m", "-p", "--",
        "/usr/local/bin/sing-box", "run", "-c", "/etc/sing-box/config.json",
    )
    if err := cmd.Start(); err != nil {
        return fmt.Errorf("start sing-box: %w", err)
    }
    // 不调用 cmd.Wait()——进程在容器 PID 命名空间内后台运行
    // 当容器停止时，PID namespace 内的所有进程自动被 SIGKILL
    go cmd.Wait() // 防止僵尸进程
    return nil
}
```

### Pattern 4: tun0 就绪轮询

**What:** 启动 sing-box 后，轮询检查 tun0 接口是否出现在容器 netns 中，确认 TUN 设备创建成功。

**When to use:** sing-box 进程启动后、应用防火墙规则前。

**Example:**

```go
func waitForTun0(ctx context.Context, containerPID uint32, timeout time.Duration) error {
    pidStr := strconv.FormatUint(uint64(containerPID), 10)
    deadline := time.After(timeout)
    ticker := time.NewTicker(200 * time.Millisecond)
    defer ticker.Stop()

    for {
        select {
        case <-deadline:
            return fmt.Errorf("tun0 not ready within %s", timeout)
        case <-ctx.Done():
            return ctx.Err()
        case <-ticker.C:
            cmd := exec.CommandContext(ctx, "nsenter", "-t", pidStr, "-n", "--",
                "ip", "link", "show", "tun0")
            if cmd.Run() == nil {
                return nil
            }
        }
    }
}
```

### Anti-Patterns to Avoid

- **在容器外运行 sing-box 然后移入 tun 接口：** sing-box 是用户态进程（不是内核接口），无法像 WireGuard 那样用 birthplace namespace 模式。必须在容器 netns 内启动。
- **使用 `auto_redirect` + 自定义 nftables：** `auto_redirect` 会让 sing-box 自行创建 nftables 规则，与我们的 `ApplyFirewallRules` 冲突。只用 `auto_route`，防火墙由我们全权控制。
- **依赖 sing-box 的 `route.auto_detect_interface`：** 容器用 `--network=none` 启动，没有默认网络接口可以自动检测。必须在 outbound 上显式设置 `bind_interface: "mgmt0"`。
- **写入 resolv.conf 但不配置 sing-box DNS hijack：** 仅写 resolv.conf 不够——sing-box 需要通过 route 规则中的 `hijack-dns` action 拦截 DNS 包并走代理出口，否则 DNS 查询虽然发往正确地址但仍走默认路由（被 nftables DROP）。

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| TUN 全流量接管 | 自己写 tun 设备 + 路由规则 | sing-box `auto_route: true` | sing-box 处理了 iproute2 策略路由表创建、规则优先级、内核路由重定向等复杂细节 |
| DNS 查询拦截 | 自己写 DNS 拦截代理 | sing-box `hijack-dns` route rule | sing-box 在 TUN 栈层面拦截 DNS 协议包并路由到配置的 DNS server |
| 代理协议实现 | 自己实现 SOCKS5/VMess/SS 客户端 | sing-box outbound | sing-box 原生支持所有五种白名单协议，用户提供的 outbound JSON 直接嵌入 |
| 进程命名空间隔离 | 自己用 Go `syscall.Setns` 切换 | `nsenter` CLI | nsenter 是标准工具，与项目现有的 verify.go 模式一致，错误处理更清晰 |

## Common Pitfalls

### Pitfall 1: 代理连接被 tun0 回环截获

**What goes wrong:** sing-box 的代理 outbound 连接发出后，被自己创建的 tun0 路由规则捕获，形成死循环——连接无法到达代理服务器。

**Why it happens:** `auto_route: true` 在容器 netns 内设置默认路由指向 tun0（iproute2 table 2022），所有出站流量（包括 sing-box 自己发出的代理连接）都会被重定向到 tun0。

**How to avoid:**
1. 在 outbound 配置中设置 `bind_interface: "mgmt0"`，使代理连接绑定到管理 veth
2. 在容器 netns 内添加 host route：`<proxy_server_ip>/32 via <mgmt_gateway> dev mgmt0`，确保到代理服务器的路由走 mgmt0
3. 确保 host route 在 sing-box 启动前添加，否则 sing-box 的路由规则可能先于 host route 生效

**Warning signs:** sing-box 日志报 connection timeout 或 connection refused，但代理服务器实际可达。

### Pitfall 2: 宿主机缺少 IP 转发和 NAT 规则

**What goes wrong:** sing-box 代理连接通过 mgmt0 到达宿主机侧 mgmt-\<id\> 后，宿主机不做转发，包被丢弃。或者转发了但源 IP 是私有地址 10.99.x.y，对端无法回包。

**Why it happens:** WireGuard 模式不需要宿主机侧转发（birthplace namespace 机制），所以现有代码没有设置。Proxy 模式下代理连接必须经宿主机路由出网。

**How to avoid:**
1. `sysctl -w net.ipv4.ip_forward=1`（PrepareHost 时检查/启用）
2. 添加 masquerade 规则：`iptables -t nat -A POSTROUTING -s 10.99.0.0/16 ! -o mgmt-+ -j MASQUERADE`（或等价 nftables 规则）
3. masquerade 规则是全局的，只需设置一次（首次 proxy 类型主机启动时）

**Warning signs:** `tcpdump -i mgmt-<id>` 能看到包到达宿主机侧但没有转发，或 `dmesg` 中有 martian source 警告。

### Pitfall 3: CleanupHost 未覆盖 proxy 模式

**What goes wrong:** 容器重建时 `CleanupHost` 只清理 WireGuard 接口和管理 veth，没有 kill 残留的 sing-box 进程。如果容器 PID namespace 未完全销毁（某些边缘情况），sing-box 可能成为僵尸进程。

**Why it happens:** 现有 `TunnelProvider.CleanupHost` 只处理 `wg-<id>` 和 `mgmt-<id>` 接口。sing-box 是用户态进程，不是网络接口。

**How to avoid:** `SingBoxProvider.CleanupHost` 通过 nsenter 检查容器 PID namespace 中是否有 sing-box 进程残留，如有则 kill。即使通常情况下 docker stop 会清理 PID namespace 中的所有进程，兜底检查也是必要的。

**Warning signs:** `docker ps` 显示容器已停止，但 `ps aux | grep sing-box` 在宿主机上仍可见相关进程。

### Pitfall 4: nftables 规则冲突与顺序问题

**What goes wrong:** proxy 模式的 nftables 规则 OUTPUT chain 中，mgmt0 到代理服务器的白名单规则必须在默认 DROP 策略之前生效。如果规则插入顺序错误，代理连接被 DROP。

**Why it happens:** nftables chain 的 policy 是 DROP，规则按添加顺序匹配（first match wins）。如果 DROP 规则在 ACCEPT 规则之前，ACCEPT 永远不会被执行。但由于 policy 是 chain 级别的 DROP，所有 ACCEPT 规则都先于 policy 判定，所以实际风险是规则遗漏（忘记添加 proxy server 白名单规则），而非顺序问题。

**How to avoid:** 使用独立函数 `ApplyProxyFirewallRules` 处理 proxy 模式，确保完整的规则集：lo ACCEPT、tun0 ACCEPT、mgmt0 → proxy_server ACCEPT（新增）、mgmt0 established/related ACCEPT、policy DROP。

**Warning signs:** sing-box 日志报连接被拒绝，`nsenter ... nft list ruleset` 显示缺少预期的 ACCEPT 规则。

### Pitfall 5: sing-box DNS 路径不完整导致泄漏

**What goes wrong:** resolv.conf 指向代理 DNS 服务器，但 DNS 查询实际绕过了代理——用户进程直接发出 UDP:53 包，被 tun0 捕获后 sing-box 用本地直连方式解析（不走代理 outbound）。

**Why it happens:** sing-box DNS 模块的 server 配置没有设置 `detour: "proxy-out"`，或者 route 规则缺少 `hijack-dns` action，导致 DNS 查询走了 direct outbound。

**How to avoid:**
1. Route rules 必须包含 `{"protocol": "dns", "action": "hijack-dns"}`，将所有 DNS 协议流量拦截到 sing-box DNS 模块
2. DNS server 配置必须设置 `detour: "proxy-out"`，确保 DNS 解析走代理 outbound
3. VerifyNetworkIntegrity 的 DNS 检查会捕获这个问题

**Warning signs:** VerifyNetworkIntegrity 的 DNS 检查失败，或 `nsenter ... dig` 测试显示 DNS 解析走了非预期路径。

## Code Examples

### 完整 sing-box 配置模板

```json
// Source: sing-box 官方文档 https://sing-box.sagernet.org/manual/proxy/client/
// + 项目特定配置（bind_interface、DNS detour）
{
  "log": {
    "level": "warn"
  },
  "dns": {
    "servers": [
      {
        "tag": "proxy-dns",
        "type": "udp",
        "server": "<ProxySpec.DNSServer>",
        "detour": "proxy-out"
      }
    ],
    "strategy": "ipv4_only"
  },
  "inbounds": [
    {
      "type": "tun",
      "tag": "tun-in",
      "interface_name": "tun0",
      "address": ["172.18.0.1/30"],
      "mtu": 1500,
      "auto_route": true,
      "strict_route": true,
      "stack": "system"
    }
  ],
  "outbounds": [
    {
      "<...ProxySpec.OutboundConfig 字段展开...>",
      "tag": "proxy-out",
      "bind_interface": "mgmt0"
    },
    {
      "type": "direct",
      "tag": "direct",
      "bind_interface": "mgmt0"
    }
  ],
  "route": {
    "rules": [
      {"action": "sniff"},
      {"protocol": "dns", "action": "hijack-dns"}
    ],
    "default_interface": "mgmt0"
  }
}
```

**关键配置说明：**

| 配置项 | 值 | 原因 |
|--------|---|------|
| `inbound.auto_route` | `true` | 自动创建 iproute2 策略路由，将所有流量导向 tun0 |
| `inbound.strict_route` | `true` | 增强路由规则严格性，减少绕过可能 |
| `inbound.stack` | `"system"` | 使用系统网络栈做 L3→L4 转换，容器环境下最简单高效 |
| `inbound.address` | `["172.18.0.1/30"]` | tun0 接口 IP，/30 子网足够（只有 tun 网关和本地地址） |
| `inbound.mtu` | `1500` | 标准 MTU，与 mgmt0 veth 一致 |
| `outbound.bind_interface` | `"mgmt0"` | 代理连接绑定到管理 veth，避免被 tun0 回环截获 |
| `dns.servers[0].detour` | `"proxy-out"` | DNS 查询走代理 outbound，防止 DNS 泄漏 |
| `route.rules: hijack-dns` | — | 拦截所有 DNS 协议流量到 sing-box DNS 模块 |
| `route.default_interface` | `"mgmt0"` | 全局默认出口接口，作为 bind_interface 的兜底 |

### Outbound JSON 合并

```go
// 用户提供的 outbound config（存储在 ProxySpec.OutboundConfig）
// 需要注入 tag 和 bind_interface 字段
func buildOutbound(userConfig json.RawMessage) (json.RawMessage, error) {
    var m map[string]any
    if err := json.Unmarshal(userConfig, &m); err != nil {
        return nil, fmt.Errorf("parse outbound config: %w", err)
    }
    m["tag"] = "proxy-out"
    m["bind_interface"] = "mgmt0"
    return json.Marshal(m)
}
```

### 从 outbound config 提取代理服务器地址

```go
// 需要从用户 outbound config 中提取 server 和 server_port
// 用于设置 host route 和 nftables 白名单规则
type proxyServerAddr struct {
    Server     string `json:"server"`
    ServerPort int    `json:"server_port"`
}

func extractProxyServer(outboundConfig json.RawMessage) (proxyServerAddr, error) {
    var addr proxyServerAddr
    if err := json.Unmarshal(outboundConfig, &addr); err != nil {
        return proxyServerAddr{}, fmt.Errorf("extract proxy server: %w", err)
    }
    if addr.Server == "" || addr.ServerPort == 0 {
        return proxyServerAddr{}, fmt.Errorf("proxy server address incomplete")
    }
    return addr, nil
}
```

### Host route 添加

```go
// 在容器 netns 内添加 host route: proxy_server_ip/32 via mgmt_gateway dev mgmt0
func addProxyServerRoute(containerNS, hostNS netns.NsHandle, proxyServerIP, mgmtGateway string) error {
    runtime.LockOSThread()
    defer runtime.UnlockOSThread()

    if err := netns.Set(containerNS); err != nil {
        return err
    }
    defer netns.Set(hostNS)

    mgmtLink, err := netlink.LinkByName("mgmt0")
    if err != nil {
        return fmt.Errorf("find mgmt0: %w", err)
    }

    dst := &net.IPNet{IP: net.ParseIP(proxyServerIP), Mask: net.CIDRMask(32, 32)}
    gw := net.ParseIP(mgmtGateway)
    route := &netlink.Route{
        LinkIndex: mgmtLink.Attrs().Index,
        Dst:       dst,
        Gw:        gw,
    }
    return netlink.RouteAdd(route)
}
```

### Proxy 模式 nftables 规则

```go
// Source: 基于现有 firewall.go 的 applyIPv4Rules 模式
// proxy 模式与 WireGuard 模式的差异：
// 1. tun0 ifindex 替代 wg ifindex
// 2. OUTPUT chain 新增：mgmt0 → proxy_server_ip:port ACCEPT（TCP + UDP）
// 3. 其余规则不变

func ApplyProxyFirewallRules(containerNS netns.NsHandle, tunIfIndex, loIfIndex, mgmtIfIndex int,
    proxyServerIP net.IP, proxyServerPort uint16) error {
    // ... 与 ApplyFirewallRules 结构相同 ...
    // OUTPUT chain:
    //   lo -> ACCEPT
    //   tun0 -> ACCEPT (替代 wg)
    //   mgmt0 + dst=proxyServerIP + dport=proxyServerPort -> ACCEPT (新增)
    //   mgmt0 + ct state established,related -> ACCEPT
    //   policy DROP
    // INPUT chain:
    //   lo -> ACCEPT
    //   tun0 -> ACCEPT (替代 wg)
    //   mgmt0 + tcp dport 22 -> ACCEPT
    //   mgmt0 + ct state established,related -> ACCEPT
    //   policy DROP
}
```

### 宿主机 masquerade 设置

```go
// 使用 iptables CLI 设置 masquerade（避免与 Docker 的 iptables 规则冲突）
func ensureHostMasquerade(ctx context.Context) error {
    // 检查规则是否已存在
    check := exec.CommandContext(ctx, "iptables", "-t", "nat", "-C", "POSTROUTING",
        "-s", "10.99.0.0/16", "!", "-o", "mgmt-+", "-j", "MASQUERADE")
    if check.Run() == nil {
        return nil // 规则已存在
    }
    // 添加规则
    add := exec.CommandContext(ctx, "iptables", "-t", "nat", "-A", "POSTROUTING",
        "-s", "10.99.0.0/16", "!", "-o", "mgmt-+", "-j", "MASQUERADE")
    if out, err := add.CombinedOutput(); err != nil {
        return fmt.Errorf("add masquerade rule: %w (%s)", err, strings.TrimSpace(string(out)))
    }
    return nil
}

func ensureIPForwarding(ctx context.Context) error {
    cmd := exec.CommandContext(ctx, "sysctl", "-w", "net.ipv4.ip_forward=1")
    if out, err := cmd.CombinedOutput(); err != nil {
        return fmt.Errorf("enable ip forwarding: %w (%s)", err, strings.TrimSpace(string(out)))
    }
    return nil
}
```

## SingBoxProvider PrepareHost 流水线

SingBoxProvider 的 `PrepareHost` 与 TunnelProvider 高度对称。完整步骤：

| Step | 操作 | 对应 TunnelProvider 步骤 | 复用/新增 |
|------|------|--------------------------|----------|
| 1 | Egress 非空校验 | 同 | 复用逻辑 |
| 2 | 校验 Proxy config 非空 | 校验 Tunnel config 非空 | 新增 |
| 3 | 获取容器 netns + PID | 同 | 复用 `GetContainerNetNS` |
| 4 | 注入管理 veth | 同 | 复用 `InjectManagementVeth` |
| 5 | 从 outbound config 提取 proxy server 地址 | — | 新增 |
| 6 | 宿主机启用 IP 转发 + masquerade | — | 新增 |
| 7 | 容器 netns 内添加 proxy server host route | — | 新增 |
| 8 | 生成 sing-box 配置 JSON | — | 新增 |
| 9 | 写入容器 `/etc/sing-box/config.json` | — | 新增 |
| 10 | 通过 nsenter 启动 sing-box 进程 | InjectWireGuard | 新增 |
| 11 | 等待 tun0 接口就绪 | — | 新增 |
| 12 | 配置容器 DNS (resolv.conf) | 同 | 复用 `ConfigureContainerDNS` |
| 13 | 解析 tun0/lo/mgmt0 ifindex | 解析 wg/lo/mgmt0 ifindex | 适配 |
| 14 | 应用 nftables 防火墙规则 | 同（但规则不同） | 新增 proxy 变体 |
| 15 | 三重校验 | 同 | 复用 `VerifyNetworkIntegrity` |

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| sing-box `inet4_address` | sing-box `address` 字段 | v1.10.0 | `inet4_address` 已在 v1.12.0 移除，必须用 `address` |
| sing-box `gso` option | 已废弃 | v1.11.0 | GSO 对透明代理无优势，已在 v1.12.0 移除 |
| sing-box legacy DNS server format | `type`-based DNS server | v1.12.0 | 新版使用 `"type": "udp"` 而非 `"address": "udp://..."` |
| `domain_strategy` in outbound | `domain_resolver` in outbound | v1.12.0 | `domain_strategy` 将在 v1.14.0 移除 |

**Deprecated/outdated:**
- `inet4_address` / `inet6_address`: 已在 v1.12.0 移除，用 `address` 数组替代
- `gso`: 已在 v1.12.0 移除
- `domain_strategy` on outbound: 将在 v1.14.0 移除，迁移到 `domain_resolver`

## Open Questions

1. **proxy server 域名解析**
   - What we know: outbound config 中的 `server` 可能是域名而非 IP；host route 需要 IP
   - What's unclear: 是否需要在配置阶段提前解析域名
   - Recommendation: Phase 7 的 proxy_config 校验已验证 `server` 字段非空。如果 server 是域名，在 SingBoxProvider 中使用 `net.ResolveIPAddr` 解析后设置 host route。sing-box outbound 本身也支持域名，但 host route 必须使用 IP。

2. **masquerade 规则持久性**
   - What we know: `iptables` 规则重启后丢失；`ensureHostMasquerade` 每次 PrepareHost 检查
   - What's unclear: 是否需要持久化到 iptables-save 或 systemd 服务
   - Recommendation: v1.1 阶段用 PrepareHost 幂等检查即可（每次启动时检查/添加），不做持久化。宿主机重启后第一次启动 proxy 类型主机时会自动重建规则。

3. **tun0 地址选择**
   - What we know: 需要一个不与 mgmt veth 子网（10.99.0.0/16）冲突的地址
   - What's unclear: 172.18.0.1/30 是否与宿主机现有 Docker 网络冲突
   - Recommendation: 使用 `172.18.0.1/30`。Docker 默认 bridge 网络使用 172.17.0.0/16。如果 Docker 网络冲突，可改为 `172.19.0.1/30` 或 `198.18.0.1/30`（RFC 5737 测试用）。由于容器用 `--network=none`，不存在 Docker bridge 冲突问题。

## Project Constraints (from .cursor/rules/)

项目约定和架构约束已在 CLAUDE.md 中定义：

- **网络安全强约束：** 必须通过 tun 风格的全局隧道路由实现全流量强制出网，不能允许直连外网
- **IP 分配：** 每个容器必须绑定出口 IP，无绑定视为非法
- **单宿主机优先：** v1 不引入多节点复杂度
- **Go 标准库优先：** 减少不必要的框架依赖
- **nftables 或 iptables：** 做默认拒绝的出站策略
- **中文沟通：** 所有面向用户的内容使用中文

## Validation Architecture

### Test Framework

| Property | Value |
|----------|-------|
| Framework | Go `testing` (标准库) |
| Config file | 无独立配置文件 |
| Quick run command | `go test ./internal/network/... -run TestSingBox -count=1 -v` |
| Full suite command | `go test ./internal/network/... -count=1 -v` |

### Phase Requirements → Test Map

| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| SING-01 | SingBoxProvider.PrepareHost 启动 sing-box tun 模式 | integration (需 root + netns) | 需实际容器环境 | ❌ Wave 0 |
| SING-02 | sing-box 配置生成包含正确的 DNS hijack 规则和 detour | unit | `go test ./internal/network/... -run TestSingBoxConfig -v` | ❌ Wave 0 |
| SING-03 | proxy 模式 nftables 规则包含 tun0 ACCEPT 和 proxy server 白名单 | unit (mock nftables conn) | `go test ./internal/network/... -run TestProxyFirewall -v` | ❌ Wave 0 |
| SING-04 | VerifyNetworkIntegrity 对 proxy 类型生效 | existing | `go test ./internal/network/... -run TestVerify -v` | ✅ verify_test.go |
| SING-05 | Dockerfile 包含 sing-box 二进制 | build smoke | `docker build -f deploy/docker/managed-user/Dockerfile .` | ❌ Wave 0 |
| SING-06 | RoutingProvider 按 TunnelType 路由到正确实现 | unit | `go test ./internal/network/... -run TestRoutingProvider -v` | ❌ Wave 0 |
| SING-07 | CleanupHost 清理 sing-box 残留进程 | unit | `go test ./internal/network/... -run TestSingBoxCleanup -v` | ❌ Wave 0 |

### Sampling Rate

- **Per task commit:** `go test ./internal/network/... -count=1 -v`
- **Per wave merge:** `go test ./... -count=1`
- **Phase gate:** Full suite green before `/gsd-verify-work`

### Wave 0 Gaps

- [ ] `internal/network/singbox_config_test.go` — covers SING-02 (配置生成正确性)
- [ ] `internal/network/routing_provider_test.go` — covers SING-06 (路由委托正确性)
- [ ] Dockerfile build smoke test — covers SING-05

## Sources

### Primary (HIGH confidence)

- [sing-box 官方文档 - TUN inbound](https://sing-box.sagernet.org/configuration/inbound/tun/) — tun 配置参数、auto_route、strict_route 行为
- [sing-box 官方文档 - Dial Fields](https://sing-box.sagernet.org/configuration/shared/dial/) — bind_interface、detour 配置
- [sing-box 官方文档 - Client Manual](https://sing-box.sagernet.org/manual/proxy/client/) — 完整 TUN 代理配置示例
- [sing-box 官方文档 - DNS](https://sing-box.sagernet.org/configuration/dns/) — DNS server 配置结构
- [sing-box 官方文档 - DNS Server](https://sing-box.sagernet.org/configuration/dns/server/) — DNS server type 定义
- [sing-box 官方文档 - Route](https://sing-box.sagernet.org/configuration/route/) — auto_detect_interface、default_interface
- [sing-box GitHub Release v1.13.3](https://github.com/SagerNet/sing-box/releases/tag/v1.13.3) — 确认 v1.13.3 为当前稳定版，release asset 命名 `sing-box-1.13.3-linux-amd64.tar.gz`
- [nsenter(1) Linux manual](http://man7.org/linux/man-pages/man1/nsenter.1.html) — nsenter 命名空间操作

### Secondary (MEDIUM confidence)

- [sing-box GitHub Issue #3440](https://github.com/SagerNet/sing-box/issues/3440) — bind_interface 在特定场景下的问题报告，确认 bind_interface 是正确的路由回避方式
- [sing-box GitHub Issue #3705](https://github.com/SagerNet/sing-box/issues/3705) — auto_redirect 与 DNS hijack 的交互问题，佐证不使用 auto_redirect 的决策

### Tertiary (LOW confidence)

- 无低置信度来源

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH — sing-box v1.13.3 版本已在官方 GitHub 确认，所有配置参数已在官方文档验证
- Architecture: HIGH — SingBoxProvider 与 TunnelProvider 高度对称，代码模式直接参考现有实现
- Pitfalls: HIGH — DNS 泄漏和路由回环问题已在官方文档和 GitHub Issue 中有明确描述和解决方案
- Process management: MEDIUM — nsenter 后台进程 + PID namespace 自动清理是标准做法，但缺少 Go 进程管理的具体边缘情况测试

**Research date:** 2026-03-28
**Valid until:** 2026-04-28（sing-box 稳定版更新周期约 1-2 月）
