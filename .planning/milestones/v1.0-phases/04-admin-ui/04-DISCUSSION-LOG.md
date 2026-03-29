# Phase 4: 后台管理界面 - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-03-27
**Phase:** 04-后台管理界面
**Areas discussed:** 后台认证与权限模型, 界面布局与状态看板, 用户与凭证管理交互, 出口 IP 资源与绑定操作

---

## 后台认证与权限模型

| Option | Description | Selected |
|--------|-------------|----------|
| 单管理员 + JWT | 管理员凭证通过环境变量设定，登录后发 JWT，前端 Bearer Token 调 API | ✓ |
| 硬编码环境变量密码 | 最简单方案，一条环境变量搞定，不可审计 | |
| 多角色 RBAC | 灵活但 v1 过度设计 | |

**User's choice:** 单管理员 + JWT
**Notes:** v1 不做多角色划分，所有功能对已认证管理员全部开放。

---

## 界面布局与状态看板

| Option | Description | Selected |
|--------|-------------|----------|
| 侧边栏导航 + 顶栏 | 直观标准后台模式，扩展性好 | ✓ |
| 顶部 Tab 导航 | 简洁，适合功能少的场景 | |
| 单页仪表板 | 一览全局，但操作密集时信息过载 | |

**User's choice:** 侧边栏导航 + 顶栏
**Notes:** 侧边栏分为仪表板概览、用户管理、出口 IP 管理、主机管理、任务列表。仪表板展示关键计数和最近任务状态。

---

## 用户与凭证管理交互

### 创建用户密码方式

| Option | Description | Selected |
|--------|-------------|----------|
| 管理员手动设定初始密码 | 管理员创建时填写密码 | ✓ |
| 系统自动生成随机密码 | 系统生成并展示一次 | |

**User's choice:** 管理员手动设定初始密码

### 删除确认策略

| Option | Description | Selected |
|--------|-------------|----------|
| 弹窗确认 + 输入用户名确认 | 防止误操作的二次确认 | ✓ |
| 简单弹窗确认 | 仅一次确认 | |

**User's choice:** 弹窗确认 + 输入用户名确认

### 密码轮换方式

| Option | Description | Selected |
|--------|-------------|----------|
| 管理员输入新密码替换 | 手动指定新密码 | |
| 自动生成新密码并展示 | 点按钮自动生成 | ✓ |

**User's choice:** 自动生成新密码并展示

---

## 出口 IP 资源与绑定操作

### 创建出口 IP 界面形式

| Option | Description | Selected |
|--------|-------------|----------|
| 右侧抽屉表单 | 不离开列表页，操作流畅 | ✓ |
| 独立页面表单 | 全屏编辑 | |

**User's choice:** 右侧抽屉表单

### 绑定交互入口

| Option | Description | Selected |
|--------|-------------|----------|
| 在用户详情页选择出口 IP | 操作贴近业务对象 | ✓ |
| 在出口 IP 详情页绑定用户 | 从 IP 视角操作 | |
| 两侧都可以绑定 | 双向入口 | |

**User's choice:** 在用户详情页选择出口 IP

### 运行中主机换绑策略

| Option | Description | Selected |
|--------|-------------|----------|
| 必须先停机才能换绑 | 符合 Phase 2 禁止运行时换 IP 决策 | ✓ |
| 允许热替换 | 自动停机→换绑→重启 | |

**User's choice:** 必须先停机才能换绑

---

## Claude's Discretion

- UI 组件库选择
- 前端路由和状态管理策略
- JWT 具体实现参数
- 仪表板数据刷新策略
- 表格分页/排序/筛选细节

## Deferred Ideas

None — discussion stayed within phase scope.
