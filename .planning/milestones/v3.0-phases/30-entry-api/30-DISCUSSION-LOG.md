# Phase 30: 控制面数据模型 + Entry API 扩展 - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-04-18
**Phase:** 30-控制面数据模型 + Entry API 扩展
**Mode:** `--auto`（全部灰区自动选「ROADMAP 推荐项 + 与 Phase 29 对齐」）
**Areas discussed:** Q4 volume 命名、Q5 API 面、Q6 host-agent、Entry 字段推导、HostActionRequest 账号字段、迁移语义

---

## Q4 · persistent volume 命名与 migration 字段语义

| Option | Description | Selected |
|--------|-------------|----------|
| 单 volume `claude-state-{id}` | REQ-F7-A / ROADMAP 倾向 | ✓ |
| 双 volume（cache 分离） | 更强隔离，运维复杂 | |
| `ccp_` 前缀 | Q4 备选 | |

**User's choice:** `[auto]` 单 volume + `persistent_volume_name` 列 `NULL` = 未分配（见 CONTEXT D-01 / D-02）。

---

## Q5 · 能力探测接口形态

| Option | Description | Selected |
|--------|-------------|----------|
| 扩展现有 `/v1/entry/.../auth` | 无额外 RTT，向后兼容 JSON | ✓ |
| 新增 `/capabilities` | 清晰分离，但超出本 phase ROADMAP | |

**User's choice:** `[auto]` 扩展现有 endpoint（CONTEXT D-03）。

---

## Q6 · host-agent 是否回传 image labels

| Option | Description | Selected |
|--------|-------------|----------|
| 不扩展 host-agent | 与 Phase 29 D 一致 | ✓ |
| 扩展返回 labels | ROADMAP 标记为不倾向 | |

**User's choice:** `[auto]` 控制面仅从 `template_image_ref` 推导（CONTEXT D-04 / D-06 / D-07）。

---

## `claude_account_id` 解析策略

| Option | Description | Selected |
|--------|-------------|----------|
| host_id 优先 → user 级回退 | 确定性、兼容尚无 host 绑定行 | ✓ |
| 仅 user 级随机取 | 多账号歧义 | |

**User's choice:** `[auto]` CONTEXT D-05。

---

## `supports_*` 真值条件

| Option | Description | Selected |
|--------|-------------|----------|
| tag == `v3.0.0` | 与 ROADMAP 验收字面一致 | ✓ |
| 语义化版本范围 `>= 3.0` | 更宽松，留待后续 | |

**User's choice:** `[auto]` 精确匹配 `v3.0.0`（CONTEXT D-07）。

---

## Claude's Discretion

- `EntryStore` 方法拆分与查询次数：交由 planner/实现（CONTEXT D-11）。

## Deferred Ideas

- 双 volume、`ccp_` 前缀、独立 capabilities、registry metadata — 见 CONTEXT `<deferred>`。
