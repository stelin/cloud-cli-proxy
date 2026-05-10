# Phase 12: 用户自助 API 与前端路由 - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-03-29
**Phase:** 12-用户自助 API 与前端路由
**Mode:** auto (all decisions auto-selected with recommended defaults)
**Areas discussed:** API 路由设计, 面板布局与信息展示, 主机重建交互, 出口 IP 展示方式

---

## API 路由设计

| Option | Description | Selected |
|--------|-------------|----------|
| /v1/user/ 前缀 + JWT user_id 过滤 | 与 /v1/admin/ 对称，复用 AuthMiddleware + UserIDFromContext | ✓ |
| /v1/portal/ 前缀 | 语义上匹配前端路由名，但与后端命名不一致 | |
| 复用 /v1/admin/ 端点加角色过滤 | 减少代码但模糊了权限边界 | |

**User's choice:** [auto] /v1/user/ 前缀 + JWT user_id 过滤 (recommended default)
**Notes:** 已有 ListHostsByUserID、GetEgressIPByHost 查询可直接复用，userGuard 允许 admin 和 user 角色

---

## 面板布局与信息展示

| Option | Description | Selected |
|--------|-------------|----------|
| 主机列表卡片视图 | 每张卡片含主机名、状态、出口 IP、创建时间 | ✓ |
| 简单表格视图 | 列表形式展示，信息密度高 | |
| 仪表盘概览 + 列表 | 顶部统计数据 + 下方列表，功能偏多 | |

**User's choice:** [auto] 主机列表卡片视图 (recommended default)
**Notes:** 用户功能较少，卡片视图更直观友好；复用 _portal.tsx 布局，不需要侧边栏

---

## 主机重建交互

| Option | Description | Selected |
|--------|-------------|----------|
| 确认对话框 + 任务进度 | 点击重建→确认警告→排队任务→显示进度 | ✓ |
| 直接重建无确认 | 简化流程但有误操作风险 | |
| 两步确认（输入主机名） | 更安全但对用户要求过高 | |

**User's choice:** [auto] 确认对话框 + 任务进度 (recommended default)
**Notes:** 复用 QueueHostAction 排队机制，前端通过状态轮询反馈进度

---

## 出口 IP 展示方式

| Option | Description | Selected |
|--------|-------------|----------|
| 详情页只读展示 IP + 隧道类型 | 主机详情页显示绑定信息，列表卡片简要展示 IP | ✓ |
| 独立出口 IP 页面 | 单独页面展示所有出口 IP，对 v1.2 来说过重 | |
| 仅在卡片中展示 | 不设详情页，信息全在卡片中 | |

**User's choice:** [auto] 详情页只读展示 IP + 隧道类型 (recommended default)
**Notes:** 用 GetEgressIPByHost 查询，用户无需看到代理配置细节

---

## Claude's Discretion

- 主机卡片样式、状态 badge 颜色、空状态展示
- 主机详情页布局
- 重建进度轮询间隔

## Deferred Ideas

- Claude 账号展示 — Phase 13
- KasmVNC 远程桌面入口 — Phase 14
