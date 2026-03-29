# Phase 7: 数据层与类型化 - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-03-28
**Phase:** 07-数据层与类型化
**Areas discussed:** proxy_config 校验粒度, EgressConfig 结构变更, 敏感字段清理策略, tunnel_type 校验实现层
**Mode:** --auto (all decisions auto-selected)

---

## proxy_config 校验粒度

| Option | Description | Selected |
|--------|-------------|----------|
| JSON 合法性校验 | 仅验证 proxy_config 是合法 JSON 且 type 在白名单内 | |
| 白名单 + 必需字段校验 | 白名单 type + 按协议验证通用和专有必需字段 | ✓ |
| 完整 sing-box 配置校验 | 完全模拟 sing-box 的配置解析，包括 TLS/transport 等嵌套校验 | |

**User's choice:** [auto] 白名单 + 必需字段校验 (recommended default)
**Notes:** 需求 DATA-04 明确要求 "白名单限制 outbound type 并验证协议必需字段"。完整 sing-box 校验过于复杂且耦合版本，不适合 v1.1。

---

## EgressConfig 结构变更

| Option | Description | Selected |
|--------|-------------|----------|
| 双指针分支 | TunnelType + Tunnel *TunnelSpec + Proxy *ProxySpec | ✓ |
| 联合体风格 | 单一 Config 字段 + interface{} 类型断言 | |
| 嵌入现有 TunnelSpec | 在 TunnelSpec 内混入 proxy 字段 | |

**User's choice:** [auto] 双指针分支 (recommended default)
**Notes:** plan 文档已明确此设计。类型安全，下游 Provider 可按指针是否为 nil 判断类型。混入字段会污染 TunnelSpec 的 WireGuard 语义。

---

## 敏感字段清理策略

| Option | Description | Selected |
|--------|-------------|----------|
| 清理 password 字段 | API 响应时清除 proxy_config 中的 password 类字段 | ✓ |
| 不清理 | proxy_config 原样返回（前端需要密码回显编辑） | |
| 整体脱敏 | proxy_config 在 API 响应中完全不返回 | |

**User's choice:** [auto] 清理 password 字段 (recommended default)
**Notes:** 与现有 wg_preshared_key 清理逻辑对齐，安全一致性优先。前端编辑时需单独处理（更新时才发送密码字段）。

---

## tunnel_type 校验实现层

| Option | Description | Selected |
|--------|-------------|----------|
| 仅 DB CHECK 约束 | 数据库层面约束，应用层不校验 | |
| 仅 API 层校验 | 应用层返回 400，DB 无约束 | |
| 三层校验 | DB CHECK + Go 常量 + API handler 校验 | ✓ |

**User's choice:** [auto] 三层校验 (recommended default)
**Notes:** 关键枚举值应多层防护。DB CHECK 防直连修改，Go 常量防拼写错误，API 层提供友好错误消息。

---

## Agent's Discretion

- proxy_config 校验函数内部组织方式
- Go 常量包归属
- sanitizeProxyConfig 实现策略
- 校验错误消息措辞

## Deferred Ideas

None — discussion stayed within phase scope
