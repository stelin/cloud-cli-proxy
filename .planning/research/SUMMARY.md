# Project Research Summary

**Project:** Cloud CLI Proxy — `claude-shell`（v1.3 里程碑）  
**Domain:** 单机 Go 二进制，透明包装 Docker 内 Claude Code；全流量经 sing-box tun + nftables；系统级指纹与容器信号收敛  
**Researched:** 2026-04-09  
**Confidence:** MEDIUM-HIGH（官方文档与仓库内架构证据为主；跨平台 Desktop/实机行为需阶段内验证）

## Executive Summary

`claude-shell` 是面向开发者的**本地交付物**：用户用单一 `claude` 入口（PATH 前置）替代官方 CLI，由 Go 二进制编排容器，在容器内运行官方 **Native Install** 路径的 Claude Code（**不**依赖已弃用的 npm 全局安装），并与云侧产品共享同一套叙事：**tun 级全局路由 + 默认拒绝旁路**，私网与回连宿主机走明确例外（`host.docker.internal` / `host-gateway`），再通过 `verify` 子命令做出口、DNS 与指纹自检。

业界同类（Toolbx、Distrobox）追求与宿主深度融合；本产品目标相反——**出网可控、指纹面收敛**。实现上应复用仓库已有 **sing-box 配置思想**（与 `buildSingBoxConfig` 同源的 tun、分流、DNS），但**编排边界**与生产 **SingBoxProvider（Linux netlink + `network=none` + mgmt veth）** 分叉：跨平台 MVP 优先 **bridge + 容器内 sing-box tun**；Linux 上可选 **`network=none` + 注入 veth** 作为「与生产同构」的加强模式。CLI 侧 STACK 倾向 **`github.com/docker/docker/client`** 做类型化编排与流式 attach；架构研究则指出 **`docker run` 子进程**与手工行为一致、TTY/信号语义直观——**路线图建议：先按依赖链打通镜像 entrypoint 与最简运行路径，再在计划中明确「子进程 MVP → SDK 收敛」或「直接 SDK + 完整 Attach/Resize」**，避免长期维持两套编排。

主要风险集中在：**TTY/信号/SIGWINCH**（交互与 resize 误杀）、**Docker Desktop 与 Linux Engine 语义差异**（host 网络、`host-gateway`、nft 在 VM 内核上的表现）、**sing-box tun + nft 规则顺序与能力集**（泄漏或局部绕开 tun）、**卷 UID 与 Claude Code 持久化/自动更新**、**`/proc` 伪造优先用 `docker run -v` 注入而非容器内 mount**（避免不当 `--privileged`）、以及 **garble** 与反射/排障权衡。缓解方向：以 **Linux 裸机/等同 VM** 为网络正确性准绳，Desktop 作 Tier-2；`verify` 与集成测试覆盖 resize、SIGINT、DNS、出口 IP；默认 **`DISABLE_AUTOUPDATER=1`** 与镜像版本策略对齐；产品表述坚持**诚实边界**，不承诺「不可检测」或对抗级反沙箱。

## Key Findings

### Recommended Stack

详见 `.planning/research/STACK.md`。核心结论：**Go 1.26.1** 与 **garble**（上游要求 Go 1.26+）对齐；CLI 默认 **Cobra（≥v1.10）** 承载根命令与 `verify`；容器侧 **sing-box 1.13.3**（与受管镜像一致）tun + **`auto_route` / `auto_redirect`（Linux）**；YAML 新代码优先 **go.yaml.in/yaml/v3**，避免无人维护的 `gopkg.in/yaml.v3`；嵌入资源用 **`embed` + AES-GCM**，密钥运行时注入。指纹面以 **静态 bind mount** 覆盖 `/proc/cpuinfo`、`/proc/meminfo` 等为主，必要时再评估 **lxcfs** 单文件 bind。Claude Code 以官方 **curl/ps1 Native** 安装为准。

**Core technologies:**

- **Go 1.26.1 + Cobra**：单二进制 CLI、子命令与补全生态成熟。  
- **garble**：发布混淆；需接受 README 限制（反射、`ReadBuildInfo` 等）并 CI 双轨。  
- **sing-box 1.13.3（tun + 路由分流）**：与现网哲学一致；注意 MPTCP/UID 分流等字段。  
- **Docker Engine Go client**（`docker/docker/client`）：推荐用于产品级编排；与架构研究中「`docker run` 子进程」可分期取舍。  
- **`go.yaml.in/yaml/v3`**：配置解析的安全维护线。

### Expected Features

详见 `.planning/research/FEATURES.md`。

**Must have (table stakes):**

- 单一 `claude` 入口、PATH 透明替换。  
- 冷启动可感知（拉镜像、建容器、健康检查、日志与退出码）。  
- 容器内可复现的 Claude Code 安装与运行（全隧道下）。  
- **tun + nftables 默认拒绝**，公网走代理、私网/回连策略明确。  
- 本地回环与 RFC1918、`host-gateway` 回宿主机。  
- **`verify`**：出口 IP、DNS、指纹/容器标记自检。  
- 降低粗粒度容器信号（如 `/.dockerenv`、可读 cgroup）。

**Should have (differentiators):**

- 与云主机一致的 **网络哲学**（全隧道 + 默认拒绝）。  
- **系统级指纹**（`machine-id`、`/proc/*`），优于仅 Node 层 patch。  
- **garble** 发布单一二进制。  
- 可选：按官方 env **收敛非必要外连**（如 `DISABLE_TELEMETRY` 等，以 Anthropic 文档为准）。

**Defer (v2+):**

- 对抗级反检测、宣称无法被检测。  
- garble 与「艺术化」输出等非核心体验可后置（FEATURES MVP 已列）。

### Architecture Approach

详见 `.planning/research/ARCHITECTURE.md`。要点：**不经过控制面**；建议模块如 `cmd/claude-shell` / `internal/claudeshell`；**抽取** `singbox_config` 中与 Provider 无关的 JSON 生成，供 CLI 与 Linux agent 共用。网络默认 **bridge + 容器内 tun + 分流** 以换跨平台；Linux 可选 **`network=none` + mgmt** 对齐生产。配置以**挂载文件为主**、环境变量为辅。生命周期：**信号转发**、**SIGWINCH**、保证容器停止与 `--rm` 清理。

**Major components:**

1. **`claude` CLI**：Docker 编排、TTY、配置渲染、（可选）`verify`。  
2. **`internal/claudeshell/config`**：合并 env/文件，输出 sing-box JSON。  
3. **镜像 entrypoint**：sing-box → nft → Claude Code 就绪顺序；与 **machine-id / proc / 反粗检测** 分层叠加。

### Critical Pitfalls

详见 `.planning/research/PITFALLS.md`。

1. **TTY / 信号 / SIGWINCH** — 使用 `-it`、评估 `--init`/`tini`，文档与测试覆盖 resize、Ctrl+C、EOF；慎改 `--sig-proxy`。  
2. **Bind mount 与 UID** — 固定镜像用户或 `--user` 对齐；`verify` 断言可写路径；`.claude` 持久化策略明确。  
3. **Docker Desktop vs Linux** — 不以 Desktop 复现生产「三重校验」同级承诺；`host.docker.internal`/`host-gateway` 单一参数表。  
4. **sing-box tun + nft** — `NET_ADMIN`、`/dev/net/tun`、strict_route、exclude_mptcp、UID 分流与规则顺序纳入 `verify`。  
5. **`/proc` 伪造** — 优先 **`docker run -v`** 注入；避免依赖容器内 `mount` 或滥用 `--privileged`。  
6. **Claude Code 自动更新与登录** — 默认 `DISABLE_AUTOUPDATER=1`；卷与账号隔离文档化。  
7. **garble** — CI 双轨、谨慎 `-tiny`/`-literals`，保留诊断构建渠道。

## Implications for Roadmap

基于合并研究，建议阶段划分如下（可与里程碑内子域再映射）。

### Phase 1: 镜像与 entrypoint 基线（网络栈先于「好用」）

**Rationale:** 无稳定 tun/nft/Claude Code 启动顺序，上层 CLI 无法验收。  
**Delivers:** 受管镜像变体或 entrypoint：sing-box 就绪 → nft → Claude Code；与健康检查/失败提示。  
**Addresses:** FEATURES 表功能「全流量经 tun」「容器内 Claude Code」、STACK sing-box 要点。  
**Avoids:** PITFALLS §5（能力与设备误配）、§6（串行入口过慢无 readiness 拆分）。

### Phase 2: sing-box 配置模板与分流（私网直连 + 公网代理）

**Rationale:** 与现网 JSON 模型对齐，减少漂移；先定 `route.rules` 再写 `verify`。  
**Delivers:** 共享包或模板：`ip_is_private`/`route_exclude`、`direct` 绑定接口（bridge 下 `eth0` 等）、DNS detour。  
**Uses:** STACK sing-box 字段；ARCHITECTURE `buildSingBoxConfig` 抽取。  
**Avoids:** §5 DNS 泄漏、§3 Desktop 与 Linux 混用假设。

### Phase 3: 最简 `claude` 包装与 Docker 编排

**Rationale:** 验证 pull/create/exec 闭环；此阶段明确 **子进程 `docker run` vs Docker SDK** 的选型并写入 PLAN。  
**Delivers:** 冷启动日志、退出码、可选 dry-run；最小非 TTY 路径跑通。  
**Implements:** ARCHITECTURE 建议构建顺序第 3 步。  
**Avoids:** §6 冷启动感知差、ImagePull 无条件全拉（STACK 已提醒）。

### Phase 4: TTY、信号、SIGWINCH 与交互体验

**Rationale:** 与「透明替换」用户预期强相关。  
**Delivers:** raw 模式、`SIGWINCH`、容器 resize/kill 策略；集成测试。  
**Avoids:** PITFALLS §1。

### Phase 5: 指纹与反粗检测（分层交付）

**Rationale:** 依赖稳定容器与挂载契约。  
**Delivers:** `machine-id`、`/proc` 文件注入、`.dockerenv`/cgroup 等逐项；`verify` 可重复断言。  
**Avoids:** §4（容器内 mount）、§8（过度承诺）。

### Phase 6: `verify` 与遥测/文档收口

**Rationale:** 信任与运维闭环。  
**Delivers:** 出口/DNS/指纹检查；官方 Claude Code data-usage env 文档化（不采信未验证第三方「源码」）。  
**Uses:** FEATURES 专题 6。

### Phase 7: garble 与多架构发布

**Rationale:** 交付物收尾；非功能前置 blocker。  
**Delivers:** `garble build` CI、arm64/amd64 镜像策略。  
**Avoids:** §9、§10。

### Phase Ordering Rationale

- **先栈后壳：** 网络与进程就绪是 CLI 的前提（ARCHITECTURE 构建顺序 1→2→3）。  
- **先通后美：** TTY/信号在「能跑」之后加固。  
- **指纹与反检测**在网络与挂载稳定后迭代，避免与 tun 问题纠缠。  
- **garble/多架构**最后收敛，降低调试成本。

### Research Flags

Phases likely needing deeper research during planning:

- **Phase 1–2（sing-box + nft + Desktop）：** 最小 capability 集合、**nft + auto_redirect** 与默认拒绝的实机矩阵（PITFALLS Gaps）。  
- **Phase 5（指纹）：** `/proc` bind 在 Docker Desktop 当前版本上的稳定性（ARCHITECTURE §11）。  
- **Phase 7（garble）：** 与 Docker client/依赖反射组合的回归策略（STACK 置信度「中」）。

Phases with standard patterns (skip extra research if PLAN 已引用官方文档):

- **Cobra CLI 结构、Docker 官方 run/attach 文档、Claude Code Native 安装与 data-usage、garble README caveats。**

## Confidence Assessment

| Area | Confidence | Notes |
|------|------------|-------|
| Stack | HIGH | Cobra、Docker client、garble README、sing-box Tun 文档、Claude 官方安装页 |
| Features | MEDIUM-HIGH | 透明代理与检测面多源一致；Claude 细节以官方为准 |
| Architecture | MEDIUM-HIGH | 与仓库 `singbox_config`、Provider 分叉分析清晰；跨平台需测 |
| Pitfalls | MEDIUM-HIGH | Docker/sing-box/garble 权威为主；反检测与策略为推理+常识 |

**Overall confidence:** MEDIUM-HIGH

### Gaps to Address

- **Claude Code + 运行时**在指定版本矩阵下的外连清单（FEATURES Gaps）——需抓包或发行说明随版本验证。  
- **Docker Desktop for Linux** vs **纯 Engine** 的 CI 矩阵（FEATURES）。  
- **Apple Silicon / rootless** 下 tun/nft 边界（FEATURES / ARCHITECTURE）。  
- **garble 与项目依赖**的最终组合（PITFALLS Gaps）——以依赖锁定后 CI 为准。

## Sources

### Primary (HIGH confidence)

- `.planning/research/STACK.md`、`FEATURES.md`、`ARCHITECTURE.md`、`PITFALLS.md`（本仓库 v1.3 调研）  
- Cobra、garble、Docker Engine Go SDK、go.yaml.in、sing-box Tun、Anthropic Claude Code setup / data-usage（各文件「来源」节所列 URL）

### Secondary (MEDIUM confidence)

- systemd CONTAINER_INTERFACE、Distrobox/Toolbx 文档、社区容器检测与 Desktop 差异讨论

### Tertiary (LOW confidence)

- 第三方对 Claude Code 内部实现的解析 — **不作为需求依据**

---
*Research completed: 2026-04-09*  
*Ready for roadmap: yes*
