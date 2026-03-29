# Phase 7: 数据层与类型化 - Context

**Gathered:** 2026-03-28
**Status:** Ready for planning

<domain>
## Phase Boundary

出口 IP 数据模型完整支持 wireguard 和 proxy 两种隧道类型。包含：DB migration 新增 tunnel_type + proxy_config 字段、Go 数据模型扩展（repository + network 层）、绑定校验按 tunnel_type 分支验证、Admin API CRUD 适配新字段、proxy_config 白名单和协议必需字段校验。

**不在本阶段范围内：** SingBoxProvider 实现（Phase 8）、受管镜像预装 sing-box（Phase 8）、前端表单动态切换（Phase 9）、代理测试 API（Phase 9）。

</domain>

<decisions>
## Implementation Decisions

### DB Migration

- **D-01:** 新增 migration 文件 `0004_proxy_tunnel.sql`
- **D-02:** `egress_ips` 表新增 `tunnel_type TEXT NOT NULL DEFAULT 'wireguard'`，带 CHECK 约束 `tunnel_type IN ('wireguard', 'proxy')`
- **D-03:** `egress_ips` 表新增 `proxy_config JSONB`，默认 NULL
- **D-04:** 现有数据自动获得 `tunnel_type='wireguard'` 默认值，无需手动回填，不影响已有行

### Go 数据模型

- **D-05:** `repository.EgressIP` 新增 `TunnelType string` 和 `ProxyConfig json.RawMessage`（可为 nil）
- **D-06:** `repository.CreateEgressIPParams` / `UpdateEgressIPParams` 新增对应字段
- **D-07:** `network.EgressConfig` 新增 `TunnelType string`，将 `Tunnel TunnelSpec` 改为 `Tunnel *TunnelSpec`（指针），新增 `Proxy *ProxySpec`
- **D-08:** 新增 `network.ProxySpec` 结构体：`OutboundConfig json.RawMessage`（sing-box outbound JSON）+ `DNSServer string`
- **D-09:** `network.EgressIPRecord` 新增 `TunnelType string` 和 `ProxyConfig json.RawMessage`
- **D-10:** `TunnelSpec` 结构体本身不变，保持 WireGuard 专有语义

### 绑定校验

- **D-11:** `ValidateEgressBinding` 根据 `record.TunnelType` 分支：
  - `wireguard`：现有逻辑不变（要求 wg_endpoint + wg_public_key 非空）
  - `proxy`：要求 `proxy_config` 非空且 JSON 可解析
- **D-12:** proxy 分支校验不需要 `GetHostWgKeys`，直接从 proxy_config 构建 `ProxySpec`
- **D-13:** proxy 分支返回的 `EgressConfig` 中 `Tunnel` 为 nil，`Proxy` 填充

### proxy_config 校验

- **D-14:** 白名单 outbound type：`socks` / `vmess` / `shadowsocks` / `trojan` / `http`
- **D-15:** 通用必需字段：`server`（非空字符串）、`server_port`（正整数）
- **D-16:** 协议专有必需字段：
  - `socks`：无额外必需字段（username/password 可选）
  - `vmess`：`uuid`（非空）
  - `shadowsocks`：`method`（非空）、`password`（非空）
  - `trojan`：`password`（非空）
  - `http`：无额外必需字段（username/password 可选）
- **D-17:** 校验在 API handler 层执行（Create/Update 时），作为独立函数 `validateProxyConfig`

### Admin API

- **D-18:** `createEgressIPRequest` / `updateEgressIPRequest` 新增 `tunnel_type` 和 `proxy_config` JSON 字段
- **D-19:** `tunnel_type` 未提供时默认为 `wireguard`（向后兼容现有前端）
- **D-20:** `tunnel_type=proxy` 时 `proxy_config` 必填，`wg_*` 字段忽略
- **D-21:** `tunnel_type=wireguard` 时 `proxy_config` 字段忽略，保持现有校验行为
- **D-22:** API 响应中 `proxy_config` 需清理密码类字段（password），与 `wg_preshared_key` 清理逻辑对齐

### tunnel_type 校验层

- **D-23:** 三层校验——DB CHECK 约束防脏数据、Go 常量（`TunnelTypeWireGuard`/`TunnelTypeProxy`）防拼写错误、API handler 层返回 400 错误提示

### Agent's Discretion

- proxy_config 校验函数的内部组织方式（switch vs map vs 接口）
- Go 常量的包归属（`repository` 包 vs `network` 包 vs 新建 `types` 包）
- sanitizeProxyConfig 的具体实现策略（深拷贝 vs 正则替换 vs JSON 遍历）
- 校验错误消息的具体措辞

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### 代理隧道设计

- `.planning/proxy-tunnel-plan.md` — 代理协议出网的完整设计方案，包含 DB schema 变更、数据模型设计、各协议 proxy_config 示例、前端字段映射表
- `.planning/REQUIREMENTS.md` — DATA-01 到 DATA-05 需求定义

### 现有数据层

- `internal/store/repository/models.go` — 当前 EgressIP 结构体和 CRUD 参数定义
- `internal/store/migrations/0002_egress_tunnel.sql` — 现有 WireGuard 字段 migration，新 migration 应延续此模式
- `internal/store/repository/queries.go` — 现有 SQL 查询实现

### 网络层

- `internal/network/types.go` — TunnelSpec / EgressConfig / HostNetworkSpec 定义
- `internal/network/validate.go` — ValidateEgressBinding 校验逻辑和 EgressIPRecord / EgressValidator 接口

### API 层

- `internal/controlplane/http/admin_egress_ips.go` — Admin 出口 IP CRUD handler，包含请求结构体和 sanitize 逻辑

### 前端类型（参考）

- `web/admin/src/hooks/use-egress-ips.ts` — EgressIP TypeScript 接口定义，Phase 9 需同步更新

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets

- `sanitizeEgressIP` (`admin_egress_ips.go`): 现有脱敏逻辑，可扩展为同时处理 proxy_config 密码字段
- `ValidateEgressBinding` (`validate.go`): 校验入口，需按 tunnel_type 分支但整体框架可复用
- `EgressValidator` 接口 (`validate.go`): 可能需要扩展以支持 proxy 类型的查询（`GetEgressIPByHost` 返回的 record 需包含新字段）
- `truncateID` (`validate.go`): 通用辅助函数，无需修改

### Established Patterns

- DB migration 使用顺序编号文件（`0001_initial.sql`, `0002_egress_tunnel.sql`, `0003_expiry_audit.sql`），由 `internal/store/migrator/migrator.go` 执行
- Repository 模型使用 JSON tag 与 API 字段名对齐（`json:"snake_case"`）
- API handler 使用 `writeJSON` 返回响应，错误用 `map[string]string{"error": "..."}`
- 事件记录使用 `h.events.RecordEvent` with metadata map
- `*string` 指针表示可选字段（nullable），`string` 表示非空字段

### Integration Points

- `internal/store/repository/queries.go` — 需新增/修改 CreateEgressIP / UpdateEgressIP / GetEgressIP 等查询以包含新列
- `internal/runtime/tasks/worker.go` — `buildEgressConfig` 调用 `ValidateEgressBinding`，返回的 `EgressConfig` 结构变更会自然传导
- `internal/agent/server.go` — 使用 `Worker` + `Provider`，`EgressConfig` 变更自动传导

</code_context>

<specifics>
## Specific Ideas

- proxy_config 使用 sing-box outbound 原生 JSON 格式存储，不为每种协议单独建列——这样 sing-box 支持的所有协议都可自动适配
- SOCKS5 是一等公民，前端默认协议（Phase 9 关注）
- `tunnel_type` 默认值为 `wireguard`，确保 v1.0 → v1.1 升级零破坏
- 各协议 proxy_config 示例参见 `.planning/proxy-tunnel-plan.md` §1

</specifics>

<deferred>
## Deferred Ideas

None — discussion stayed within phase scope

</deferred>

---

*Phase: 07-data-layer-typing*
*Context gathered: 2026-03-28*
