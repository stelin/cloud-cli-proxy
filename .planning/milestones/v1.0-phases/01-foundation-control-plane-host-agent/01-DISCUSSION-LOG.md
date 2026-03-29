# Phase 1: 基础控制面与主机代理 - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-03-26
**Phase:** 1-基础控制面与主机代理
**Areas discussed:** 控制面与主机代理边界, 用户主机生命周期模型, 受管镜像与持久化边界, 任务与状态模型

---

## 控制面与主机代理边界

### Q1 特权操作隔离级别

| Option | Description | Selected |
|--------|-------------|----------|
| 独立主机代理进程 | 控制面只写状态和下发任务，特权操作全部进 agent | ✓ |
| 控制面内独立 runtime 模块 | 代码分层但仍同进程执行特权动作 | |
| 先最小直连后续再拆 | 先快后补边界 | |

**User's choice:** 独立主机代理进程
**Notes:** 优先把特权边界做硬。

### Q2 控制面到 agent 交互方式

| Option | Description | Selected |
|--------|-------------|----------|
| 本机私有 API/RPC | 明确调用语义，便于鉴权与状态回传 | ✓ |
| 任务落库 + agent 轮询 | 实现直白，但实时性弱 | |
| 两者并用 | 能力更完整但更重 | |

**User's choice:** 本机私有 API/RPC
**Notes:** 作为主链路。

### Q3 Phase 1 的 agent 职责范围

| Option | Description | Selected |
|--------|-------------|----------|
| 只管运行时特权动作 | 容器生命周期与宿主机准备动作，网络强约束留 Phase 2 | ✓ |
| 运行时动作 + 网络接线骨架 | 提前预埋 Phase 2 骨架 | |
| 尽量全包未来宿主机操作 | 前瞻性高但范围膨胀 | |

**User's choice:** 只管运行时特权动作
**Notes:** 控制范围，避免 Phase 1 过重。

### Q4 agent 不可用时的处理

| Option | Description | Selected |
|--------|-------------|----------|
| 明确失败，不做降级 | 控制面保留记录，拒绝特权兜底 | ✓ |
| 只读可用，写操作失败 | 保留状态查看 | |
| 控制面临时兜底执行 | 可用性更高但边界被打穿 | |

**User's choice:** 明确失败，不做降级
**Notes:** 与硬边界策略一致。

---

## 用户主机生命周期模型

### Q1 用户主机模型

| Option | Description | Selected |
|--------|-------------|----------|
| 长期存在的一台主机 | 可启动/停止/重建，像受管云主机 | ✓ |
| 按需临时会话容器 | 每次进入新建，用完销毁 | |
| 两种都支持 | 灵活但复杂 | |

**User's choice:** 长期存在的一台主机
**Notes:** 保持“可管理可复用”的产品心智。

### Q2 初始状态策略

| Option | Description | Selected |
|--------|-------------|----------|
| 仅创建记录，不自动建容器 | 首次需要时再创建/拉起 | ✓ |
| 创建用户即建容器（停止态） | 提前准备但占资源 | |
| 创建用户即建并启动容器 | 资源成本过高 | |

**User's choice:** 仅创建记录，不自动建容器
**Notes:** 先控制资源使用。

### Q3 “重建”语义

| Option | Description | Selected |
|--------|-------------|----------|
| 重建同一台受管主机 | 销毁当前实例并按模板重建，保留身份/绑定 | ✓ |
| 新建替代主机并保留旧机 | 更安全但切换复杂 | |
| 先不细化 | 后续补定义 | |

**User's choice:** 重建同一台受管主机
**Notes:** 为后续运维动作保持一致语义。

### Q4 单用户主机数量

| Option | Description | Selected |
|--------|-------------|----------|
| 一用户一主机 | MVP 复杂度最低 | |
| 允许一用户多主机 | 更灵活但成本高 | |
| 当前一用户一主机，模型预留扩展 | 行为不变但设计可扩展 | ✓ |

**User's choice:** 当前一用户一主机，模型预留扩展
**Notes:** 兼顾 MVP 收敛与后续扩展。

---

## 受管镜像与持久化边界

### Q1 持久化范围

| Option | Description | Selected |
|--------|-------------|----------|
| 主目录持久化，系统层不持久化 | 兼顾主机感与可控性 | ✓ |
| 极少持久化 | 更干净但主机感弱 | |
| 几乎全状态保留 | 主机感强但一致性差 | |

**User's choice:** 主目录持久化，系统层不持久化
**Notes:** 与“受管主机”定位一致。

### Q2 基础能力放置位置

| Option | Description | Selected |
|--------|-------------|----------|
| 全部固化进基础镜像 | 满足开箱即用与稳定基线 | ✓ |
| 仅 OpenSSH 在镜像，其余启动时安装 | 启动慢且不稳定 | |
| 镜像极简，首次手工补环境 | 不符合产品目标 | |

**User's choice:** 全部固化进基础镜像
**Notes:** 明确满足 RUNT-02。

### Q3 重建时数据卷处理

| Option | Description | Selected |
|--------|-------------|----------|
| 保留主目录卷，重置系统层 | 默认安全且可复用 | |
| 全量清空（含用户数据） | 最干净但可复用性差 | |
| 双模式（默认保留 + 工厂重置） | 平衡默认体验与彻底恢复 | ✓ |

**User's choice:** 双模式（默认保留 + 工厂重置）
**Notes:** 后续需要明确触发入口与权限保护。

### Q4 镜像更新策略

| Option | Description | Selected |
|--------|-------------|----------|
| 固定版本并显式升级 | 可追溯、可回滚 | ✓ |
| 每次重建拉最新 | 不可预测 | |
| 稳定通道自动更新 | 隐式变化仍存在 | |

**User's choice:** 固定版本并显式升级
**Notes:** 优先稳定性与可审计性。

---

## 任务与状态模型

### Q1 动作执行模型

| Option | Description | Selected |
|--------|-------------|----------|
| 统一异步任务模型 | 控制面落任务，agent 执行并回写 | ✓ |
| 同步调用即时报错 | 快但后续返工概率高 | |
| 混合模式 | 容易引入一致性复杂度 | |

**User's choice:** 统一异步任务模型
**Notes:** 与独立 agent 架构一致。

### Q2 最小任务状态机

| Option | Description | Selected |
|--------|-------------|----------|
| pending/running/succeeded/failed + canceled | 可观测性与复杂度平衡 | ✓ |
| pending/running/done | 信息不足 | |
| 细粒度状态全量上 | Phase 1 过重 | |

**User's choice:** pending/running/succeeded/failed + canceled
**Notes:** 先落最小可用状态机。

### Q3 失败重试策略

| Option | Description | Selected |
|--------|-------------|----------|
| 默认不自动重试，人工重试 | 先保证故障可见 | ✓ |
| 固定次数自动重试 | 可能掩盖问题 | |
| 按任务类型差异化 | 建模成本高 | |

**User's choice:** 默认不自动重试，人工重试
**Notes:** 失败原因需结构化可检索。

### Q4 Phase 1 运维可见性

| Option | Description | Selected |
|--------|-------------|----------|
| 后台任务列表 + 最后错误摘要 | 满足基础排障 | ✓ |
| 仅日志查看 | 操作效率低 | |
| 完整观测台 | 超出 Phase 1 重心 | |

**User's choice:** 后台任务列表 + 最后错误摘要
**Notes:** 完整观测能力可在后续阶段增强。

---

## the agent's Discretion

- 任务表与审计表的字段细节、索引、保留策略。
- 主机代理 API 的具体传输协议与序列化格式。
- 工厂重置的交互细节和权限确认流程。

## Deferred Ideas

None — discussion stayed within phase scope.
