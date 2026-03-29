# Phase 8: SingBoxProvider 与受管镜像 - Context

**Gathered:** 2026-03-28
**Status:** Ready for planning

<domain>
## Phase Boundary

代理类型的主机能通过 sing-box tun 模式实现全流量代理出网，安全性和校验标准等同 WireGuard。包含：SingBoxProvider 实现（tun inbound + 代理 outbound + DNS）、Provider 工厂按 tunnel_type 自动路由、受管镜像预装 sing-box 二进制、sing-box 进程生命周期管理、nftables 适配 tun0 接口、三重校验对代理类型生效。

**不在本阶段范围内：** 前端表单动态切换（Phase 9）、代理测试 API（Phase 9）、代理健康检测/告警（Future）。

</domain>

<decisions>
## Implementation Decisions

### Provider 工厂架构

- **D-01:** 创建 `RoutingProvider` 实现 `Provider` 接口，内部持有 `TunnelProvider` 和 `SingBoxProvider` 两个实例
- **D-02:** `RoutingProvider.PrepareHost` 读取 `HostNetworkSpec.Egress.TunnelType`，委托给对应实现；`CleanupHost` 同理
- **D-03:** `cmd/host-agent/main.go` 注入 `RoutingProvider`（替代直接注入 `TunnelProvider`），对外仍是单一 `Provider` 接口
- **D-04:** 现有 WireGuard 路径完全不受影响——`RoutingProvider` 在 `TunnelType == "wireguard"` 时直接委托给 `TunnelProvider`

### sing-box 配置生成

- **D-05:** 在 SingBoxProvider 内定义 Go 结构体表示 sing-box 配置模型（`singBoxConfig` / `singBoxInbound` / `singBoxDNS` / `singBoxRoute` 等），用于构建完整配置
- **D-06:** `ProxySpec.OutboundConfig`（用户提供的 sing-box outbound JSON）直接嵌入 outbound 数组，不做二次解析
- **D-07:** 固定模板部分：inbound 为 tun 模式（`auto_route: true, strict_route: true`），DNS server 使用 `ProxySpec.DNSServer`，route 为默认全流量
- **D-08:** 配置序列化为 JSON 后通过 `/proc/<pid>/root/etc/sing-box/config.json` 写入容器文件系统

### sing-box 出网路径

- **D-09:** sing-box 在容器 netns 内运行，其 proxy outbound 通过管理 veth（mgmt0）→ 宿主机出网，到达代理服务器
- **D-10:** 在容器 netns 内添加 host route：`<proxy_server_ip>/32 via <mgmt_gateway> dev mgmt0`，使 sing-box 的代理连接走 mgmt0
- **D-11:** sing-box outbound 配置使用 `bind_interface: "mgmt0"` 确保代理连接绑定到管理 veth，不被 tun auto_route 截获
- **D-12:** 宿主机侧需为管理 veth 启用 IP 转发 + masquerade，使代理流量能正确路由
- **D-13:** nftables 规则需为 proxy 模式扩展：允许 mgmt0 上 OUTPUT 到 proxy_server_ip:port 的连接（在 WireGuard 模式下 mgmt0 仅允许 established/related）

### sing-box 进程管理

- **D-14:** 使用 `nsenter -t <pid> -n -m -p` 进入容器的 net/mount/pid 命名空间，后台启动 `sing-box run -c /etc/sing-box/config.json`
- **D-15:** sing-box 进程在容器 PID 命名空间内，容器停止（docker stop）时自动被终止
- **D-16:** `CleanupHost` 增加兜底清理：通过 nsenter 检查并 kill 残留的 sing-box 进程
- **D-17:** `PrepareHost` 启动 sing-box 后需等待 tun0 接口出现（轮询 `ip link show tun0`），超时则报错
- **D-18:** tun 接口使用固定名 `tun0`——每个容器有独立 netns，不会冲突

### 受管镜像预装

- **D-19:** Dockerfile 中从 sing-box GitHub Release 下载 Linux amd64 预编译二进制，安装到 `/usr/local/bin/sing-box`
- **D-20:** 创建配置目录 `/etc/sing-box/`，运行时由 SingBoxProvider 写入配置文件
- **D-21:** sing-box 版本以 Dockerfile ARG 形式管理，便于升级

### nftables 适配

- **D-22:** `ApplyFirewallRules` 需支持 proxy 模式变体：接受 tun0 ifindex（替代 wg ifindex）+ proxy server 信息
- **D-23:** proxy 模式 OUTPUT chain 增加规则：允许 mgmt0 到 `<proxy_server_ip>:<proxy_server_port>` 的 TCP/UDP 出站
- **D-24:** 其余规则保持一致：默认 DROP、lo 允许、IPv6 全部 DROP

### 三重校验

- **D-25:** `VerifyNetworkIntegrity` 已在 Phase 7 适配 proxy DNS 路径（使用 `Proxy.DNSServer`），本阶段确保端到端验证通过
- **D-26:** 泄漏检测（`/dev/tcp/1.1.1.1/80` 超时失败）在 proxy 模式下同样生效——nftables 默认 DROP 阻断直连

### Agent's Discretion

- sing-box 进程启动后 tun0 就绪的等待策略（轮询间隔、最大超时）
- sing-box 配置中 tun 的具体参数（inet4_address、mtu、stack 选择 system/gvisor/mixed）
- masquerade 规则的具体实现方式（iptables vs nftables on host side）
- sing-box 启动失败时的错误恢复和回滚策略
- sing-box 二进制的具体版本选择

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### 代理隧道整体设计

- `.planning/proxy-tunnel-plan.md` — 代理协议出网的完整设计方案，包含架构图、sing-box 配置模板、Provider 工厂设计、各协议 proxy_config 示例
- `.planning/REQUIREMENTS.md` — SING-01 到 SING-07 需求定义

### Phase 7 上下文（数据层已完成）

- `.planning/phases/07-data-layer-typing/07-CONTEXT.md` — 数据层决策，包含 EgressConfig/ProxySpec/TunnelSpec 设计、校验分支、API 适配

### 现有网络层（SingBoxProvider 对标实现）

- `internal/network/provider.go` — Provider 接口定义（PrepareHost / CleanupHost）
- `internal/network/tunnel_provider_linux.go` — WireGuard TunnelProvider 实现，SingBoxProvider 的直接对标
- `internal/network/wireguard.go` — WireGuard 接口注入逻辑（InjectWireGuard），理解 birthplace namespace 模式
- `internal/network/firewall.go` — ApplyFirewallRules 实现，需扩展支持 proxy 模式
- `internal/network/verify.go` — VerifyNetworkIntegrity 三重校验，已支持 Proxy DNS 路径

### 数据类型定义

- `internal/network/types.go` — TunnelSpec / ProxySpec / EgressConfig / HostNetworkSpec 当前定义
- `internal/network/validate.go` — ValidateEgressBinding 校验逻辑，已按 tunnel_type 分支

### 容器生命周期

- `internal/runtime/tasks/worker.go` — Worker 中 startHost/rebuildHost/stopHost 流程，PrepareHost/CleanupHost 调用点
- `internal/agent/server.go` — host-agent Server 构造，当前直接注入 TunnelProvider
- `cmd/host-agent/main.go` — host-agent 入口，Provider 实例化位置

### 受管镜像

- `deploy/docker/managed-user/Dockerfile` — 受管用户镜像，需预装 sing-box 二进制

### 网络辅助

- `internal/network/netns.go` — GetContainerNetNS 容器 netns 获取
- `internal/network/mgmt_veth.go` — InjectManagementVeth 管理 veth 注入
- `internal/network/dns.go` — ConfigureContainerDNS 配置

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets

- `GetContainerNetNS` (`netns.go`): 获取容器 netns 句柄和 PID，SingBoxProvider 直接复用
- `InjectManagementVeth` (`mgmt_veth.go`): 管理 veth 注入，SingBoxProvider 直接复用
- `ConfigureContainerDNS` (`dns.go`): DNS 配置写入 /etc/resolv.conf，可复用于 proxy 模式（使用 ProxySpec.DNSServer）
- `ApplyFirewallRules` (`firewall.go`): 防火墙规则，需扩展支持 proxy 模式（tun ifindex + proxy server 白名单）
- `VerifyNetworkIntegrity` (`verify.go`): 三重校验，已支持 proxy DNS 路径，直接复用
- `resolveContainerIfIndexes` (`tunnel_provider_linux.go`): 解析容器内接口 ifindex，需适配 tun0 接口名

### Established Patterns

- Provider 接口模式：`PrepareHost(ctx, HostNetworkSpec) error` + `CleanupHost(ctx, HostNetworkSpec) error`
- netns 操作使用 `netlink` + `netns` 库，在宿主机和容器 netns 间切换
- WireGuard 接口在宿主机创建后移入容器 netns（birthplace namespace 模式）——sing-box 是 userspace 进程，无法直接复用此模式
- 防火墙规则使用 `google/nftables` 库在容器 netns 内应用
- 容器文件系统通过 `/proc/<pid>/root/` 路径从宿主机访问

### Integration Points

- `cmd/host-agent/main.go` — Provider 实例化点，需改为 RoutingProvider
- `internal/runtime/tasks/worker.go` — PrepareHost/CleanupHost 调用点，无需修改（接口不变）
- `internal/network/firewall.go` — ApplyFirewallRules 需扩展签名或新增 proxy 变体
- `deploy/docker/managed-user/Dockerfile` — 需添加 sing-box 二进制

</code_context>

<specifics>
## Specific Ideas

- sing-box tun 模式的 `auto_route: true` + `strict_route: true` 确保内核级全流量接管，与 WireGuard 等效
- 管理 veth 复用为 sing-box 代理连接的出口路径（不新增 veth），通过 host route + bind_interface 精确路由
- SingBoxProvider 的 PrepareHost 流水线与 TunnelProvider 高度对称：差异仅在 WireGuard 注入 → sing-box 配置写入 + 进程启动
- `resolveContainerIfIndexes` 需适配：WireGuard 模式查找 `wg-xxxx`，proxy 模式查找 `tun0`

</specifics>

<deferred>
## Deferred Ideas

None — discussion stayed within phase scope

</deferred>

---

*Phase: 08-singboxprovider*
*Context gathered: 2026-03-28*
