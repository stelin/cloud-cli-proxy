# Feature Landscape：claude-shell 本地透明代理

**Domain：** 单机 Go 二进制，用 Docker 透明包裹 Claude Code，全流量经 sing-box tun + nftables，宿主机指纹与容器可观测性可控。  
**Researched：** 2026-04-09  
**Confidence：** **MEDIUM-HIGH**（透明代理与容器检测面：多源一致 + 部分官方文档；Claude Code「源码级」细节：**以 Anthropic 官方文档为准**，社区/媒体二次解读标为 LOW）

---

## 与已有能力的关系（依赖边界）

| 已有（云主机平台） | claude-shell 复用 / 不复用 |
|-------------------|---------------------------|
| 受管镜像、WireGuard/sing-box、nftables、三重网络校验思路 | **复用设计理念**；本地为**另一交付形态**，镜像与编排由 `claude` 二进制驱动，不依赖控制面 API |
| `tools/spoof-fingerprint.js`（Node 层 patch `os`） | v1.3 目标为**容器内系统级**指纹（`/etc/machine-id`、`/proc/*` 等），**不依赖**该脚本；可作对照：Node 仅能覆盖运行时 API，挡不住直接读 `/proc` 的本机工具 |
| 管理后台 / JWT | **非表功能**；本地 CLI 可独立发布 |

---

## Table Stakes（不做则不像「可替换的 claude」）

| Feature | Why Expected | Complexity | Dependencies | Notes |
|--------|--------------|------------|--------------|-------|
| 单一 `claude` 入口，用户可放在 `PATH` 前替代官方 CLI | 产品承诺即「透明替换」 | Med | Docker（或兼容运行时）、镜像拉取/缓存策略 | 与 PROJECT.md「单一 Go 二进制」一致 |
| 首次/冷启动：拉镜像、建容器、健康检查可感知 | 长耗时不可接受为「卡死」 | Med | 镜像仓库、本地磁盘 | 需明确日志与退出码 |
| 容器内安装/运行 Claude Code（如 Bun standalone） | 与「不依赖 npm/spoof.js」一致 | Med | 官方/既定安装路径、网络仅走隧道 | 安装脚本需在全隧道下可复现 |
| 全流量出站经代理（tun）+ 默认拒绝旁路 | 与云侧产品同一安全承诺 | High | sing-box、tun、nftables | 与现有 RoutingProvider 哲学一致，实现场所在本地 |
| 本地回环与私网访问宿主机（`localhost`、RFC1918 等） | IDE、浏览器、本地 API 常见 | Med | **host-gateway / 等价路由** | PROJECT.md 已列 host-gateway |
| `verify`：出口 IP、DNS、指纹/容器标记自检 | 运维与用户信任 | Med | 与隧道同构的探测命令 | 对齐云侧「代理测试」心智 |
| 反容器粗检测（如 `/.dockerenv`、`/proc/1/cgroup` 可读性） | 降低「一眼容器」信号 | Low-Med | 镜像 entrypoint + bind mount | 无法保证对抗专业沙箱指纹 |

---

## Differentiators（差异化，非人人会做）

| Feature | Value Proposition | Complexity | Dependencies | Notes |
|--------|-------------------|------------|--------------|-------|
| 与云主机**同一套**网络哲学（tun 全量 + nft 默认拒绝） | 品牌与技术叙事一致 | High | 本地内核能力、Capabilities | 本地权限模型与云端不同，需单独验证 |
| 系统级指纹：`machine-id`、`cpuinfo`/`meminfo` 等 | 对齐安全研究中的常见采集面，优于仅 Node patch | Med | 只读 bind、稳定种子策略 | 见下文「指纹面」 |
| `garble` 等混淆发布单一二进制 | 降低 casual reverse；非安全银弹 | Low-Med | Go 构建链 | PROJECT.md 已列 |
| 可选：最小化 Claude Code 非必要外连（见官方 env） | 与「出口可控」叙事一致 | Low | 文档化默认 env | 见下文官方遥测 |

---

## Anti-Features（明确不做或慎做）

| Anti-Feature | Why Avoid | What to Do Instead |
|--------------|-----------|-------------------|
| 完整「反沙箱」/ 反取证（对抗商业风控、恶意用途） | 法律、伦理与维护成本 | 仅服务**合规的出口 IP + 开发隔离**叙事 |
| 100% 无法被检测的容器 | 技术上不承诺；cgroup/mountinfo 等面持续演化 | 明确边界：降低脚本级检测，不宣称隐身 |
| 与 Distrobox/Toolbx 同款「深度主机融合」 | 目标相反：需网络隔离与指纹控制，非 HOME/X11 全家桶 | 最小挂载、显式 host 访问 |
| 依赖 Docker `host` 网络作主路径 | 破坏隔离与路由可控性 | `network=none` + tun + 显式 host 路由（与现有架构一致） |
| 把未证实的「源码泄露」细节写进产品规格 | 来源混杂 | 以 Anthropic 官方 data-usage 为准；第三方仓库标为待验证 |

---

## 专题研究（对应需求 1–7）

### 1. 透明包裹 CLI 的既有工具（Docker / Toolbx / Distrobox）

**典型模式**

- **Toolbx（原 Toolbox）**：Podman/OCI 上叠 `toolbox create | enter | run`，面向**开发/排障**，默认把用户 HOME、X11/Wayland、D-Bus、`/run/host` 等与主机**深度绑定**；目标是「不在宿主机装包」，不是强隔离。
- **Distrobox**：POSIX shell 包装 Docker/Podman/lilipod，`distrobox enter` 执行容器内 shell，支持 `distrobox-export` 把应用导出到宿主机菜单；README 明确 **隔离不是目标**，与主机**紧耦合**。
- **与 claude-shell 的对比**：上述工具普遍追求**透明融入主机**；claude-shell 追求 **出网可控 + 指纹与容器信号收敛**，产品目标相反，可借鉴的是：**OCI 生命周期**（create/start/exec）、**非交互脚本化**、**dry-run 打印实际 docker 参数**（Distrobox 的 `-d` 模式）便于排障。

**可借鉴表功能**：一键进入/执行、命名容器复用、日志与退出码。

---

### 2. 设备指纹：容器里常见要「对齐」的面

以下为安全与运维工具常读的**系统面**（语言运行时 patch 往往盖不住）：

| 表面 | 说明 |
|------|------|
| `/etc/machine-id`、`/var/lib/dbus/machine-id` | 稳定设备标识 |
| `/proc/cpuinfo`、`/proc/meminfo` | CPU 型号、核数、内存叙事 |
| `uname -a` / `uname(2)` | 内核 release、架构 |
| `hostname` / `/etc/hostname` | 与证书、遥测主机名常见关联 |
| 网络：`ip`/`ss`、路由表、默认网关、DNS `resolv.conf` | 与「像不像真机」强相关 |
| **MAC / 网卡列表** | Node 的 `os.networkInterfaces` 已在本仓库 spoof 脚本覆盖；容器内需对齐虚拟网卡叙事 |
| **容器特有**：`/.dockerenv`、`/proc/1/cgroup`、`/proc/self/mountinfo` 中的 overlay 路径等 | 见第 3 节 |

**建议**：在 FEATURES 层把「指纹」定义为 **对用户可见工具一致」+ `verify` 可断言，而非对抗所有采集器。

---

### 3. 容器「反检测」：安全研究常见检查（防御方视角）

常见**启发式**（多源一致，含社区与 systemd 讨论）：

- 根文件系统 inode、`/.dockerenv` 存在性（Docker 非正式惯例；维护者曾提示勿依赖其长期存在）。
- `/proc/1/cgroup`：v1 时代含 `docker` 路径较多；**cgroup v2** 下常简化为 `0::/`，需结合其他信号。
- **`/proc/self/mountinfo`**：overlay、`docker/containers/<id>` 等路径可暴露运行环境。
- **`container=` 环境变量**（systemd [容器接口](https://systemd.io/CONTAINER_INTERFACE/) 建议）：若在进程环境中存在可辅助判定。
- **PID 1 进程名**、init 是否为 systemd 等。

**对产品含义**：PROJECT.md 中的「删 `.dockerenv`、伪造 cgroup」属于**降低脚本级信号**；在 cgroup v2 + 新内核上需持续实测。**不应**在路线图写「无法被检测」。

---

### 4. 透明代理模式：proxychains / tsocks / tun2socks / sing-box tun

| 机制 | 工作层次 | 典型优点 | 典型限制 |
|------|----------|----------|----------|
| **proxychains / tsocks** | `LD_PRELOAD` 钩 `connect()` 等 | 无需 root、按进程启用 | 多针对 **TCP**；**静态链接**、Go 默认 net、部分 DNS 路径易**泄漏**；与「全流量强制出网」不完全同构 |
| **tun2socks** | TUN 三层 + 用户态栈转 SOCKS 等 | 应用无感、覆盖面大 | 需 TUN/CAP_NET_ADMIN 等；需与路由、DNS 策略一起设计 |
| **sing-box / Clash 等 tun 模式** | 虚拟网卡 + 规则路由 | 与项目现有 **sing-box tun + nftables 默认拒绝** 一致 | 本地二进制需处理与 Docker 网络命名空间关系 |

**结论**：claude-shell 的「透明」对用户是 **一条 claude 命令**；对网络栈应是 **tun 级全局策略**，而非仅 `LD_PRELOAD`。与 proxychains 同类工具**互补而非替代**（后者更适合无 root 的单次命令实验）。

---

### 5. PATH / alias / shim 透明替换系统命令

**常见模式**

- **Shim 二进制**：同名可执行文件放在 **`PATH` 更前**；内核 `execve` 解析路径顺序。稳定、可版本共存。
- **Shell alias**：仅当前交互 shell，子进程/脚本常不继承，**不适合**作为唯一机制。
- **Wrapper 脚本**：易注入 env（如 `NODE_OPTIONS`），但 shebang 与可移植性需测。
- **distro 级**：`update-alternatives` 等，偏系统包管理，不适合单文件 CLI 分发。

**表功能**：文档说明「把目录放在 PATH 前」；可选安装器写入 `~/.local/bin` 等。

**差异化**：若 Go 二进制同时充当 **docker 编排器 + shim**，单一分发物更易叙事实；需注意与系统包管理器安装的 `claude` **冲突提示**。

---

### 6. Claude Code：遥测与数据流（官方 vs 未验证来源）

**HIGH confidence — [Anthropic Claude Code Data usage](https://docs.anthropic.com/en/docs/claude-code/data-usage)**

- **Statsig**：运营指标（延迟、可靠性、使用模式等）；文档写明 **不包含代码或文件路径**。TLS + 静态表述中的存储加密。退出：`DISABLE_TELEMETRY=1`。
- **Sentry**：错误日志。退出：`DISABLE_ERROR_REPORTING=1`。
- **`/feedback`**：会发送含代码的对话历史；退出：`DISABLE_FEEDBACK_COMMAND=1`。
- **会话满意度问卷**：仅数字评分；`CLAUDE_CODE_DISABLE_FEEDBACK_SURVEY=1` 或 broader：`CLAUDE_CODE_DISABLE_NONESSENTIAL_TRAFFIC`。
- **Bedrock/Vertex 等**：文档表格写明默认关闭部分遥测；以官方表格为准。

**LOW confidence — 第三方「源码解析」仓库、媒体报道**

- 可能存在与版本强绑定的实现细节；**不作为需求或合规依据**。若需深度列清单，应针对**具体版本**做静态或运行观测，并单独开 phase。

**对产品含义**：FEATURES 层应区分：**(A)** API 交互内容（必然经 Anthropic 政策约束）与 **(B)** Statsig/Sentry 等可关通道；容器内可通过 **环境变量 + 网络策略** 组合呈现「默认最省非必要外连」，并在文档中引用官方变量名。

---

### 7. Docker Desktop / Engine：访问宿主机（`host.docker.internal`、`host-gateway`）

**Docker Desktop（Mac/Win）**：[官方 Networking how-tos](https://docs.docker.com/desktop/networking/) 说明使用 **`host.docker.internal`** 解析到主机侧地址；另有 `gateway.docker.internal` 指向 Docker VM 网关。

**Linux Engine**：`host.docker.internal` **并非**在所有环境默认存在；常见做法为  
`--add-host=host.docker.internal:host-gateway`（Docker **20.10+** 引入特殊值 `host-gateway`，映射到宿主机网关 IP，常为 `docker0` 网段）。Compose 中为 `extra_hosts: ["host.docker.internal:host-gateway"]`。注意：**build 阶段**对 `host-gateway` 的支持与运行时不同，构建期若需访问主机常需静态 IP 或拆分阶段（见 [compose 讨论](https://github.com/docker/compose/issues/9768)）。

**与 PROJECT 对齐**：「本地流量经 host-gateway 回连宿主机」应写清：**目标 IP 段 + DNS 不走泄漏路径 + 与 tun 默认路由的优先级**，并在 `verify` 中可检查。

---

## Feature Dependencies（简图）

```
单一 claude 二进制 (PATH shim)
    → Docker 生命周期 (pull/create/start/exec)
        → 镜像内 Claude Code 安装与运行
            → sing-box tun + nftables（全出站）
                → 例外路由：RFC1918 / localhost → host-gateway
    → entrypoint：machine-id、proc 绑定、反粗检测
    → verify 子命令
```

---

## MVP Recommendation（仅 claude-shell）

**优先**

1. 可复现的容器内 Claude Code 启动路径（与隧道共存）。  
2. sing-box tun + nftables 默认拒绝 + 明确 DNS 策略。  
3. host-gateway / 私网路由验证（`verify`）。  
4. 系统级指纹最小集（与 PROJECT 一致）+ 文档化局限。

**可延后**

- garble 与 UI 级「艺术化」输出。  
- 对抗级反检测（超出粗粒度）。

---

## Sources

| 主题 | 来源 | Confidence |
|------|------|------------|
| Distrobox / Toolbx 目标与集成方式 | [Distrobox GitHub 文档](https://github.com/89luca89/distrobox)、[containers/toolbox README](https://github.com/containers/toolbox/) | HIGH |
| 容器检测启发式 | [Skyper 博客](https://blog.skyplabs.net/posts/container-detection/)、[Docker run metrics / cgroup](https://docs.docker.com/config/containers/runmetrics/)、社区 cgroup v2 讨论 | MEDIUM（启发式非标准） |
| proxychains / tun2socks 差异 | [ProxyChains-NG](https://github.com/rofl0r/proxychains)、[xjasonlyu/tun2socks](https://github.com/xjasonlyu/tun2socks)、社区对比文 | MEDIUM |
| Claude Code 遥测与退出 | [Anthropic Data usage](https://docs.anthropic.com/en/docs/claude-code/data-usage) | HIGH |
| host.docker.internal / host-gateway | [Docker Desktop networking](https://docs.docker.com/desktop/networking/)、[Moby host-gateway](https://github.com/moby/moby/pull/40007) 引入背景、Compose issue | HIGH-MEDIUM |
| systemd 容器接口 | [systemd.io CONTAINER_INTERFACE](https://systemd.io/CONTAINER_INTERFACE/) | HIGH |

---

## Gaps / 后续 Phase 专项

- 指定 **Claude Code + Bun** 版本矩阵下的实际外连列表（需运行抓包或查阅发行说明，随版本变）。  
- **Docker Desktop for Linux** 与纯 **Linux Engine** 行为差异的 CI 矩阵。  
- Apple Silicon / rootless Docker 下 **tun/nft** 能力边界。
