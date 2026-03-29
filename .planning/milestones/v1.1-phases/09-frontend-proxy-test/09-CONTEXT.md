# Phase 9: 前端适配与代理测试 - Context

**Gathered:** 2026-03-28
**Status:** Ready for planning

<domain>
## Phase Boundary

管理后台完整支持代理类型出口 IP 的配置、验证和测试结果展示。包含：出口 IP 创建/编辑表单按隧道类型动态切换字段、代理协议模式下按协议类型动态渲染对应字段、高级 JSON 编辑模式、一键代理测试 API（连通性 + 出口 IP 匹配 + DNS 泄漏检测）、测试结果前端展示（列表页状态指示器 + 详情弹窗）。

**不在本阶段范围内：** 代理健康定时检测/告警（Future PROX-01）、代理配置模板（Future PROX-02）、批量代理测试（Future PROX-03）、多节点负载均衡（Future PROX-04）。

</domain>

<decisions>
## Implementation Decisions

### 隧道类型切换方式

- **D-01:** 在 Drawer 表单中使用 Select 下拉选择隧道类型（WireGuard / 代理协议），放在基础信息（标签、IP 地址、提供商）和协议配置字段之间
- **D-02:** 选择 WireGuard 时显示现有 wg_* 字段，选择代理协议时显示协议类型选择器 + 对应字段
- **D-03:** 创建模式下默认选中 WireGuard（向后兼容，与 Phase 7 D-19 一致）
- **D-04:** 编辑模式下根据已保存的 tunnel_type 回填，并相应显示对应字段组

### 代理协议表单设计

- **D-05:** 选择"代理协议"后，先通过 Select 下拉选择协议类型（socks / vmess / shadowsocks / trojan / http），SOCKS5 默认选中
- **D-06:** 各协议按 proxy-tunnel-plan.md 字段映射表动态渲染：
  - socks: server, port, username(选填), password(选填)
  - vmess: server, port, uuid, security, alter_id, tls 开关 + server_name
  - shadowsocks: server, port, method, password
  - trojan: server, port, password, tls 开关 + server_name
  - http: server, port, username(选填), password(选填), tls 开关
- **D-07:** server 和 port 作为通用字段始终显示在协议专有字段之前
- **D-08:** 密码类字段使用 type="password" 且带可见性切换按钮

### JSON 编辑模式

- **D-09:** 在代理协议配置区域提供"表单模式" / "JSON 模式"切换按钮
- **D-10:** JSON 模式使用 monospace textarea（不引入 Monaco/CodeMirror 依赖），提供语法错误提示
- **D-11:** 表单模式 → JSON 模式时，将当前表单值序列化为 sing-box outbound JSON 填入编辑器
- **D-12:** JSON 模式 → 表单模式时，尝试解析 JSON 回填表单字段；解析失败则提示用户
- **D-13:** 提交时统一使用 JSON 格式的 proxy_config 发送到后端

### 测试 API 与交互

- **D-14:** 列表页每行操作菜单新增"测试"按钮
- **D-15:** 点击测试后显示 loading 状态（按钮变为 spinner），API 调用 `POST /v1/admin/egress-ips/{id}/test`
- **D-16:** 测试结果在 Dialog 中展示详情：连通性（pass/fail + 延迟）、出口 IP 匹配（预期 vs 实际）、DNS 泄漏（检测到的 DNS 服务器列表）
- **D-17:** 测试结果总状态使用颜色编码：全部通过=绿色、部分失败=黄色、全部失败=红色
- **D-18:** 后端测试 API 设置 30 秒总超时（TEST-04）

### 列表页展示增强

- **D-19:** 出口 IP 列表表格新增"隧道类型"列，使用 Badge 区分（WireGuard 蓝色 / 代理 紫色）
- **D-20:** 新增"测试状态"列，显示最近一次测试结果的状态圆点（绿色=通过 / 红色=失败 / 灰色=未测试）
- **D-21:** 状态圆点带 Tooltip 显示测试时间

### TypeScript 类型更新

- **D-22:** `EgressIP` 接口新增 `tunnel_type: string`、`proxy_config: Record<string, unknown> | null` 字段
- **D-23:** 新增 `useTestEgressIP` mutation hook，调用测试 API
- **D-24:** 新增 `TestResult` 接口，包含 status、tested_at、results（connectivity/egress_ip/dns_leak 子项）

### Agent's Discretion

- Zod schema 的具体组织方式（一个大 schema 带 discriminatedUnion vs 多个小 schema）
- 表单组件的内部拆分策略（单文件 vs 拆分为 WgFields / ProxyFields 子组件）
- 测试结果 Dialog 的具体布局细节
- DNS 泄漏检测的具体实现方式（后端如何判定 DNS 泄漏）

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### 代理隧道整体设计

- `.planning/proxy-tunnel-plan.md` — 代理协议出网完整设计方案，§7 前端适配的字段映射表、§9 后台一键代理测试 API 设计和返回格式
- `.planning/REQUIREMENTS.md` — TEST-01 到 TEST-04（代理测试）+ UI-01 到 UI-05（前端适配）需求定义

### Phase 7/8 上下文

- `.planning/phases/07-data-layer-typing/07-CONTEXT.md` — 数据层决策：tunnel_type/proxy_config 模型、proxy_config 白名单、API 脱敏
- `.planning/phases/08-singboxprovider/08-CONTEXT.md` — SingBoxProvider 决策：Provider 工厂架构、sing-box 配置模型

### 现有前端代码

- `web/admin/src/hooks/use-egress-ips.ts` — EgressIP TypeScript 接口和 CRUD hooks（需扩展 tunnel_type/proxy_config 字段 + 新增测试 hook）
- `web/admin/src/components/egress-ips/egress-ip-drawer.tsx` — 现有出口 IP Drawer 表单（react-hook-form + zod + shadcn/ui Sheet），需改造为动态表单
- `web/admin/src/routes/_dashboard/egress-ips/index.tsx` — 出口 IP 列表页（shadcn/ui Table + DropdownMenu），需新增列和测试功能

### 现有 UI 组件

- `web/admin/src/components/ui/` — 可用 shadcn/ui 组件：badge, button, card, dialog, dropdown-menu, input, label, select, separator, sheet, table, tooltip

### 后端 API

- `internal/controlplane/http/admin_egress_ips.go` — Admin 出口 IP CRUD handler（已支持 tunnel_type + proxy_config），需新增测试 endpoint

### API 客户端

- `web/admin/src/lib/api.ts` — apiFetch 工具函数，Bearer token 认证

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets

- `EgressIPDrawer` (`egress-ip-drawer.tsx`): react-hook-form + zod 表单框架，可扩展为动态字段切换
- `Select` / `Input` / `Badge` / `Dialog` / `Tooltip` (shadcn/ui): 现有 UI 组件，直接复用
- `useEgressIPs` / `useCreateEgressIP` / `useUpdateEgressIP` hooks: TanStack Query 模式，新增 `useTestEgressIP` mutation 遵循同一模式
- `apiFetch`: API 客户端，已处理 401 和 JSON 解析

### Established Patterns

- 表单使用 react-hook-form + zod resolver，`useForm<FormValues>` + `form.register()` + `form.handleSubmit()`
- CRUD 使用 TanStack Query 的 `useQuery` + `useMutation`，mutation 成功后 `invalidateQueries`
- 列表页使用 shadcn/ui Table，操作列用 DropdownMenu
- 抽屉表单使用 shadcn/ui Sheet，480px 宽度
- 中文 UI 文案，日期使用 `zh-CN` locale 格式化
- 状态用 Badge 展示，`variant="default"` 为正常状态，`variant="secondary"` 为次要状态

### Integration Points

- `egress-ip-drawer.tsx` — 需改造表单 schema 和字段渲染逻辑
- `egress-ips/index.tsx` — 需新增列、测试按钮、测试结果展示
- `use-egress-ips.ts` — 需扩展 EgressIP 接口、新增测试相关 hook
- `admin_egress_ips.go` — 后端需新增 `POST /egress-ips/{id}/test` handler

</code_context>

<specifics>
## Specific Ideas

- SOCKS5 场景下用户只需填 4 个字段（服务器、端口、用户名、密码），平台自动处理 tun 全流量接管和 DNS 防泄漏
- 测试结果结构遵循 proxy-tunnel-plan.md §9 设计：`{ status, tested_at, results: { connectivity, egress_ip, dns_leak } }`
- 代理测试 API 实现方式：SOCKS5 直接用 Go 的 `golang.org/x/net/proxy` 发起请求；非 SOCKS5 协议临时启动 sing-box 进程作为本地 SOCKS5 转发再测试
- 出口 IP 创建完成后可考虑自动弹出"是否立即测试"（作为 Agent's Discretion 处理）

</specifics>

<deferred>
## Deferred Ideas

None — discussion stayed within phase scope

</deferred>

---

*Phase: 09-frontend-proxy-test*
*Context gathered: 2026-03-28*
