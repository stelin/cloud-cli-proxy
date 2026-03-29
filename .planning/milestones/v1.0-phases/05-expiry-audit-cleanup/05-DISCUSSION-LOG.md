# Phase 5: 到期、审计与清理 - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-03-27
**Phase:** 05-到期、审计与清理
**Mode:** auto
**Areas discussed:** 到期模型与策略, 过期主机处理策略, 事件记录范围与分类, 后台审计展示, 对账与漂移检测

---

## 到期模型与策略

| Option | Description | Selected |
|--------|-------------|----------|
| users 表新增 expires_at TIMESTAMPTZ NULL | NULL 表示永不过期，最简单且兼容现有数据 | ✓ |
| 新增独立 user_expiry 表 | 支持到期历史，但增加复杂度 | |
| 仅通过 status 字段手动管理 | 不满足 LIFE-04 自动到期要求 | |

**User's choice:** [auto] users 表新增 expires_at TIMESTAMPTZ NULL (recommended default)
**Notes:** 与现有 bootstrap 中 `expired` 状态拦截无缝衔接，迁移成本最低。

| Option | Description | Selected |
|--------|-------------|----------|
| 控制面后台定时 goroutine 周期扫描 | 简单可靠，与控制面同生命周期 | ✓ |
| 数据库触发器自动更新 | 减少应用层逻辑，但增加 DB 层复杂度 | |
| 仅在认证时检查 | 不满足"运行中主机按策略被处理"要求 | |

**User's choice:** [auto] 控制面后台定时 goroutine 周期扫描 (recommended default)
**Notes:** 可配置扫描间隔，与控制面主进程同生命周期管理。

| Option | Description | Selected |
|--------|-------------|----------|
| 管理 API 和后台 UI 均支持 | 完整满足 LIFE-04 | ✓ |
| 仅 API 支持 | 不符合 Phase 4 建立的后台管理模式 | |

**User's choice:** [auto] 管理 API 和后台 UI 均支持 (recommended default)
**Notes:** 管理员可设置、修改和清除到期时间，也可重新激活已过期用户。

---

## 过期主机处理策略

| Option | Description | Selected |
|--------|-------------|----------|
| 到期时自动下发 stop 动作 | 符合成功标准，利用现有 QueueHostAction | ✓ |
| 仅标记用户过期，主机不处理 | 不满足"运行中主机按策略被处理"要求 | |
| 提供宽限期后再停止 | 增加复杂度，v1 不必要 | |

**User's choice:** [auto] 到期时自动下发 stop 动作 (recommended default)
**Notes:** 复用现有 `QueueHostAction(ActionStopHost)` 链路。

| Option | Description | Selected |
|--------|-------------|----------|
| v1 不做宽限期，到期即执行 | 保持简单，符合 MVP 约束 | ✓ |
| 提供可配置宽限期 | 增加策略复杂度 | |

**User's choice:** [auto] v1 不做宽限期，到期即执行 (recommended default)
**Notes:** 与"宁可失败，不可打穿"原则一致。

---

## 事件记录范围与分类

| Option | Description | Selected |
|--------|-------------|----------|
| 覆盖 auth/user/host/admin/reconcile 多类事件 | 完整满足 ADMN-04，与现有 worker 事件并存 | ✓ |
| 仅记录认证和到期事件 | 不满足生命周期和管理员操作审计要求 | |
| 新增独立审计表 | 增加 schema 复杂度，与现有 events 表重复 | |

**User's choice:** [auto] 覆盖 auth/user/host/admin/reconcile 多类事件 (recommended default)
**Notes:** 新增 13 种事件类型，沿用现有 RecordEvent + JSONB metadata 模式。

| Option | Description | Selected |
|--------|-------------|----------|
| 沿用 JSONB metadata，统一携带 user_id/operator | 与 worker 中已有模式一致 | ✓ |
| 引入结构化事件 schema | 增加开发成本，v1 不必要 | |

**User's choice:** [auto] 沿用 JSONB metadata，统一携带 user_id/operator (recommended default)
**Notes:** 灵活且与现有事件记录行为完全兼容。

---

## 后台审计展示

| Option | Description | Selected |
|--------|-------------|----------|
| 侧栏新增事件日志入口，全局时间线 | 独立于任务列表，与侧栏结构一致 | ✓ |
| 合并到任务列表页 | 混淆任务和事件概念 | |
| 仅 API 暴露，不做前端 | 不符合"可查看"要求 | |

**User's choice:** [auto] 侧栏新增事件日志入口，全局时间线 (recommended default)
**Notes:** 支持按类型、用户、主机、时间范围筛选。

---

## 对账与漂移检测

| Option | Description | Selected |
|--------|-------------|----------|
| 控制面定时扫描，通过 host-agent 查询容器状态 | 遵循特权边界，利用现有通道 | ✓ |
| 控制面直接 Docker API 查询 | 违反 Phase 1 特权隔离决策 | |
| 仅手动触发对账 | 不满足"不手工 SSH 排障"要求 | |

**User's choice:** [auto] 控制面定时扫描，通过 host-agent 查询容器状态 (recommended default)
**Notes:** 不一致时记录事件并修正 DB 状态。

| Option | Description | Selected |
|--------|-------------|----------|
| 超时任务自动标记 failed | 利用现有 ListPendingTasks 和 UpdateTaskStatus | ✓ |
| 仅告警不自动处理 | 需要人工干预，不满足对账自动化要求 | |

**User's choice:** [auto] 超时任务自动标记 failed (recommended default)
**Notes:** 可配置超时阈值，默认 10 分钟。

---

## Claude's Discretion

- 定时器间隔参数和配置方式
- 事件列表页分页策略
- 对账扫描并发策略和批量大小
- 事件 metadata 字段命名约定
- 仪表板事件摘要卡片展示条数

## Deferred Ideas

None — discussion stayed within phase scope.
