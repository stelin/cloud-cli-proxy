# Phase 3: 启动入口与 SSH 接入 - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-03-27
**Phase:** 03-启动入口与 SSH 接入
**Areas discussed:** 启动入口与认证交互, 启动进度与状态反馈, SSH 就绪与交接, 失败映射与重试体验
**Mode:** auto

---

## 启动入口与认证交互

| Option | Description | Selected |
|--------|-------------|----------|
| 单一短 `curl` bootstrap + 终端内用户名/密码输入（推荐） | 对齐产品“一条命令启动”目标，脚本只做交互与 API 调用，控制面负责认证与编排 | ✓ |
| 安装本地 CLI 后再启动 | 需要额外安装步骤，破坏首用摩擦目标 | |
| 浏览器登录后回终端复制 token | 多步骤切换，偏离 SSH-only 终端主路径 | |

**User's choice:** auto-selected 推荐方案（单一短 `curl` bootstrap + 终端认证）
**Notes:** `[auto] 启动入口与认证交互 — 选择推荐默认：单一短 curl 入口，密码无回显，控制面统一鉴权。`

---

## 启动进度与状态反馈

| Option | Description | Selected |
|--------|-------------|----------|
| 轮询任务状态并映射为阶段化终端提示（推荐） | 复用现有任务状态机（pending/running/succeeded/failed），实现成本低且可观测 | ✓ |
| 长连接流式推送日志 | 交互更实时，但当前代码基线无 SSE/WebSocket，额外改造较大 | |
| 只显示“请稍候”静态提示 | 信息不足，失败时不利于定位与支持 | |

**User's choice:** auto-selected 推荐方案（任务轮询 + 阶段提示）
**Notes:** `[auto] 启动进度与状态反馈 — 选择推荐默认：轮询启动状态接口，输出分阶段进度。`

---

## SSH 就绪与交接

| Option | Description | Selected |
|--------|-------------|----------|
| 启动成功 + SSH readiness gate 通过后再返回并 `exec ssh`（推荐） | 满足 `RUNT-03`，确保用户进入的是可用会话而非“容器已起但 SSH 未就绪” | ✓ |
| 任务成功后立即给连接信息 | 风险是 SSH 服务尚未可用，用户体验不稳定 | |
| 仅返回手工连接参数让用户自己执行 ssh | 增加人工步骤，破坏“一条命令直达”体验 | |

**User's choice:** auto-selected 推荐方案（就绪门槛后自动 ssh handoff）
**Notes:** `[auto] SSH 就绪与交接 — 选择推荐默认：加入 SSH ready gate，随后自动 exec ssh。`

---

## 失败映射与重试体验

| Option | Description | Selected |
|--------|-------------|----------|
| 稳定错误分类 + 中文终端提示 + 无自动重试（推荐） | 延续 Phase 1/2 的失败策略，避免隐式恢复，便于运维定位 | ✓ |
| 失败后自动重试若干次 | 可能掩盖根因，与既有“显式重试”策略冲突 | |
| 统一输出“启动失败”不区分原因 | 用户与运维都缺乏可执行下一步信息 | |

**User's choice:** auto-selected 推荐方案（稳定错误分类 + 显式重试）
**Notes:** `[auto] 失败映射与重试体验 — 选择推荐默认：细分错误码、不给自动重试、返回明确下一步。`

---

## Claude's Discretion

- 轮询频率、超时阈值与退避参数。
- 终端进度展示的具体视觉形式（spinner / 行内更新 / 分段日志）。
- SSH 连接参数展开方式（命令行参数或临时配置文件）。

## Deferred Ideas

None.
