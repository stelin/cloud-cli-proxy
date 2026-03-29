# Phase 1: 基础控制面与主机代理 - Context

**Gathered:** 2026-03-26
**Status:** Ready for planning

<domain>
## Phase Boundary

本阶段仅交付单宿主机控制面的基础能力、持久化存储、受管用户镜像，以及宿主机特权操作边界。重点是把控制面与主机代理的职责切开，并让用户主机生命周期可管理、可复用、可重建。隧道强制出网细节（Phase 2）、终端接入体验（Phase 3）和后台 UI 能力（Phase 4）不在本阶段范围。

</domain>

<decisions>
## Implementation Decisions

### 控制面与主机代理边界
- **D-01:** 采用独立主机代理进程，控制面不直接执行 Docker/网络/宿主机特权操作。
- **D-02:** 控制面与主机代理通过本机私有 API/RPC 通信，作为主链路。
- **D-03:** Phase 1 中主机代理只负责运行时特权动作（创建、启动、停止、重建与宿主机准备动作）；隧道网络强约束细节放到 Phase 2。
- **D-04:** 主机代理不可用时明确失败并返回可诊断错误，不做控制面特权兜底。

### 用户主机生命周期模型
- **D-05:** 每个用户默认对应一台长期受管主机，不采用按会话临时创建的模型。
- **D-06:** 用户创建后先写入业务记录，不自动创建容器实例。
- **D-07:** “重建”语义为销毁当前实例并按模板重建同一台受管主机，保留身份与绑定关系。
- **D-08:** v1 行为保持“一用户一主机”，但数据模型预留未来“一用户多主机”扩展能力。

### 受管镜像与持久化边界
- **D-09:** 默认持久化用户主目录，系统层状态不持久化。
- **D-10:** OpenSSH、基础 Shell 工具与 `claude code` 全部固化在受管基础镜像中。
- **D-11:** 重建提供双模式：默认“保留主目录并重置系统层”，可选“工厂重置（全量清空）”。
- **D-12:** 镜像版本固定（digest/tag pin），升级通过显式后台动作触发，不在重建时隐式拉取最新。

### 任务与状态模型
- **D-13:** 创建/启动/停止/重建统一采用异步任务模型：控制面写任务，主机代理执行并回写状态。
- **D-14:** Phase 1 最小任务状态机为 `pending -> running -> succeeded|failed`，并支持 `canceled`。
- **D-15:** 默认不自动重试，失败原因结构化记录，由运营手动重试。
- **D-16:** Phase 1 先保证后台可见“任务列表 + 最后错误摘要”，不做完整任务观测台。

### the agent's Discretion
- 任务与事件的具体字段设计、索引策略和归档策略。
- 主机代理 API 的具体传输实现（在满足“本机私有接口”约束前提下）。
- “工厂重置”入口形态与安全确认交互细节。

</decisions>

<specifics>
## Specific Ideas

- 产品心智应是“可管理、可复用、可回收的云主机”，而不是一次性临时容器。
- 特权边界遵循“宁可失败，不可打穿”的原则：agent 异常时拒绝执行高权限动作。

</specifics>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### 阶段范围与成功标准
- `.planning/ROADMAP.md` — Phase 1 的目标、成功标准与 01-01/01-02/01-03 计划拆分。

### 需求约束
- `.planning/REQUIREMENTS.md` — `RUNT-01` 与 `RUNT-02` 的定义和 v1 全局约束映射。

### 产品原则与非功能约束
- `.planning/PROJECT.md` — 单宿主机、SSH-only、受控出网、开箱即用等核心产品边界。

### 架构边界
- `.planning/codebase/ARCHITECTURE.md` — 控制面/宿主机代理/用户容器的职责分段与特权隔离原则。

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- 现有 GSD 工具链（`.codex/get-shit-done/bin/gsd-tools.cjs` 与 `bin/lib/*.cjs`）已覆盖阶段初始化、状态记录与文档提交流程，可直接复用为工程推进骨架。
- `.planning/codebase/ARCHITECTURE.md` 已给出目标拓扑，可直接作为 Phase 1 设计输入。

### Established Patterns
- 仓库当前以规划与工作流代码为主，业务运行时代码尚未落地，Phase 1 为 greenfield 实现。
- 项目采用文档驱动交付路径：`ROADMAP -> CONTEXT -> PLAN -> EXECUTE`。

### Integration Points
- 本阶段产出的控制面服务、主机代理、持久化 schema 会成为 Phase 2~5 的共享依赖。
- 本阶段定义的任务状态与错误摘要会直接喂给后续后台管理可视化能力。

</code_context>

<deferred>
## Deferred Ideas

None — discussion stayed within phase scope.

</deferred>

---

*Phase: 01-foundation-control-plane-host-agent*
*Context gathered: 2026-03-26*
