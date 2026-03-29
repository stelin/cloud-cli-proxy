# Phase 2: 隧道出网强制层 - Research

**Researched:** 2026-03-27
**Domain:** Linux 网络命名空间 / WireGuard 隧道 / nftables 防火墙 / DNS 隔离
**Confidence:** HIGH

## Summary

Phase 2 是整个产品承诺的核心层：保证每个用户容器只能通过指定出口 IP 出网，不允许任何旁路。技术实现路径非常明确——使用 `--network=none` 创建完全隔离的 Docker 容器，在容器启动后通过 Linux netns 机制注入 WireGuard 隧道接口和管理 veth 对，再用 nftables 在容器命名空间内实施默认拒绝策略。

Go 生态中已有成熟且活跃维护的库可以完全编程化地完成上述操作：`vishvananda/netns` 做命名空间切换、`vishvananda/netlink` 做接口/路由/地址管理、`wgctrl` 做 WireGuard 配置、`google/nftables` 做防火墙规则。这四个库覆盖了 Phase 2 所有网络操作需求，不需要在生产路径中 shell out 到 `ip` 或 `wg` 命令。

最大的风险不是"能不能做到"，而是"有没有堵住所有缝隙"。DNS 通过 nscd/systemd-resolved 绕过 namespace 隔离、Go goroutine 调度器在 OS 线程间迁移导致 namespace 操作串台、容器 PID 竞态——这些都是容易被忽视但会直接破坏出网约束的坑。

**Primary recommendation:** 对每个用户容器，用 `--network=none` 启动后注入 WireGuard + 管理 veth，在容器 netns 内用 nftables 实施默认拒绝，并在标记就绪前必须通过出口 IP / DNS / 泄漏三重校验。

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions
- **D-01:** v1 运行时采用"单主机单活跃出口 IP"语义；每台主机在任意时刻只允许一个生效出口 IP。
- **D-02:** 同一个出口 IP 可以在后台被分配给多个账号或主机，并允许这些主机同时运行、同时共用该出口 IP；唯一性不在 IP 资源层，而在"每台主机运行时只认一个活跃出口"这一层。
- **D-03:** 每台主机都绑定一个独立的 `claude` 账号，出口 IP 漂移会触发账号风控，因此禁止运行时隐式换 IP、自动漂移或回落到其他出口。
- **D-04:** 启动前若缺少出口 IP 绑定、隧道拉起失败、出口 IP 校验失败、DNS 校验失败或泄漏阻断失败，任务必须立即失败，不做自动重试。
- **D-05:** Phase 2 不允许为同一主机自动切换到其他出口 IP，也不允许在失败后自动执行"更重"的恢复动作；恢复必须依赖显式运维或后续任务重试。
- **D-06:** 网络准备与强校验放在真正 `start` / 进入可运行态之前执行，不在单纯 `create` 阶段提前把网络视为就绪。
- **D-07:** DNS 必须跟着当前活跃出口 IP 的受控隧道一起走，不能使用宿主机解析器、Docker 默认解析器或任何与该出口路径分离的解析方案。
- **D-08:** "出口 IP 路径"和"DNS 路径"视为同一个网络契约的一部分；如果两者之一不符合绑定结果，就视为整台主机网络未就绪。
- **D-09:** 主机在被标记为可用前，至少必须通过三类强校验：出口 IP 与绑定结果一致、DNS 解析走受控路径、非隧道直连出站被默认阻断。
- **D-10:** 运行中的主机一旦发现当前出口 IP 不可达、检测结果变化、DNS 脱离受控路径，或出现任何未配置出口的旁路，这台主机必须立刻标记为失败/异常。
- **D-11:** "除了已配置出口外不允许存在其他任何出口路径"是本阶段最高优先级约束。
- **D-12:** 任务和事件记录必须按网络失败类型细分（绑定缺失、出口 IP 不匹配、DNS 不走受控路径、非隧道直连阻断失效、出口 IP 当前不可达）。
- **D-13:** Phase 2 先按失败类型细分记录，不额外引入更复杂的危急等级模型。

### Claude's Discretion
- 具体采用 WireGuard 原生接线、辅助守护进程还是其他兼容实现，只要满足"受控隧道 + netns + 默认拒绝"的项目硬约束即可。
- 具体使用 nftables 还是 iptables，以及规则组织形式，只要遵守 Docker 官方规则链模型并保持默认拒绝即可。
- 出口校验、DNS 校验和泄漏阻断测试的具体命令、探测端点和重用脚本结构。

### Deferred Ideas (OUT OF SCOPE)
None — discussion stayed within phase scope.
</user_constraints>

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| NET-01 | 每个可运行容器都必须至少绑定一个出口 IP 资源；如果未绑定，启动必须失败 | 数据模型扩展 + 启动前绑定校验逻辑；使用已有 `host_egress_bindings` 表和 `ListHostBindings` 查询 |
| NET-02 | 用户容器中的所有出站流量都必须被强制导向指定的全隧道路由路径 | WireGuard + netns 全隧道模型：`--network=none` → 注入 wg0 → 默认路由指向 wg0 |
| NET-03 | 用户容器中的 DNS 解析也必须走受控路径，不能回落到宿主机或 Docker 默认解析器 | 命名空间级 `resolv.conf` 配置，指向隧道侧 DNS 服务器 |
| NET-04 | 对用户容器而言，任何未走隧道的出站流量都必须被默认阻断 | nftables 在容器 netns 内实施 OUTPUT chain 默认拒绝 + 显式白名单 |
| NET-05 | 在主机被标记为可接入前，系统会验证出口 IP 和 DNS 路径都符合预期 | 三重校验：出口 IP 匹配、DNS 路径正确、直连泄漏被阻断 |
</phase_requirements>

## Standard Stack

### Core

| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| `vishvananda/netns` | v0.0.5 | Go 中进行 Linux 网络命名空间切换和获取 | 社区标准库，containerd 等主流项目依赖，API 极简且稳定 |
| `vishvananda/netlink` | v1.3.1 | 通过 netlink 协议管理网络接口、IP 地址、路由 | 模仿 iproute2 的 Go 实现，覆盖 LinkAdd/LinkSetNsFd/RouteAdd 等全部所需操作 |
| `golang.zx2c4.com/wireguard/wgctrl` | v0.0.0-20241231 | 编程化配置 WireGuard 设备（密钥、端点、对等体） | WireGuard 官方 Go 控制库，支持内核态和用户态设备 |
| `google/nftables` | v0.3.0 | 编程化管理 nftables 表、链和规则 | Google 维护的纯 Go nftables 库，不依赖 libnftnl |

### Supporting

| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| `golang.org/x/sys/unix` | latest | 底层系统调用（CLONE_NEWNET, setns, nsenter） | netns/netlink 的间接依赖，也用于直接调 unshare/setns |
| Go 标准库 `os/exec` | Go 1.25+ | 在特定 netns 内执行校验命令（curl/dig） | 校验步骤需要在容器 netns 内运行探测命令时使用 |
| Go 标准库 `net` | Go 1.25+ | TCP/UDP 连通性探测 | 泄漏测试中尝试直连外部地址以验证是否被阻断 |

### Alternatives Considered

| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| `google/nftables` | `os/exec` 调 `nft` 命令 | 更简单但依赖外部二进制，解析输出脆弱，不如纯 Go 库可靠 |
| `google/nftables` | `os/exec` 调 `iptables` | iptables 在现代发行版中可能是 nft 的兼容层；如果宿主机确定只有 iptables 也可接受 |
| `vishvananda/netlink` | `os/exec` 调 `ip` 命令 | 同上理由——shell out 不如编程接口健壮 |
| 纯 Go 网络操作 | Shell 脚本 wrapper | 适合原型验证但不适合生产路径，错误处理和状态管理不可控 |

**Installation:**
```bash
go get github.com/vishvananda/netns@v0.0.5
go get github.com/vishvananda/netlink@v1.3.1
go get golang.zx2c4.com/wireguard/wgctrl@latest
go get github.com/google/nftables@v0.3.0
```

## Architecture Patterns

### Recommended Project Structure
```
internal/
├── network/
│   ├── provider.go          # Provider 接口（已有）
│   ├── wireguard.go          # WireGuard 隧道管理器
│   ├── namespace.go          # netns 创建、切换、接口注入
│   ├── firewall.go           # nftables 默认拒绝策略
│   ├── dns.go                # 容器 DNS 配置
│   ├── verify.go             # 出口 IP / DNS / 泄漏三重校验
│   ├── types.go              # EgressConfig, TunnelSpec 等共享类型
│   └── errors.go             # 网络失败类型细分（D-12 要求）
├── store/
│   └── migrations/
│       └── 0002_egress_tunnel.sql  # 扩展 egress_ips 表、新增 tunnel_configs
```

### Pattern 1: 容器网络接线流程（核心流程）

**What:** 在容器启动后、标记就绪前，完成完整的网络隔离和隧道接入。

**When to use:** 每次 `startHost` 或 `rebuildHost` 时。

**流程：**
```
┌─────────────────────────────────────────────────────────┐
│ 1. 启动前校验                                            │
│    ├── 查询 host_egress_bindings（已有 ListHostBindings）  │
│    ├── 绑定为空 → 立即失败（NET-01, D-04）                 │
│    └── 加载绑定的 egress_ip 详情 + 隧道配置                 │
├─────────────────────────────────────────────────────────┤
│ 2. 容器创建 / 启动                                       │
│    ├── docker create --network=none（完全网络隔离）         │
│    └── docker start（容器运行，只有 lo 接口）               │
├─────────────────────────────────────────────────────────┤
│ 3. 获取容器 PID 和 netns                                  │
│    ├── docker inspect → State.Pid                        │
│    └── /proc/<pid>/ns/net → 目标命名空间 fd               │
├─────────────────────────────────────────────────────────┤
│ 4. 注入管理 veth 对（SSH 管理路径）                         │
│    ├── 创建 veth pair: mgmt0 ↔ mgmt1                    │
│    ├── mgmt1 移入容器 netns                               │
│    ├── mgmt0 留在宿主机，挂到管理网段                       │
│    └── 配 IP：宿主机侧 10.99.0.1/30, 容器侧 10.99.0.2/30  │
├─────────────────────────────────────────────────────────┤
│ 5. 注入 WireGuard 隧道接口                                │
│    ├── 在宿主机 netns 创建 wg-<hostID> 接口               │
│    ├── 配置 WireGuard：密钥、端点、AllowedIPs              │
│    ├── 移入容器 netns（UDP socket 留在宿主机）              │
│    ├── 在容器内配 IP 和默认路由 → wg 接口                   │
│    └── wg 接口 up                                        │
├─────────────────────────────────────────────────────────┤
│ 6. 配置 DNS                                              │
│    └── 写 /etc/resolv.conf（容器内）→ 隧道侧 DNS 地址      │
├─────────────────────────────────────────────────────────┤
│ 7. 应用防火墙规则                                         │
│    ├── nftables OUTPUT chain: 默认 drop                  │
│    ├── 允许 wg 接口的所有出站                              │
│    ├── 允许 lo 回环                                       │
│    ├── 允许管理 veth 上的 established/related 入站          │
│    └── 拒绝其他一切                                       │
├─────────────────────────────────────────────────────────┤
│ 8. 三重校验（D-09 要求）                                   │
│    ├── 出口 IP 校验：curl api.ipify.org == 绑定的出口 IP   │
│    ├── DNS 校验：dig 查询 → 确认 DNS 服务器是隧道侧的       │
│    └── 泄漏测试：直连外部 IP（绕过 wg）→ 必须被阻断          │
├─────────────────────────────────────────────────────────┤
│ 9. 标记就绪 / 失败                                        │
│    ├── 全部通过 → 更新 host status = ready                │
│    └── 任一失败 → 立即失败 + 按类型记录事件（D-04, D-12）    │
└─────────────────────────────────────────────────────────┘
```

### Pattern 2: WireGuard Namespace Wiring（WireGuard 官方推荐模型）

**What:** WireGuard 接口在宿主机 netns 中创建，移入容器 netns 后，其加密 UDP socket 仍留在宿主机侧。

**Why:** 这是 WireGuard 官方文档（wireguard.com/netns）推荐的标准容器化用法。容器内看到的唯一出网路径就是 wg 接口，所有明文流量在容器 netns 内通过 wg 接口发出，经宿主机侧的 UDP socket 加密后送往远端出口节点。

**Example (Go pseudo-code):**
```go
func wireContainer(hostNS, containerNS netns.NsHandle, spec TunnelSpec) error {
    runtime.LockOSThread()
    defer runtime.UnlockOSThread()

    // 1. 在宿主机 netns 中创建 WireGuard 接口
    wgLink := &netlink.Wireguard{LinkAttrs: netlink.LinkAttrs{Name: spec.InterfaceName}}
    if err := netlink.LinkAdd(wgLink); err != nil {
        return fmt.Errorf("create wg link: %w", err)
    }

    // 2. 配置 WireGuard（密钥、端点、对等体）
    client, _ := wgctrl.New()
    defer client.Close()
    client.ConfigureDevice(spec.InterfaceName, wgtypes.Config{
        PrivateKey: &spec.PrivateKey,
        Peers: []wgtypes.PeerConfig{{
            PublicKey:  spec.PeerPublicKey,
            Endpoint:   spec.PeerEndpoint,
            AllowedIPs: []net.IPNet{{IP: net.IPv4zero, Mask: net.CIDRMask(0, 32)}},
        }},
    })

    // 3. 移入容器 netns（UDP socket 自动留在宿主机侧）
    link, _ := netlink.LinkByName(spec.InterfaceName)
    netlink.LinkSetNsFd(link, int(containerNS))

    // 4. 切到容器 netns 配地址和路由
    netns.Set(containerNS)
    defer netns.Set(hostNS)

    containerLink, _ := netlink.LinkByName(spec.InterfaceName)
    netlink.AddrAdd(containerLink, &netlink.Addr{IPNet: spec.TunnelAddress})
    netlink.LinkSetUp(containerLink)
    netlink.RouteAdd(&netlink.Route{
        LinkIndex: containerLink.Attrs().Index,
        Dst:       &net.IPNet{IP: net.IPv4zero, Mask: net.CIDRMask(0, 32)},
    })

    return nil
}
```

### Pattern 3: 管理 veth 对（SSH 管理路径）

**What:** 用 veth 对在宿主机和容器 netns 之间建立一条私有管理链路，仅供 SSH 接入使用。

**When to use:** 因为容器使用 `--network=none` + WireGuard 全隧道，SSH 流量不能走 WireGuard（否则用户需要先连 WireGuard 才能 SSH），需要一条独立的宿主机到容器的管理路径。

**约束：**
- veth 对只用于宿主机 → 容器方向的 SSH 管理流量
- 容器内的 nftables 规则必须限制 veth 上只允许 SSH 入站 + established/related 回包
- 绝不能通过 veth 对提供到外部网络的路由
- 管理网段使用私有地址（如 `10.99.0.0/30`），不与隧道地址冲突

### Pattern 4: 默认拒绝防火墙（nftables in container netns）

**What:** 在容器命名空间内创建 nftables 规则，实施严格的出站控制。

**Example (Go pseudo-code):**
```go
func applyFirewallRules(containerNS netns.NsHandle, wgIfName, mgmtIfName string) error {
    runtime.LockOSThread()
    defer runtime.UnlockOSThread()
    netns.Set(containerNS)

    conn, _ := nftables.New(nftables.WithNetNSFd(int(containerNS)))

    table := conn.AddTable(&nftables.Table{Family: nftables.TableFamilyIPv4, Name: "filter"})

    outputChain := conn.AddChain(&nftables.Chain{
        Name:     "output",
        Table:    table,
        Type:     nftables.ChainTypeFilter,
        Hooknum:  nftables.ChainHookOutput,
        Priority: nftables.ChainPriorityFilter,
        Policy:   ptrOf(nftables.ChainPolicyDrop),  // 默认拒绝
    })

    // 允许 loopback
    conn.AddRule(/* match oif lo → accept */)

    // 允许 WireGuard 接口出站
    conn.AddRule(/* match oif wg → accept */)

    // 允许管理 veth 上的 established/related
    conn.AddRule(/* match oif mgmt + ct state established,related → accept */)

    return conn.Flush()
}
```

### Anti-Patterns to Avoid

- **使用 Docker 默认 bridge 然后叠加路由规则：** Docker 的 bridge 驱动会注入自己的 iptables/nftables 规则，与自定义出网策略互相干扰，难以保证无旁路。
- **依赖应用层代理（SOCKS/HTTP_PROXY）控制出网：** 无法覆盖 DNS、WebRTC、非代理感知的二进制以及任何绕过环境变量的流量。
- **在 `docker create` 阶段就认为网络已就绪（违反 D-06）：** 网络准备必须在 start 之后、就绪标记之前执行。
- **Go 中不 Lock OS Thread 就做 namespace 操作：** goroutine 可能在 OS 线程间迁移，导致 namespace 操作作用在错误的 netns 上。
- **用 `--privileged` 或 `--cap-add=NET_ADMIN` 运行用户容器：** 用户可以自己修改网络配置绕过隧道约束。所有网络配置必须从宿主机侧（host-agent）注入。

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| 网络接口/路由管理 | 解析 `ip` 命令输出 | `vishvananda/netlink` | 命令输出格式不稳定，netlink 是内核原生协议 |
| WireGuard 配置 | 解析 `wg` 命令输出 | `wgctrl` | 官方 Go 库，直接走 generic netlink |
| nftables 规则 | 拼 `nft` 命令字符串 | `google/nftables` | 纯 Go netlink 实现，类型安全，无需解析文本 |
| 网络命名空间切换 | 直接调 `syscall.Setns` | `vishvananda/netns` | 封装了 fd 管理和线程安全细节 |
| WireGuard 密钥生成 | `exec("wg genkey")` | `wgtypes.GeneratePrivateKey()` | 纯 Go 实现，密码学安全 |
| 出口 IP 验证 | 自建 IP echo 服务 | `https://api.ipify.org` 或 `https://ifconfig.me` | 稳定的公共服务，无需维护 |

**Key insight:** Phase 2 的网络操作全部是 Linux 内核原语（netlink, netns, nftables, WireGuard），Go 生态中每一层都有成熟的纯 Go 库直接对接内核协议，不需要也不应该依赖 shell 命令。

## Common Pitfalls

### Pitfall 1: Go goroutine 在 OS 线程间迁移导致 namespace 操作串台
**What goes wrong:** 在没有 `runtime.LockOSThread()` 的情况下调用 `netns.Set()`，goroutine 被调度到其他线程后，实际操作的是错误的 namespace。
**Why it happens:** Go 的 M:N 调度模型允许 goroutine 在 OS 线程间迁移，而 network namespace 是绑定到 OS 线程的。
**How to avoid:** 所有涉及 namespace 切换的代码块必须：
1. 调用 `runtime.LockOSThread()` 锁定当前 goroutine 到当前线程
2. 完成操作后 `defer runtime.UnlockOSThread()`
3. 使用 `defer netns.Set(origNS)` 恢复原始 namespace
**Warning signs:** 间歇性的"接口不存在"错误，或规则被应用到了宿主机 namespace。

### Pitfall 2: DNS 通过 nscd / systemd-resolved 绕过 namespace 隔离
**What goes wrong:** 容器内的 `resolv.conf` 指向了正确的隧道 DNS，但实际解析请求仍然通过宿主机的 DNS 缓存/转发服务泄漏。
**Why it happens:** `nscd` 通过 Unix socket（位于 `/run`）与进程通信，而 Unix socket 不受网络 namespace 隔离。`systemd-resolved` 的 `libnss_resolve` 模块也有类似行为。
**How to avoid:**
1. 容器镜像中不安装 nscd
2. 容器内 `resolv.conf` 直接写 IP 地址，不使用 `127.0.0.53`（systemd-resolved stub）
3. DNS 校验步骤验证实际 DNS 服务器响应来源，而不只是检查 `resolv.conf` 内容
**Warning signs:** HTTP 出口 IP 正确但 DNS 查询日志显示来源是宿主机 IP。

### Pitfall 3: 容器 PID 竞态和命名空间消失
**What goes wrong:** 在获取容器 PID 后、操作其 namespace 前，容器已停止或重启，导致 PID 失效或指向不同进程。
**Why it happens:** Docker start 和 PID inspect 之间有时间窗口；容器可能因内部错误快速退出。
**How to avoid:**
1. `docker start` → `docker inspect` → 验证容器状态仍为 running
2. 打开 `/proc/<pid>/ns/net` 获取 fd 后保持持有（即使容器停止，fd 仍有效）
3. 在 namespace 操作前检查 fd 有效性
**Warning signs:** "no such process" 或 "invalid argument" 错误出现在网络配置阶段。

### Pitfall 4: Docker 防火墙规则与自定义规则冲突
**What goes wrong:** Docker 引擎在宿主机 namespace 中维护自己的 iptables/nftables 规则链，自定义规则可能被 Docker 覆盖或干扰。
**Why it happens:** Docker 使用固定的规则链名称和优先级，重启 Docker 或容器时会重新生成规则。
**How to avoid:**
1. 使用 `--network=none`，Docker 不会为该容器创建任何网络规则
2. 自定义防火墙规则应用在容器 namespace 内部（不是宿主机 namespace），完全不受 Docker 影响
3. 遵守 Docker 官方的规则链模型，不在 DOCKER 或 DOCKER-USER 链中直接操作
**Warning signs:** 容器重启后防火墙规则丢失，或 Docker 升级后规则行为变化。

### Pitfall 5: 管理 veth 对成为旁路出网路径
**What goes wrong:** 管理 veth 提供了宿主机网络访问，如果容器内进程利用此路径转发流量到外网，等于绕过了 WireGuard 隧道。
**Why it happens:** veth 对连接了两个 namespace，如果宿主机侧启用了 IP 转发且路由允许，容器可以通过 veth → 宿主机 → 外网。
**How to avoid:**
1. 容器 namespace 内 nftables：veth 接口上只允许 SSH 入站 + established/related 回包
2. 宿主机侧：管理网段不做 NAT/MASQUERADE，不配默认路由
3. 容器内不为管理 veth 配置默认网关
4. 泄漏测试必须包含"通过管理 veth 尝试出网"的场景
**Warning signs:** 泄漏测试发现容器可以通过非 WireGuard 路径到达外部 IP。

### Pitfall 6: WireGuard 接口名冲突
**What goes wrong:** 多个容器同时接线时，WireGuard 接口名称在宿主机 namespace 中冲突。
**Why it happens:** 所有 WireGuard 接口最初都在宿主机 namespace 创建，如果使用固定名称（如 `wg0`）会冲突。
**How to avoid:** 使用 host ID 派生接口名：`wg-<hostID[:8]>`，确保唯一性。移入容器 namespace 后可重命名为固定名称（如 `wg0`）。
**Warning signs:** 创建第二个容器时报 "RTNETLINK answers: File exists"。

## Code Examples

### Example 1: 启动前绑定校验

```go
func validateEgressBinding(ctx context.Context, repo Repository, hostID string) (EgressConfig, error) {
    bindings, err := repo.ListHostBindings(ctx, hostID)
    if err != nil {
        return EgressConfig{}, fmt.Errorf("query bindings: %w", err)
    }
    if len(bindings) == 0 {
        return EgressConfig{}, &NetworkError{
            Type:    ErrBindingMissing,
            Message: fmt.Sprintf("host %s has no egress IP binding", hostID),
        }
    }

    egressIP, err := repo.GetEgressIP(ctx, bindings[0].EgressIPID)
    if err != nil {
        return EgressConfig{}, fmt.Errorf("load egress IP: %w", err)
    }

    tunnel, err := repo.GetTunnelConfig(ctx, egressIP.ID)
    if err != nil {
        return EgressConfig{}, fmt.Errorf("load tunnel config: %w", err)
    }

    return EgressConfig{
        ExpectedIP:  egressIP.IPAddress,
        TunnelSpec:  tunnel,
    }, nil
}
```

### Example 2: 获取 Docker 容器的 netns fd

```go
func getContainerNetNS(ctx context.Context, containerName string) (netns.NsHandle, uint32, error) {
    cmd := exec.CommandContext(ctx, "docker", "inspect",
        "-f", "{{.State.Pid}}", containerName)
    output, err := cmd.Output()
    if err != nil {
        return 0, 0, fmt.Errorf("inspect container pid: %w", err)
    }

    pid, err := strconv.ParseUint(strings.TrimSpace(string(output)), 10, 32)
    if err != nil {
        return 0, 0, fmt.Errorf("parse pid: %w", err)
    }
    if pid == 0 {
        return 0, 0, fmt.Errorf("container not running (pid=0)")
    }

    ns, err := netns.GetFromPid(int(pid))
    if err != nil {
        return 0, 0, fmt.Errorf("get netns from pid %d: %w", pid, err)
    }

    return ns, uint32(pid), nil
}
```

### Example 3: 三重校验

```go
type VerifyResult struct {
    EgressIPMatch bool
    ActualEgressIP string
    DNSCorrect    bool
    ActualDNS     string
    LeakBlocked   bool
    LeakTarget    string
}

func verifyNetworkIntegrity(ctx context.Context, containerPID uint32, expected EgressConfig) (VerifyResult, error) {
    var result VerifyResult
    nsenterPrefix := []string{"nsenter", "-t", strconv.Itoa(int(containerPID)), "-n", "--"}

    // 1. 出口 IP 校验
    cmd := exec.CommandContext(ctx, nsenterPrefix[0],
        append(nsenterPrefix[1:], "curl", "-4", "--max-time", "5", "-s", "https://api.ipify.org")...)
    out, err := cmd.Output()
    if err != nil {
        return result, &NetworkError{Type: ErrEgressUnreachable, Message: "egress IP check failed"}
    }
    result.ActualEgressIP = strings.TrimSpace(string(out))
    result.EgressIPMatch = result.ActualEgressIP == expected.ExpectedIP

    // 2. DNS 校验
    cmd = exec.CommandContext(ctx, nsenterPrefix[0],
        append(nsenterPrefix[1:], "cat", "/etc/resolv.conf")...)
    out, _ = cmd.Output()
    result.ActualDNS = extractNameserver(string(out))
    result.DNSCorrect = result.ActualDNS == expected.TunnelSpec.DNSServer

    // 3. 泄漏测试（尝试直连一个公共 IP，应被阻断）
    result.LeakTarget = "1.1.1.1:80"
    cmd = exec.CommandContext(ctx, nsenterPrefix[0],
        append(nsenterPrefix[1:], "timeout", "3",
            "bash", "-c", "echo | nc -w 2 1.1.1.1 80")...)
    err = cmd.Run()
    result.LeakBlocked = (err != nil) // 连接失败 = 泄漏被阻断 = 正确

    return result, nil
}
```

## Data Model Extensions

### egress_ips 表扩展

当前 `egress_ips` 表只有基本的 IP 地址和标签。Phase 2 需要扩展以存储隧道配置：

```sql
ALTER TABLE egress_ips ADD COLUMN wg_endpoint TEXT;
ALTER TABLE egress_ips ADD COLUMN wg_public_key TEXT;
ALTER TABLE egress_ips ADD COLUMN wg_preshared_key TEXT;
ALTER TABLE egress_ips ADD COLUMN wg_allowed_ips TEXT NOT NULL DEFAULT '0.0.0.0/0';
ALTER TABLE egress_ips ADD COLUMN wg_dns_server TEXT;
ALTER TABLE egress_ips ADD COLUMN wg_peer_address INET;
```

### 每主机 WireGuard 密钥

每个主机需要一个 WireGuard 私钥（用于与远端出口节点建立隧道），可以在 `hosts` 表扩展或新建 `host_wg_keys` 表：

```sql
ALTER TABLE hosts ADD COLUMN wg_private_key TEXT;
ALTER TABLE hosts ADD COLUMN wg_public_key TEXT;
```

密钥在主机首次创建时通过 `wgtypes.GeneratePrivateKey()` 生成并持久化。

### 网络失败事件类型（D-12）

利用现有 `events` 表的 `type` 字段和 `metadata` JSONB 字段，定义以下细分事件类型：

| 事件类型 | `type` 值 | metadata 内容 |
|---------|-----------|---------------|
| 绑定缺失 | `net.binding_missing` | `{"host_id": "..."}` |
| 出口 IP 不匹配 | `net.egress_ip_mismatch` | `{"expected": "1.2.3.4", "actual": "5.6.7.8"}` |
| DNS 不走受控路径 | `net.dns_leak` | `{"expected_dns": "10.0.0.1", "actual_dns": "8.8.8.8"}` |
| 非隧道直连未被阻断 | `net.leak_not_blocked` | `{"target": "1.1.1.1:80"}` |
| 出口 IP 不可达 | `net.egress_unreachable` | `{"endpoint": "vpn.example.com:51820"}` |
| 隧道拉起失败 | `net.tunnel_setup_failed` | `{"error": "..."}` |

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| Docker 默认 bridge + iptables MASQUERADE | `--network=none` + 编程化 netns 注入 | Docker 29+ 推 nftables 后端 | Docker 不再干扰自定义网络栈 |
| 手动 `wg` CLI 配置 | `wgctrl` Go 库编程化配置 | wgctrl 持续维护 | 无需 shell out，错误处理更可靠 |
| iptables 手动规则 | `google/nftables` 编程化 + nftables 内核后端 | nftables 成为主流发行版默认 | 纯 Go 操作，无需解析命令输出 |

**Deprecated/outdated:**
- Docker `--link`：已弃用，不应用于任何生产场景
- `iptables-legacy`：在较新发行版中 iptables 命令实际调用 nft 后端，行为可能与预期不同

## Open Questions

1. **WireGuard 出口节点 / VPN 服务端的部署和管理**
   - What we know: Phase 2 负责容器侧的隧道接入和校验，隧道的远端是一个 WireGuard server（提供出口 IP）。
   - What's unclear: WireGuard server 的部署、密钥分发和 peer 管理是否由本系统承担，还是外部管理。
   - Recommendation: Phase 2 将 WireGuard server 信息视为外部配置输入（通过 `egress_ips` 表的扩展字段），不在 Phase 2 内自动化部署 WireGuard server。管理员手工配置或后续阶段补自动化。

2. **管理 veth 的 SSH 端口映射策略**
   - What we know: 用户需要通过 SSH 访问容器，管理 veth 提供了宿主机到容器的私有路径。
   - What's unclear: Phase 3 的 SSH handoff 具体走什么路径（直接 SSH 到管理 IP？控制面代理？端口映射？）。
   - Recommendation: Phase 2 先建好管理 veth 基础设施和 SSH 端口可达性，具体的用户 SSH 接入体验留给 Phase 3。

3. **IPv6 隧道路径**
   - What we know: 当前需求和决策都围绕 IPv4 出口 IP。
   - What's unclear: 是否需要在隧道内支持 IPv6。
   - Recommendation: Phase 2 只处理 IPv4。如果容器内进程发起 IPv6 连接，nftables 规则应同时在 inet/ip6 family 中默认拒绝。

## Environment Availability

> 本阶段依赖的核心工具都是 Linux 内核原语，只需宿主机具备以下能力：

| Dependency | Required By | Available | Version | Fallback |
|------------|------------|-----------|---------|----------|
| Linux kernel netns | 网络隔离 | ✓（Linux 宿主机） | 4.x+ | — |
| WireGuard 内核模块 | 隧道接口 | ✓（Linux 5.6+ 内置） | 内核自带 | `wireguard-tools` 用户态 |
| nftables 内核支持 | 防火墙规则 | ✓（Linux 3.13+，现代发行版默认） | 内核自带 | iptables（见 Discretion） |
| `docker` CLI | 容器管理 | ✓ | 28.x | — |
| `ip` (iproute2) | Go 库调用 netlink，不直接需要 `ip` CLI | ✓ | 宿主机发行版 | 仅 preflight 使用 |
| `nsenter` | 校验步骤在容器 netns 内执行命令 | ✓（util-linux） | 宿主机发行版 | Go 直接 setns + exec |
| `curl` (容器内) | 出口 IP 校验 | ✓（受管镜像应包含） | — | `wget` 或 Go 直接 HTTP 请求 |

**Missing dependencies with no fallback:** None — 所有依赖都是 Linux 标准能力。

**Preflight 扩展：** `deploy/scripts/host-preflight.sh` 已检查 `docker`、`ip`、`nft|iptables`、`systemctl`。Phase 2 应额外检查：
- WireGuard 内核模块可加载：`modprobe wireguard` 或 `ip link add wg-test type wireguard`
- `nsenter` 可用：`command -v nsenter`
- nftables 用户态工具可用：`command -v nft`

## Validation Architecture

### Test Framework
| Property | Value |
|----------|-------|
| Framework | Go `testing` (标准库) |
| Config file | 无独立配置 — Go 内置 |
| Quick run command | `go test ./internal/network/... -count=1 -short` |
| Full suite command | `go test ./... -count=1` |

### Phase Requirements → Test Map

| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| NET-01 | 无绑定时启动失败 | unit | `go test ./internal/network/... -run TestBindingValidation -count=1` | ❌ Wave 0 |
| NET-02 | 所有出站流量走 WireGuard | integration (needs root + netns) | `sudo go test ./internal/network/... -run TestTunnelRouting -count=1 -tags integration` | ❌ Wave 0 |
| NET-03 | DNS 走受控路径 | integration | `sudo go test ./internal/network/... -run TestDNSControl -count=1 -tags integration` | ❌ Wave 0 |
| NET-04 | 非隧道流量被阻断 | integration | `sudo go test ./internal/network/... -run TestLeakBlocked -count=1 -tags integration` | ❌ Wave 0 |
| NET-05 | 就绪前三重校验 | integration | `sudo go test ./internal/network/... -run TestReadinessVerification -count=1 -tags integration` | ❌ Wave 0 |

### Sampling Rate
- **Per task commit:** `go test ./internal/network/... -count=1 -short`
- **Per wave merge:** `go test ./... -count=1`
- **Phase gate:** Full suite green before `/gsd-verify-work`

### Wave 0 Gaps
- [ ] `internal/network/provider_test.go` — NET-01 绑定校验单元测试
- [ ] `internal/network/integration_test.go` — NET-02~NET-05 集成测试（需要 root 权限和 Linux 环境，使用 `//go:build integration` tag）
- [ ] `internal/network/errors_test.go` — 错误类型细分验证

**Note:** NET-02 到 NET-05 的集成测试需要 root 权限和真实 Linux 内核（netns / WireGuard / nftables），在 CI 中需要特权容器或专用 runner。单元测试覆盖接口契约和数据校验逻辑，不依赖特权。

## Project Constraints (from .cursor/rules/)

- 所有面向用户的回复和说明默认使用中文
- 高权限网络动作必须留在 host-agent 执行，控制面不直接持有 Docker / 网络特权
- 任务默认不做自动重试，失败要结构化记录
- 偏向 WireGuard + netns，不接受 Docker 默认 bridge 或应用层代理作为主生产路径
- 单宿主机优先，不提前引入多节点复杂度

## Sources

### Primary (HIGH confidence)
- https://www.wireguard.com/netns/ — WireGuard 官方 namespace 集成文档，确认"birthplace namespace"机制和容器化用法
- https://pkg.go.dev/github.com/vishvananda/netns — netns v0.0.5 Go 文档
- https://pkg.go.dev/github.com/vishvananda/netlink — netlink v1.3.1 Go 文档
- https://pkg.go.dev/golang.zx2c4.com/wireguard/wgctrl — wgctrl Go 文档，确认 API 范围和平台支持
- https://github.com/google/nftables — google/nftables v0.3.0，Apache 2.0 许可
- https://docs.docker.com/engine/network/firewall-nftables — Docker 29+ nftables 后端文档

### Secondary (MEDIUM confidence)
- https://oneuptime.com/blog/post/2026-03-20-dns-resolution-network-namespace/view — 2026 年 namespace DNS 配置指南
- https://superuser.com/questions/1862057/dns-leaks-using-network-namespaces — nscd/systemd-resolved DNS 泄漏分析
- https://haavard.name/posts/2026-02-17-secure-and-isolated-application-networks-using-wireguard/ — 2026 年 WireGuard + netns 隔离实践

### Tertiary (LOW confidence)
- None — 所有关键技术发现都有官方文档或多源交叉验证支撑

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH — 四个核心库都有官方文档和活跃维护记录确认
- Architecture: HIGH — WireGuard 官方文档明确描述了 namespace 容器化模型，与项目需求精确匹配
- Pitfalls: HIGH — DNS 泄漏、goroutine 线程迁移、Docker 规则冲突等均有多源佐证
- Data model: MEDIUM — 扩展方案合理但未在生产中验证过此特定组合

**Research date:** 2026-03-27
**Valid until:** 2026-04-27（核心技术稳定，30 天内不太可能有破坏性变更）
