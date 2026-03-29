# Phase 1: 基础控制面与主机代理 - Research

**Researched:** 2026-03-26
**Domain:** 单宿主机控制面骨架与宿主机代理协调
**Confidence:** HIGH

<user_constraints>
## User Constraints (from CONTEXT.md)

**CRITICAL:** 以下决策来自 `01-CONTEXT.md`，后续规划不得违背。

### Locked Decisions
- **D-01~D-04:** 采用独立主机代理进程，控制面不到宿主机层，私有 RPC 负责 Docker、网络、特权操作，失败时明确报错。
- **D-05~D-08:** 用户默认一台长期受管主机，创建时只写记录，重建语义为销毁并按模板重建，数据模型预留多主机扩展。
- **D-09~D-12:** 默认持久化主目录，系统层由受管镜像（含 OpenSSH、基础 Shell、`claude code`）承载；重建可选保留主目录或工厂重置，镜像版本用 digest/tag pin，升级需显式触发。
- **D-13~D-16:** 所有操作走异步任务：控制面写任务、Agent 执行、状态 `pending -> running -> succeeded|failed`（含 `canceled`），不自动重试，任务列表与错误摘要即可满足 v1 可见性。

### the agent's Discretion
- 任务/事件字段、索引与归档策略的具体设计。
- 主机代理 API 的通信细节（只要是本机私有接口即可）。
- “工厂重置”的入口与安全确认交互形式。

### Deferred Ideas (OUT OF SCOPE)
- 无；本阶段研究在既定范围内。
</user_constraints>

<research_summary>
## Summary

本阶段需要掌握的核心是：控制面用 PostgreSQL 维护用户、主机、任务、出口 IP 和事件的真实状态；宿主机代理扮演唯一可以触碰 Docker/网络/特权的角色；异步任务模型为启动/停止/重建提供可观察性；受管镜像（含 OpenSSH 与 `claude code`）加上恒定版本 pin 保证可重建性。以上结论来自 `01-CONTEXT`、`PROJECT`、`ARCHITECTURE` 与各类研究文档，确保控制面骨架、持久层和特权边界在 Phase 1 内可交付。

要做好的标准路径是：先用 Go 搭控制面服务（`cmd/control-plane`）与主机 Agent（`cmd/host-agent`），用 PostgreSQL 18.3 保存状态，再让 Agent 通过 Docker 28.x 和 netns/WireGuard 接管网络，最终通过异步任务回写任务状态与错误摘要，提供基本的后台可视化。这个流程将 Phase 1 的控制面与受管主机职责彻底分离，并为 Phase 2~3 留出都可复用的契约。

**Primary recommendation:** 先把控制面 + PostgreSQL 状态 + 宿主机代理高权限边界搭起来，再用异步任务映射启动/重建/停止流程，把受管镜像版本、主目录持久化和错误摘要都钉死在 Phase 1 的交付之中。
</research_summary>

<standard_stack>
## Standard Stack

### Core
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| Go | 1.26.1 | 控制面 API、任务编排、主机 Agent | 官方稳定补丁版本，适合 single-binary 运维系统与 network/ process control |
| PostgreSQL | 18.3 | 保存用户、主机、出口 IP、任务与审计事件 | 事务强、成熟、适合运营数据 |
| Docker Engine | 28.x | 运行用户容器、镜像管理、提供 API | 当前主线版本，净化网络控制能力 |
| OpenSSH | 10.2p1 | 容器内 SSH Server | 标准 SSH 兼容，满足 `curl -> SSH` 流程 |
| WireGuard + Linux netns | 最新稳定 | 全隧道出口 IP 控制 | 允许把隧道接口移入 namespace，彻底接管默认路由 |

### Supporting
| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| pgx | v5.x | PostgreSQL 驱动 | 后端落地用户 / 主机 / 任务表时直接使用 |
| nftables / iptables | 发行版版本 | 默认拒绝路由、细粒度防火墙 | 网络管理模块接管 namespace 时需配置 |
| systemd timers | 宿主机版本 | 运行控制面 + 到期治理等守护 | 保持 Phase 1 自启动、可恢复 |

### Alternatives Considered
| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| Go 1.26.1 | Node.js 后端 | 语言统一但得额外处理进程管理与网络控制 |
| PostgreSQL | SQLite | 轻量但缺事务与审计，难以支撑到期治理 |
| WireGuard/netns | 应用层代理（比如 HTTP 代理） | 无法防住 DNS|TCP|UDP 等全流量泄漏 |

**Installation:**
```bash
# 后端依赖
go env -w GO111MODULE=on
go install ./cmd/control-plane
go install ./cmd/host-agent
# PostgreSQL 18.3 直接从官方 repo 安装
```
</standard_stack>

<architecture_patterns>
## Architecture Patterns

### Recommended Project Structure
```
cmd/
├── control-plane/        # 控制面主程序
└── host-agent/           # 宿主机特权代理

internal/
├── api/                  # HTTP 处理器与中间件
├── auth/                 # 认证与凭证逻辑
├── bootstrap/            # `curl` 启动入口与终端输出
├── containers/           # Docker 集成与镜像模板管理
├── network/              # netns、WireGuard、DNS 与防火墙约束
├── sessions/             # 启动任务与就绪状态机
├── users/                # 用户生命周期领域逻辑
├── expiry/               # 到期治理与校验机制
├── audit/                # 事件记录与查询
└── store/                 # PostgreSQL 数据访问层

web/
├── admin/                # React + Vite 后台管理
└── shared/                # 共享类型、客户端生成、API stub

deploy/
├── docker/               # 受管镜像与基础镜像定义
├── systemd/              # control-plane + host-agent 服务单位
└── scripts/               # 部署、备份、维护脚本
```

### Pattern 1：控制面 + 宿主机 Agent 边界
**What:** 控制面只处理用户/任务/状态，Agent 负责 Docker、网络、特权动作。
**When to use:** Phase 1 就必须启用，避免 Web/API 持有太多特权。
**Example:** 控制面通过本地 RPC 命令让 Agent 创建容器并加入全隧道 namespace。

### Pattern 2：异步任务状态机
**What:** 所有生命周期操作都通过任务记录，状态从 `pending` 走向 `succeeded|failed`，包括 `canceled`，并记录失败原因。
**When to use:** 启动/停止/重建、后台生命周期操作。
**Example:** 任务写入 PostgreSQL，然后由 Agent 拉取并更新状态，控制面监听并输出错误摘要。

### Pattern 3：命名空间优先的隧道网络
**What:** 把隧道网络当成容器的默认路由，DNS、TCP、UDP 全面走该路径。
**When to use:** Phase 1 搭建受控镜像时即要准备 nets；Phase 2 细化出口 IP 绑定。
**Example:** Agent 为容器创建专用 netns、配置 WireGuard 接口、配置默认路由与 DNS，并在启动检查前验证出口 IP。

### Anti-Patterns to Avoid
- **控制面直接操作 Docker 或 iptables：** 安全半径过大，不利于审计，Phase 1 必须抽出 Agent。
- **跳过任务状态回写：** 会让启动失败无法诊断，用户和运营都无法可靠重试。
- **让容器自己决定默认路由：** 会留下 DNS/流量泄漏隐患。
</architecture_patterns>

<dont_hand_roll>
## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| 全流量隧道 | 自己写一个用户态代理转发 | WireGuard + netns | 代理难覆盖 DNS/UDP，netns 可以让隧道彻底成为默认路由 |
| 任务观测台 | 直接把日志拼凑在 shell 输出 | PostgreSQL 任务表 + 聚合视图 | 结构化状态便于后台展示与重试 |
| 受管镜像管理 | 在部署脚本里动态拼接镜像 | 固定 digest/tag + 定期后台升级 | 固定版本保证可重现性，后台升级可用后审计 |

**Key insight:** Phase 1 目标是把控制面、PostgreSQL、受管镜像、宿主机 Agent 这几块的边界钉死，任何自建的代理/调度逻辑都会导致调试难度、洩漏风险与不可重复的部署，别自造轮子。
</dont_hand_roll>

<common_pitfalls>
## Common Pitfalls

### Pitfall 1：Docker 默认网络泄漏
**What goes wrong:** 容器保留默认 bridge 或其他接口，流量绕过出口 IP。
**Why it happens:** 控制面认为只要绑定 metadata 就完成，忽略了 netns 默认路由。
**How to avoid:** Agent 在启动阶段必须配置 WireGuard/netns，出站流量未通过验证就不能标记就绪。
**Warning signs:** 容器的 `ip route` 显示多条 default，出口验证显示宿主机 IP。

### Pitfall 2：任务状态没有结构化失败原因
**What goes wrong:** 失败后的 `curl` 体验只有“启动失败”，运营看不到根因。
**Why it happens:** 只记录 `failed` 状态而不写 `error_reason`。
**How to avoid:** 每次 agent 都要把 `error_code`/`error_message` 回写 PostgreSQL，并在后台显示最后一个错误摘要。
**Warning signs:** 后台错误日志空空如也，却有大量 `failed` 的任务。

### Pitfall 3：工厂重建留下残余状态
**What goes wrong:** 重建之后残留旧容器或网络接口，导致后续启动失败。
**Why it happens:** Agent 重建时只删容器但不清理 netns 或任务状态。
**How to avoid:** 重建任务要清理 Docker container、网络接口、任务与 Agent 本地状态，再重新创建镜像模板实例。
**Warning signs:** `docker ps -a` 看到旧容器、netns 仍存在、任务日志显示数据冲突。
</common_pitfalls>

<code_examples>
## Code Examples

### 异步任务状态流
```go
type TaskStatus string

const (
    TaskPending   TaskStatus = "pending"
    TaskRunning   TaskStatus = "running"
    TaskSucceeded TaskStatus = "succeeded"
    TaskFailed    TaskStatus = "failed"
    TaskCanceled  TaskStatus = "canceled"
)

type Task struct {
    ID          uuid.UUID
    UserID      uuid.UUID
    Kind        string // create|stop|rebuild
    Status      TaskStatus
    ErrorCode   string
    ErrorDetail string
    UpdatedAt   time.Time
}

func (t *Task) Transition(next TaskStatus) {
    // 在 control-plane 里校验状态机并写入 PostgreSQL
}
```

### Agent 网络就绪检查示意
```go
func agentCheckNetwork(ctx context.Context, containerID string) error {
    if !wireguardIsUp(containerNetns(containerID)) {
        return errors.New("wireguard不在容器 netns 中")
    }
    if !dnsPointsToTunnel(containerID) {
        return errors.New("DNS 未走受控隧道")
    }
    return nil
}
```

### 重建流程简化伪代码
```go
func rebuild(task Task) error {
    if err := agent.DestroyContainer(task.ContainerID); err != nil {
        return err
    }
    if err := agent.CleanNetns(task.ContainerID); err != nil {
        return err
    }
    return agent.CreateManagedContainer(task.TemplateDigest)
}
```
</code_examples>

<sota_updates>
## State of the Art (2024-2025)

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| 把 Docker 默认 network 作为出口 | 用 netns + WireGuard 确保 default route 被替换 | 2025-2026 | 彻底防止 DNS 与 UDP 绕过 |
| 控制面直接操作所有特权 | 控制面驱动特权 Agent | 2025-2026 | 降低攻击面、方便未来多宿主机扩展 |
| 只有任务状态，没有错误详情 | 结构化任务失败原因并展示 | 2025 | 提升运营排障效率 |

**New tools/patterns to consider:**
- 使用 Cron/systemd timers 触发任务回收和到期治理，Phase 1 即可复用。
- 把 PostgreSQL 任务表作为信息源，为 Phase 4 UI 与后续验证提供统一视图。

**Deprecated/outdated:**
- 让容器自行决定 DNS/default-route：Phase 1 已针对 `netns + WireGuard` 做出强约束，任何回退都会引入洩漏风险。
</sota_updates>

## Validation Architecture

Phase 1 的验证应围绕以下架构要素：
- **控制面骨架与 PostgreSQL schema：** 验证 `users`、`hosts`、`tasks`、`bindings`、`events` 表完整且支持到期、任务状态与错误日志，确保读写路径成熟。（参见 `REQUIREMENTS` 中 RUNT-01/RUNT-02 关联需求）
- **受管镜像 + 主目录持久化：** 用固定 digest 的镜像启动容器，验证 `OpenSSH`、`claude code` 在镜像中可用，并在重建后可选保留主目录。
- **主机 Agent 高权限边界：** 验证 Agent 负责 Docker 创建/销毁、netns+WireGuard 配置、`nftables` 默认拒绝；控制面只通过本地 RPC 下发任务。
- **异步任务模型：** 验证任务从 `pending` 到 `running` 再到 `succeeded|failed`，失败时写入 `error_code` 和 `error_message`，任务列表展示最后失败摘要且支持 `canceled`。
- **数据回写与观察：** 模拟 Agent 失败时控制面能读取 PostgreSQL 状态并向前端/启动流程展示最后错误（Phase 1 先做基本任务列表 + 错误摘要）。

以上验证结果会成为后续 VALIDATION.md 的基础，Phase 2 只需在此基础上补充出口 IP 与 DNS 全隧道校验。

## Open Questions

1. **隧道 provide 选择与 Agent 接口：** 已知要用 WireGuard/netns，但具体是自己维护 WireGuard 接口还是调用现成隧道守护进程尚不确定。建议 Phase 1 先抽象接口，后续 Phase 2 实现时再填具体 provider。
2. **PostgreSQL 备份节奏：** Phase 1 需保证持久层可恢复，但还未规划备份策略，建议在控制面稳定后明确备份、恢复步骤。
3. **工厂重置的触发与安全确认：** 需求点在 `01-CONTEXT` 的 agent discretion 中，Phase 1 可以先做 CLI/后台入口，后续再补安全确认方案。

<sources>
## Sources

### Primary (HIGH confidence)
- `.planning/phases/01-foundation-control-plane-host-agent/01-CONTEXT.md` — Phase 1 的锁定决策、agent discretion 与需求边界。
- `.planning/ROADMAP.md` — Phase 1 目标、成功标准与三条计划。
- `.planning/REQUIREMENTS.md` + `.planning/PROJECT.md` — RUNT 系列需求、核心产品原则与 v1 约束。
- `.planning/codebase/ARCHITECTURE.md` + `.planning/research/ARCHITECTURE.md` — 推荐的总体架构、组件职责、项目结构与模式。
- `.planning/research/STACK.md` — 核心与辅助技术栈、备选项与开发工具。
- `.planning/research/PITFALLS.md` — 常见坑、技术债类比与阶段映射。
- `.planning/research/SUMMARY.md` + `.planning/research/FEATURES.md` — 功能定位、关键验证与差异化建议。
</sources>

<metadata>
## Metadata

**Research scope:**
- Core technology: Control plane + host agent + PostgreSQL task state machine
- Ecosystem: Docker 28.x + OpenSSH 10.2p1 + WireGuard/netns 全隧道
- Patterns: 独立 Agent、异步任务、命名空间默认路由
- Pitfalls: 网络泄漏、任务可观察性、重建残留、权限边界

**Confidence breakdown:**
- Standard stack: HIGH — 基于 research/STACK 提供的版本 + 官方推荐路线。
- Architecture: HIGH — 多份研究文档一致推荐控制面 + Agent + netns。
- Pitfalls: HIGH — research/PITFALLS 中列出并标记给各阶段。
- Code examples: MEDIUM — 源于当前需求与异步任务模型的通用写法。

**Research date:** 2026-03-26
**Valid until:** 2026-04-25（稳定技术，30 天审查）
</metadata>

---
*Phase: 01-foundation-control-plane-host-agent*
*Research completed: 2026-03-26*
*Ready for planning: yes*
