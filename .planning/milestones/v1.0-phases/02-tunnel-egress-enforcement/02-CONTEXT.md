# Phase 2: 隧道出网强制层 - Context

**Gathered:** 2026-03-26
**Status:** Ready for planning

<domain>
## Phase Boundary

本阶段只交付出口 IP 绑定、受控隧道接线、DNS 受控路径和泄漏校验能力，确保每个进入运行态的用户主机都只能通过指定出口 IP 出网。后台管理界面、SSH 接入体验、多宿主机调度和更复杂的 IP 运营策略不属于本阶段范围。

</domain>

<decisions>
## Implementation Decisions

### 出口 IP 绑定与资源语义
- **D-01:** v1 运行时采用“单主机单活跃出口 IP”语义；每台主机在任意时刻只允许一个生效出口 IP。
- **D-02:** 同一个出口 IP 可以在后台被分配给多个账号或主机，并允许这些主机同时运行、同时共用该出口 IP；唯一性不在 IP 资源层，而在“每台主机运行时只认一个活跃出口”这一层。
- **D-03:** 每台主机都绑定一个独立的 `claude` 账号，出口 IP 漂移会触发账号风控，因此禁止运行时隐式换 IP、自动漂移或回落到其他出口。

### 启动失败与恢复策略
- **D-04:** 启动前若缺少出口 IP 绑定、隧道拉起失败、出口 IP 校验失败、DNS 校验失败或泄漏阻断失败，任务必须立即失败，不做自动重试。
- **D-05:** Phase 2 不允许为同一主机自动切换到其他出口 IP，也不允许在失败后自动执行“更重”的恢复动作；恢复必须依赖显式运维或后续任务重试。
- **D-06:** 网络准备与强校验放在真正 `start` / 进入可运行态之前执行，不在单纯 `create` 阶段提前把网络视为就绪。

### DNS 与受控路径约束
- **D-07:** DNS 必须跟着当前活跃出口 IP 的受控隧道一起走，不能使用宿主机解析器、Docker 默认解析器或任何与该出口路径分离的解析方案。
- **D-08:** “出口 IP 路径”和“DNS 路径”视为同一个网络契约的一部分；如果两者之一不符合绑定结果，就视为整台主机网络未就绪。

### 运行态异常与就绪门槛
- **D-09:** 主机在被标记为可用前，至少必须通过三类强校验：出口 IP 与绑定结果一致、DNS 解析走受控路径、非隧道直连出站被默认阻断。
- **D-10:** 运行中的主机一旦发现当前出口 IP 不可达、检测结果变化、DNS 脱离受控路径，或出现任何未配置出口的旁路，这台主机必须立刻标记为失败/异常，直到出口恢复并重新通过校验前都不能视为可用。
- **D-11:** “除了已配置出口外不允许存在其他任何出口路径”是本阶段最高优先级约束，高于启动速度、自动恢复和资源利用率，因为出口变化会直接带来 `claude` 账号风控风险。

### 运维可见性与记录粒度
- **D-12:** 任务和事件记录必须按网络失败类型细分，至少区分：绑定缺失、出口 IP 不匹配、DNS 不走受控路径、非隧道直连阻断失效、出口 IP 当前不可达。
- **D-13:** Phase 2 先按失败类型细分记录，不额外引入更复杂的危急等级模型；但后续阶段应能基于这些细分原因继续增强后台展示和告警。

### the agent's Discretion
- 具体采用 `WireGuard` 原生接线、辅助守护进程还是其他兼容实现，只要满足“受控隧道 + netns + 默认拒绝”的项目硬约束即可。
- 具体使用 `nftables` 还是 `iptables`，以及规则组织形式，只要遵守 Docker 官方规则链模型并保持默认拒绝即可。
- 出口校验、DNS 校验和泄漏阻断测试的具体命令、探测端点和重用脚本结构。

</decisions>

<specifics>
## Specific Ideas

- 用户明确要求：每台主机都绑定一个独立的 `claude` 账号，账号对出口 IP 变化高度敏感。
- 用户明确要求：除了管理员配置的出口外，不允许主机存在其他任何出站路径。
- 用户明确要求：一旦当前出口 IP 挂掉、检测值变化或不再符合绑定结果，主机要立刻失败，直到出口恢复并重新校验通过。

</specifics>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### 阶段范围与需求约束
- `.planning/ROADMAP.md` — Phase 2 的目标、成功标准与 `02-01/02-02/02-03` 计划拆分。
- `.planning/REQUIREMENTS.md` — `NET-01` 到 `NET-05` 的正式定义，以及 Phase 2 在 v1 需求中的映射。
- `.planning/PROJECT.md` — 单宿主机、SSH-only、全隧道路由、出口 IP 绑定不可缺失等产品级边界。
- `.planning/STATE.md` — 当前阶段焦点与从 Phase 1 继承的近期决策。

### 既有架构与前置阶段约束
- `.planning/phases/01-foundation-control-plane-host-agent/01-CONTEXT.md` — Phase 1 已锁定的特权边界、任务模型、失败策略与 `internal/network.Provider` 预留接口。
- `.planning/codebase/ARCHITECTURE.md` — 控制面 / host-agent / 用户容器的职责分段，以及“用户容器默认出网必须被隧道接管”的硬边界。
- `.planning/research/ARCHITECTURE.md` — 推荐采用网络管理器负责 `netns`、隧道、DNS 与防火墙约束的整体结构。

### 网络实现方向与风险依据
- `.planning/research/SUMMARY.md` — Phase 2 作为“产品承诺最核心的一层”的研究结论，以及 `WireGuard + netns` 的推荐方向。
- `.planning/research/STACK.md` — `WireGuard + Linux netns`、`nftables/iptables` 与 Docker 网络能力的推荐技术栈选择。
- `.planning/research/PITFALLS.md` — Docker 普通网络逃逸、DNS 泄漏、默认路由旁路等 Phase 2 关键坑点与规避方式。
- `CLAUDE.md` — 项目级技术栈、明确不推荐方案和架构/网络硬约束汇总。

### 运行时与镜像边界
- `deploy/docker/managed-user/README.md` — 受管用户镜像的运行时约定，以及“Phase 2 只在模板旁新增网络准备钩子接口”的约束。

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- `internal/network/provider.go`：已经定义 `Provider` 接口和 `PrepareHost` / `CleanupHost` 钩子，Phase 2 可以直接替换 `NoopProvider` 为真实网络实现。
- `internal/store/migrations/0001_initial.sql`：已经具备 `egress_ips` 与 `host_egress_bindings` 表，可直接承接出口资源与绑定关系。
- `internal/store/repository/queries.go`：已有 `ListHostBindings`，能作为启动前绑定校验与后续任务编排的读取入口。
- `deploy/scripts/host-preflight.sh`：已检查 `docker`、`ip`、`nft|iptables`、`systemctl` 等宿主机能力，可继续扩展为 Phase 2 的基础前置检查。

### Established Patterns
- 高权限网络动作必须继续留在 `host-agent`，控制面只通过本机私有接口下发任务，不能直接持有 Docker / 网络特权。
- 任务默认不做自动重试；失败要结构化记录并暴露给运维，而不是静默恢复。
- 项目已经明确偏向 `WireGuard + netns` 的命名空间优先模型，不接受 Docker 默认 bridge 或应用层代理作为主生产路径。

### Integration Points
- `internal/runtime/tasks/worker.go` 当前会在 `createHost` 中调用 `provider.PrepareHost`；由于本阶段决定“网络准备放在 start 前”，规划阶段需要评估是否调整调用时机或拆出更清晰的准备步骤。
- `internal/agent/server.go` 已经把 host action 执行与任务状态回写接好，Phase 2 可以沿用该通道扩展网络错误类型和事件记录。
- `tasks` / `events` 表与现有任务状态机会直接承载 Phase 2 的网络失败摘要、校验结果和异常状态留痕。

</code_context>

<deferred>
## Deferred Ideas

None — discussion stayed within phase scope.

</deferred>

---

*Phase: 02-tunnel-egress-enforcement*
*Context gathered: 2026-03-26*
