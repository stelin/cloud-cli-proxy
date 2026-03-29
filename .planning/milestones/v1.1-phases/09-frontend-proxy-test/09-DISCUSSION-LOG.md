# Phase 9: 前端适配与代理测试 - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-03-28
**Phase:** 09-frontend-proxy-test
**Areas discussed:** 隧道类型切换方式, 代理协议表单设计, JSON 编辑模式, 测试结果交互, 列表页类型展示
**Mode:** --auto (all decisions auto-selected)

---

## 隧道类型切换方式

| Option | Description | Selected |
|--------|-------------|----------|
| Select 下拉 | 与现有 status Select 组件模式一致，放在基础信息与协议配置之间 | ✓ |
| Tab 切换 | 使用 Tabs 组件在 WireGuard/代理协议间切换 | |
| Radio group | 使用 RadioGroup 水平排列两个选项 | |

**User's choice:** [auto] Select 下拉（recommended default — 与现有表单中 status 的 Select 模式一致）
**Notes:** 现有 Drawer 已使用 shadcn/ui Select 组件展示 status，隧道类型选择器遵循同一模式最为自然。

---

## 代理协议表单设计

| Option | Description | Selected |
|--------|-------------|----------|
| 协议 Select → 动态字段 | 先选协议类型（Select 下拉），再动态渲染对应字段，SOCKS5 默认 | ✓ |
| Accordion 折叠 | 所有协议字段组用 Accordion 展示，展开当前选中的 | |
| 统一字段 + 条件隐藏 | 所有字段始终存在，根据协议类型条件隐藏不需要的 | |

**User's choice:** [auto] 协议 Select → 动态字段（recommended default — 符合 Phase 7 "SOCKS5 一等公民"决策 + proxy-tunnel-plan.md 字段映射表）
**Notes:** 每种协议的必需字段差异较大（SOCKS5 最简 4 字段，VMess 需要 uuid/security/alter_id/tls），动态渲染最清晰。

---

## JSON 编辑模式

| Option | Description | Selected |
|--------|-------------|----------|
| monospace textarea | 纯 textarea + monospace 字体，不引入额外依赖 | ✓ |
| Monaco Editor | 功能丰富的代码编辑器，支持 JSON 语法高亮和验证 | |
| CodeMirror | 轻量代码编辑器，支持 JSON 模式 | |

**User's choice:** [auto] monospace textarea（recommended default — 管理后台场景不需要完整 IDE 体验，避免引入重依赖）
**Notes:** 高级 JSON 模式主要面向有经验的管理员直接编辑 sing-box outbound 配置，textarea 足够。后续如需增强可升级到 CodeMirror。

---

## 测试结果交互

| Option | Description | Selected |
|--------|-------------|----------|
| 列表页行内按钮 + Dialog 详情 | 列表页每行操作菜单添加测试按钮，结果在 Dialog 中展示 | ✓ |
| Drawer 内测试 | 在编辑 Drawer 底部添加测试按钮和结果展示 | |
| Toast 通知 | 测试结果以 Toast 形式简短通知 | |

**User's choice:** [auto] 列表页行内按钮 + Dialog 详情（recommended default — 与 proxy-tunnel-plan.md 设计一致，Dialog 能展示完整的三项检测结果）
**Notes:** 测试结果包含连通性、出口 IP 匹配和 DNS 泄漏三个子项，需要足够的展示空间。Dialog 比 Toast 更合适。

---

## 列表页类型展示

| Option | Description | Selected |
|--------|-------------|----------|
| Badge 列 | 新增隧道类型列，用 Badge 组件区分 WireGuard/代理 | ✓ |
| 图标列 | 用小图标区分（🔒 WireGuard / 🌐 代理） | |
| 标签/IP 列合并 | 在标签或 IP 地址旁用小标记区分 | |

**User's choice:** [auto] Badge 列（recommended default — 与现有 status Badge 展示模式一致）
**Notes:** 现有列表已用 Badge 展示状态（可用/已禁用），隧道类型用相同组件保持视觉一致性。

---

## Agent's Discretion

- Zod schema 的具体组织方式
- 表单子组件拆分策略
- 测试结果 Dialog 布局细节
- DNS 泄漏检测的后端实现方式

## Deferred Ideas

None — discussion stayed within phase scope
