# Phase 8: SingBoxProvider 与受管镜像 - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-03-28
**Phase:** 08-SingBoxProvider 与受管镜像
**Areas discussed:** Provider 工厂架构, sing-box 配置生成, sing-box 出网路径, sing-box 进程管理, 受管镜像安装
**Mode:** --auto (all decisions auto-selected)

---

## Provider 工厂架构

| Option | Description | Selected |
|--------|-------------|----------|
| RoutingProvider 包装器 | 创建 RoutingProvider 实现 Provider 接口，内部持有 TunnelProvider 和 SingBoxProvider，按 EgressConfig.TunnelType 委托。host-agent 注入 RoutingProvider，对外仍是单一 Provider | ✓ |
| Worker 层 switch | 在 Worker 中持有两个 Provider，根据 egressConfig.TunnelType 选择调用 | |
| 每次动态构造 | 每次 PrepareHost 前根据类型 new 一个对应 Provider | |

**User's choice:** [auto] RoutingProvider 包装器 (recommended default)
**Notes:** 保持 Provider 接口不变，Worker/Server 层零修改。RoutingProvider 封装路由逻辑，与 SING-06 需求直接对应。

---

## sing-box 配置生成

| Option | Description | Selected |
|--------|-------------|----------|
| Go 结构体序列化 | 定义 Go 结构体表示 sing-box 配置模型，将 ProxySpec.OutboundConfig 嵌入 outbound 数组，序列化为 JSON | ✓ |
| text/template 模板 | 使用 Go 模板生成 sing-box 配置 JSON | |
| 字符串拼接 | 直接构建 JSON 字符串 | |

**User's choice:** [auto] Go 结构体序列化 (recommended default)
**Notes:** 类型安全，易于测试，支持 JSON tag 控制序列化。outbound 部分使用 json.RawMessage 直接嵌入用户配置，不做二次解析。

---

## sing-box 出网路径

| Option | Description | Selected |
|--------|-------------|----------|
| 管理 veth 复用 | sing-box 在容器 netns 内运行，proxy outbound 通过 mgmt0 → 宿主机出网。容器内添加 proxy server host route，nftables 允许 mgmt0 proxy 出站，宿主机做 masquerade | ✓ |
| 新增 proxy veth | 创建第二个 veth pair 专用于代理流量，与管理 veth 分离 | |
| 修改容器网络模式 | 不使用 --network=none，改用 Docker 自定义网络 + nftables 限制 | |

**User's choice:** [auto] 管理 veth 复用 (recommended default)
**Notes:** 这是核心架构决策。--network=none 容器中 sing-box 无法直接访问代理服务器，需通过管理 veth 提供出网路径。复用已有 mgmt0 避免增加 veth 对数量。sing-box bind_interface 确保代理连接不被 tun auto_route 截获。

---

## sing-box 进程管理

| Option | Description | Selected |
|--------|-------------|----------|
| nsenter 启动 + 容器停止自动清理 | nsenter -t <pid> -n -m -p 进入容器 net/mount/pid 命名空间启动 sing-box。进程在容器 PID ns 内，容器停止时自动终止。CleanupHost 增加兜底清理 | ✓ |
| docker exec 启动 | docker exec -d <container> sing-box run。简单但容器需 CAP_NET_ADMIN | |
| 容器 startup service | 在容器启动脚本中配置 sing-box 服务。但配置在 PrepareHost 时才写入，时序冲突 | |

**User's choice:** [auto] nsenter 启动 + 容器停止自动清理 (recommended default)
**Notes:** nsenter 方式让 host-agent（root）控制进程启动，无需给容器额外 capabilities。进入 PID ns 确保容器停止时进程被自动回收。tun 接口固定名 tun0（独立 netns 不冲突）。

---

## 受管镜像安装

| Option | Description | Selected |
|--------|-------------|----------|
| 官方预编译二进制 | 从 sing-box GitHub Release 下载 Linux amd64 二进制到 /usr/local/bin/sing-box | ✓ |
| 从源码编译 | Dockerfile 中 go build sing-box | |
| 包管理器 | 通过 apt 或第三方仓库安装 | |

**User's choice:** [auto] 官方预编译二进制 (recommended default)
**Notes:** 最小化镜像构建时间和体积。版本通过 Dockerfile ARG 管理。创建 /etc/sing-box/ 配置目录。

---

## Agent's Discretion

- sing-box tun 参数（inet4_address、mtu、stack）
- tun0 就绪等待策略（轮询间隔、超时）
- 宿主机 masquerade 规则实现方式
- sing-box 启动失败的回滚策略
- sing-box 具体版本选择

## Deferred Ideas

None — discussion stayed within phase scope
