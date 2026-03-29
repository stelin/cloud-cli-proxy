# Phase 7: 数据层与类型化 - Research

**Researched:** 2026-03-28
**Domain:** Go 数据模型扩展、PostgreSQL schema migration、JSONB 校验
**Confidence:** HIGH

## Summary

本阶段的核心工作是让 `egress_ips` 表和 Go 数据模型支持 `wireguard` 和 `proxy` 两种隧道类型。技术上属于典型的"已有表加列 + 已有结构体加字段 + 校验逻辑分支"模式，不涉及新外部依赖，所有变更都在项目已建立的技术栈范围内完成。

主要工作分四个维度：(1) DB migration 新增 `tunnel_type` 和 `proxy_config` 两列；(2) Go 侧 `repository.EgressIP` / `network.EgressConfig` / `network.EgressIPRecord` 等结构体扩展；(3) `ValidateEgressBinding` 按 `tunnel_type` 分支校验；(4) Admin API handler 适配新字段、`proxy_config` 白名单校验、响应脱敏。

**Primary recommendation:** 按"migration → models → repository queries → network types → validation → API handler"的依赖链顺序实施，每一步都可独立编译验证。

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions
- **D-01:** 新增 migration 文件 `0004_proxy_tunnel.sql`
- **D-02:** `egress_ips` 表新增 `tunnel_type TEXT NOT NULL DEFAULT 'wireguard'`，带 CHECK 约束 `tunnel_type IN ('wireguard', 'proxy')`
- **D-03:** `egress_ips` 表新增 `proxy_config JSONB`，默认 NULL
- **D-04:** 现有数据自动获得 `tunnel_type='wireguard'` 默认值，无需手动回填
- **D-05:** `repository.EgressIP` 新增 `TunnelType string` 和 `ProxyConfig json.RawMessage`
- **D-06:** `repository.CreateEgressIPParams` / `UpdateEgressIPParams` 新增对应字段
- **D-07:** `network.EgressConfig` 新增 `TunnelType string`，将 `Tunnel TunnelSpec` 改为 `Tunnel *TunnelSpec`（指针），新增 `Proxy *ProxySpec`
- **D-08:** 新增 `network.ProxySpec` 结构体：`OutboundConfig json.RawMessage` + `DNSServer string`
- **D-09:** `network.EgressIPRecord` 新增 `TunnelType string` 和 `ProxyConfig json.RawMessage`
- **D-10:** `TunnelSpec` 结构体本身不变
- **D-11:** `ValidateEgressBinding` 根据 `record.TunnelType` 分支
- **D-12:** proxy 分支校验不需要 `GetHostWgKeys`，直接从 `proxy_config` 构建 `ProxySpec`
- **D-13:** proxy 分支返回的 `EgressConfig` 中 `Tunnel` 为 nil，`Proxy` 填充
- **D-14:** 白名单 outbound type：`socks` / `vmess` / `shadowsocks` / `trojan` / `http`
- **D-15:** 通用必需字段：`server`（非空字符串）、`server_port`（正整数）
- **D-16:** 协议专有必需字段已确定（socks 无额外、vmess 要 uuid、ss 要 method+password、trojan 要 password、http 无额外）
- **D-17:** 校验在 API handler 层执行，作为独立函数 `validateProxyConfig`
- **D-18 ~ D-22:** Admin API 请求/响应结构适配，含向后兼容和脱敏
- **D-23:** 三层校验——DB CHECK + Go 常量 + API handler 400 错误

### Claude's Discretion
- proxy_config 校验函数的内部组织方式（switch vs map vs 接口）
- Go 常量的包归属（`repository` 包 vs `network` 包 vs 新建 `types` 包）
- sanitizeProxyConfig 的具体实现策略（深拷贝 vs 正则替换 vs JSON 遍历）
- 校验错误消息的具体措辞

### Deferred Ideas (OUT OF SCOPE)
None — discussion stayed within phase scope
</user_constraints>

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| DATA-01 | 管理员创建出口 IP 时选择隧道类型（wireguard 或 proxy） | DB migration D-02 + API handler D-18/D-19 + Go 常量 D-23 |
| DATA-02 | 代理类型的出口 IP 存储 sing-box outbound JSON 配置 | DB migration D-03 + repository model D-05/D-06 + `json.RawMessage` 存储模式 |
| DATA-03 | 绑定校验根据 tunnel_type 分支验证 | ValidateEgressBinding 重构 D-11/D-12/D-13 + EgressIPRecord 扩展 D-09 |
| DATA-04 | proxy_config 白名单和协议必需字段校验 | validateProxyConfig 函数 D-14/D-15/D-16/D-17 |
| DATA-05 | DB migration 确保现有 WG 出口 IP 自动获得默认值 | `DEFAULT 'wireguard'` D-02/D-04 保证零回填 |
</phase_requirements>

## Standard Stack

### Core

| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| `encoding/json` | Go 标准库 | `json.RawMessage` 存取 JSONB、`json.Unmarshal` 做 proxy_config 校验 | 项目已在用，`json.RawMessage` 是存储非结构化 JSON 的标准方式 |
| `github.com/jackc/pgx/v5` | v5.7.6 | PostgreSQL 驱动，JSONB 列的读写 | 项目已在用，pgx 原生支持 `[]byte` ↔ JSONB |
| Go 标准库 `net` | Go 标准库 | IP 地址解析（现有用法） | 项目已在用 |

### Supporting

无需新增外部依赖。本阶段所有功能均可用 Go 标准库 + 项目已有的 pgx 完成。

### Alternatives Considered

| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| `json.RawMessage` 手动校验 | `github.com/xeipuuv/gojsonschema` | JSON Schema 校验更声明式，但引入新依赖且白名单校验逻辑简单到不需要 |
| 手写 switch 做协议分支 | 接口 + 注册表模式 | 当前只有 5 种协议，switch 最直观；协议数量超过 10 时再考虑注册表 |

## Architecture Patterns

### 变更影响面总览

```
internal/store/migrations/
└── 0004_proxy_tunnel.sql              # 新增

internal/store/repository/
├── models.go                          # EgressIP / CreateParams / UpdateParams 加字段
└── queries.go                         # 所有 EgressIP 相关查询加列

internal/network/
├── types.go                           # EgressConfig 重构 + 新增 ProxySpec
└── validate.go                        # EgressIPRecord 加字段 + ValidateEgressBinding 分支

internal/controlplane/http/
└── admin_egress_ips.go                # request/response 结构 + validateProxyConfig + sanitize

internal/runtime/tasks/
└── worker.go                          # repoValidator 适配新字段（自动传导）
```

### Pattern 1: tunnel_type 分支校验

**What:** `ValidateEgressBinding` 先读取 `record.TunnelType`，按 `wireguard` / `proxy` 分支构建不同的 `EgressConfig`。

**When to use:** 每次 host 启动或重建调用 `buildEgressConfig` 时。

**Example:**

```go
func ValidateEgressBinding(ctx context.Context, v EgressValidator, hostID string) (EgressConfig, error) {
    record, err := v.GetEgressIPByHost(ctx, hostID)
    if err != nil {
        return EgressConfig{}, &NetworkError{
            Type: ErrBindingMissing, Message: "no egress IP bound to host", HostID: hostID,
        }
    }

    switch record.TunnelType {
    case TunnelTypeProxy:
        return validateProxyBinding(record, hostID)
    default:
        return validateWireGuardBinding(ctx, v, record, hostID)
    }
}
```

### Pattern 2: json.RawMessage 存储 JSONB

**What:** `proxy_config` 在 Go 侧用 `json.RawMessage`（即 `[]byte`），pgx 天然支持 `[]byte` ↔ PostgreSQL `JSONB` 的双向转换。

**When to use:** 需要存储非结构化 JSON、避免为每种协议建模具体 struct 时。

**Example:**

```go
type EgressIP struct {
    // ... existing fields ...
    TunnelType  string          `json:"tunnel_type"`
    ProxyConfig json.RawMessage `json:"proxy_config,omitempty"`
}
```

pgx `Scan` 时直接用 `&item.ProxyConfig`（`*[]byte`），pgx 会把 JSONB 数据赋值进去。对于 NULL 值，`json.RawMessage` 会保持 `nil`。

### Pattern 3: proxy_config 白名单校验

**What:** 先解析为 `map[string]any` 提取 `type` 字段做白名单检查，再按协议验证必需字段。

**When to use:** Admin API Create/Update handler 中，`tunnel_type=proxy` 时对 `proxy_config` 做结构校验。

**Example:**

```go
var allowedOutboundTypes = map[string]bool{
    "socks": true, "vmess": true, "shadowsocks": true, "trojan": true, "http": true,
}

func validateProxyConfig(raw json.RawMessage) error {
    var parsed map[string]any
    if err := json.Unmarshal(raw, &parsed); err != nil {
        return fmt.Errorf("proxy_config is not valid JSON: %w", err)
    }

    outboundType, _ := parsed["type"].(string)
    if !allowedOutboundTypes[outboundType] {
        return fmt.Errorf("unsupported outbound type %q, allowed: socks, vmess, shadowsocks, trojan, http", outboundType)
    }

    server, _ := parsed["server"].(string)
    if server == "" {
        return fmt.Errorf("proxy_config.server is required")
    }

    port, _ := parsed["server_port"].(float64)
    if port <= 0 || port > 65535 {
        return fmt.Errorf("proxy_config.server_port must be a positive integer (1-65535)")
    }

    switch outboundType {
    case "vmess":
        if uuid, _ := parsed["uuid"].(string); uuid == "" {
            return fmt.Errorf("proxy_config.uuid is required for vmess")
        }
    case "shadowsocks":
        if method, _ := parsed["method"].(string); method == "" {
            return fmt.Errorf("proxy_config.method is required for shadowsocks")
        }
        if password, _ := parsed["password"].(string); password == "" {
            return fmt.Errorf("proxy_config.password is required for shadowsocks")
        }
    case "trojan":
        if password, _ := parsed["password"].(string); password == "" {
            return fmt.Errorf("proxy_config.password is required for trojan")
        }
    }

    return nil
}
```

### Pattern 4: 响应脱敏（sanitizeProxyConfig）

**What:** API 响应中需要清理 `proxy_config` 里的 `password` 字段，与现有 `sanitizeEgressIP` 清理 `wg_preshared_key` 的模式对齐。

**Recommended approach:** JSON 反序列化为 `map[string]any` → 删除/替换敏感 key → 序列化回 `json.RawMessage`。

**Example:**

```go
var sensitiveProxyFields = []string{"password"}

func sanitizeProxyConfig(raw json.RawMessage) json.RawMessage {
    if raw == nil {
        return nil
    }
    var parsed map[string]any
    if err := json.Unmarshal(raw, &parsed); err != nil {
        return nil
    }
    for _, key := range sensitiveProxyFields {
        if _, ok := parsed[key]; ok {
            parsed[key] = "***"
        }
    }
    sanitized, _ := json.Marshal(parsed)
    return sanitized
}
```

### Anti-Patterns to Avoid

- **为每种协议建独立 DB 列：** proxy_config 存 JSONB 是锁定决策，不要为 `vmess_uuid` / `ss_method` 等建列。
- **在 network 包直接 import repository 包：** 当前架构用 `EgressIPRecord` 做中间层，不要打破这个依赖边界。
- **校验逻辑放在 repository 层：** D-17 锁定校验在 API handler 层执行，repository 只做存取。

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| JSONB 列读写 | 自定义 SQL 编解码 | pgx 原生 `[]byte` ↔ JSONB | pgx v5 天然支持，scan 到 `[]byte` / `json.RawMessage` 无需任何适配器 |
| JSON 校验 | 完整 JSON Schema validator | `json.Unmarshal` + 手写 switch | 白名单仅 5 种协议，每种最多 3 个必需字段，不值得引入 JSON Schema |
| NULL 处理 | 自定义 NullJSON 类型 | `json.RawMessage`（nil 即 NULL） | pgx 对 `*[]byte` 的 NULL 处理是开箱即用的 |

## Common Pitfalls

### Pitfall 1: pgx JSONB NULL scan

**What goes wrong:** `json.RawMessage` 是 `[]byte` 的别名，pgx scan NULL JSONB 时如果目标是非指针 `[]byte`，不会出错但会得到 `nil` 值。需要确认 query 中对 NULL 的处理。

**Why it happens:** PostgreSQL JSONB 列 `NULL` 和 JSON `null` 是两种不同的值。`proxy_config` 默认为 `NULL`（D-03），对 WireGuard 类型的行一直是 NULL。

**How to avoid:** 直接 scan 到 `&item.ProxyConfig`（`json.RawMessage` 即 `[]byte`），pgx 会正确处理 NULL → nil。不要在 SQL 中用 `COALESCE(proxy_config, '{}'::jsonb)`，否则 WireGuard 行会拿到空 JSON 对象而非 nil。

**Warning signs:** WireGuard 类型的 EgressIP 返回时 `proxy_config` 显示为 `{}` 而非 null/omitted。

### Pitfall 2: 所有 EgressIP 查询都需要更新

**What goes wrong:** 只更新了 `CreateEgressIP` 和 `ListEgressIPs` 的 SQL，遗漏了 `GetEgressIP`、`GetEgressIPByHost`、`GetHostDetail` 中的 JOIN 查询。

**Why it happens:** `queries.go` 中有 7 处以上的 SQL 查询涉及 `egress_ips` 表的列列表。

**How to avoid:** 全文搜索 `FROM egress_ips` 和 `JOIN egress_ips`，确保每一处 SELECT 和 INSERT/UPDATE 都包含新列。具体需要更新的方法：

| 方法 | 操作 | 需要新增的列 |
|------|------|-------------|
| `ListEgressIPs` | SELECT | `tunnel_type`, `proxy_config` |
| `GetEgressIP` | SELECT | `tunnel_type`, `proxy_config` |
| `GetEgressIPByHost` | SELECT | `tunnel_type`, `proxy_config` |
| `GetHostDetail` | SELECT (JOIN) | `tunnel_type`, `proxy_config` |
| `CreateEgressIP` | INSERT + RETURNING | `tunnel_type`, `proxy_config` |
| `UpdateEgressIP` | UPDATE SET + RETURNING | `tunnel_type`, `proxy_config` |

**Warning signs:** 某些 API 返回的 EgressIP 缺少 `tunnel_type` 字段。

### Pitfall 3: EgressConfig.Tunnel 由值类型改为指针类型的级联影响

**What goes wrong:** `EgressConfig.Tunnel` 从 `TunnelSpec` 改为 `*TunnelSpec` 后，所有直接访问 `cfg.Tunnel.PrivateKey` 的代码会在 proxy 模式下 panic（nil pointer dereference）。

**Why it happens:** D-07 要求这个变更以区分两种隧道类型。现有 `tunnel_provider_linux.go` 中的 `PrepareHost` 会直接读取 `spec.Egress.Tunnel` 字段。

**How to avoid:** 
1. 在 `TunnelProvider.PrepareHost` 入口处加 nil 检查：`if spec.Egress.Tunnel == nil { return error }`
2. 或者依赖 Phase 8 的 Provider 工厂在调用前就按 `TunnelType` 路由，确保 `TunnelProvider` 永远不会收到 proxy 类型的 spec

**Warning signs:** 编译通过但 proxy 类型 host 启动时 panic。

### Pitfall 4: tunnel_type 未提供时的向后兼容

**What goes wrong:** 现有前端还没有 `tunnel_type` 字段（Phase 9 才做），如果 API handler 不设默认值，Create 请求会被 DB CHECK 约束拒绝或存入空字符串。

**Why it happens:** D-19 要求 `tunnel_type` 未提供时默认为 `wireguard`。

**How to avoid:** 在 API handler 的 Create/Update 中，如果 `req.TunnelType` 为空则设为 `"wireguard"`。同时 DB 列的 `DEFAULT 'wireguard'` 作为第二道防线。

**Warning signs:** 现有前端创建出口 IP 失败，400 错误。

### Pitfall 5: json.Number vs float64

**What goes wrong:** `json.Unmarshal` 到 `map[string]any` 时，JSON 数字默认解析为 `float64`。`server_port: 1080` 会变成 `float64(1080)`，类型断言 `parsed["server_port"].(int)` 会失败。

**Why it happens:** Go 的 `encoding/json` 标准行为。

**How to avoid:** 用 `parsed["server_port"].(float64)` 做断言，然后转为 `int` 检查范围（1-65535）。或用 `json.Decoder.UseNumber()` 解析为 `json.Number`，但 `map[string]any` 方式更简单直接。

**Warning signs:** `server_port` 校验始终报错"not a positive integer"。

### Pitfall 6: UPDATE 时 proxy_config 的条件写入

**What goes wrong:** `UpdateEgressIP` 现在对所有字段做无条件 SET。如果 `tunnel_type=wireguard` 的 Update 请求中 `proxy_config` 为 nil，会把现有的 proxy_config 覆盖为 NULL。

**Why it happens:** D-20/D-21 定义了 `tunnel_type=proxy` 时忽略 `wg_*` 字段，`tunnel_type=wireguard` 时忽略 `proxy_config`。但现有 SQL 是全字段 SET。

**How to avoid:** 在 API handler 层做清理：`tunnel_type=wireguard` 时强制 `proxy_config = nil`，`tunnel_type=proxy` 时强制 `wg_*` 字段全部 nil。这样 SQL 仍然是全字段 SET，但值已经在 handler 层被正确设置。

**Warning signs:** 编辑 WireGuard 类型出口 IP 后 proxy_config 从有值变为 NULL（虽然对 WG 类型影响不大，但数据一致性有问题）。

## Code Examples

### Migration SQL

```sql
-- 0004_proxy_tunnel.sql
-- 出口 IP 新增隧道类型和代理配置字段

ALTER TABLE egress_ips
    ADD COLUMN tunnel_type TEXT NOT NULL DEFAULT 'wireguard';

ALTER TABLE egress_ips
    ADD CONSTRAINT egress_ips_tunnel_type_check
    CHECK (tunnel_type IN ('wireguard', 'proxy'));

ALTER TABLE egress_ips
    ADD COLUMN proxy_config JSONB;
```

### Repository Model 扩展

```go
type EgressIP struct {
    ID             string          `json:"id"`
    Label          string          `json:"label"`
    IPAddress      string          `json:"ip_address"`
    Provider       string          `json:"provider"`
    Status         string          `json:"status"`
    TunnelType     string          `json:"tunnel_type"`
    WgEndpoint     *string         `json:"wg_endpoint,omitempty"`
    WgPublicKey    *string         `json:"wg_public_key,omitempty"`
    WgPresharedKey *string         `json:"wg_preshared_key,omitempty"`
    WgAllowedIPs   string          `json:"wg_allowed_ips"`
    WgDNSServer    *string         `json:"wg_dns_server,omitempty"`
    WgPeerAddress  *string         `json:"wg_peer_address,omitempty"`
    ProxyConfig    json.RawMessage `json:"proxy_config,omitempty"`
    CreatedAt      time.Time       `json:"created_at"`
    UpdatedAt      time.Time       `json:"updated_at"`
}
```

### Network Types 扩展

```go
const (
    TunnelTypeWireGuard = "wireguard"
    TunnelTypeProxy     = "proxy"
)

type ProxySpec struct {
    OutboundConfig json.RawMessage
    DNSServer      string
}

type EgressConfig struct {
    EgressIPID string
    ExpectedIP string
    TunnelType string
    Tunnel     *TunnelSpec
    Proxy      *ProxySpec
}
```

### repoValidator 适配

```go
func (rv *repoValidator) GetEgressIPByHost(ctx context.Context, hostID string) (network.EgressIPRecord, error) {
    eip, err := rv.repo.GetEgressIPByHost(ctx, hostID)
    if err != nil {
        return network.EgressIPRecord{}, err
    }
    return network.EgressIPRecord{
        ID:             eip.ID,
        IPAddress:      eip.IPAddress,
        TunnelType:     eip.TunnelType,
        WgEndpoint:     eip.WgEndpoint,
        WgPublicKey:    eip.WgPublicKey,
        WgPresharedKey: eip.WgPresharedKey,
        WgAllowedIPs:   eip.WgAllowedIPs,
        WgDNSServer:    eip.WgDNSServer,
        WgPeerAddress:  eip.WgPeerAddress,
        ProxyConfig:    eip.ProxyConfig,
    }, nil
}
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| pgx v4 需要 `pgtype.JSONB` | pgx v5 直接用 `[]byte` / `json.RawMessage` scan JSONB | pgx v5 (2023) | 不需要特殊类型适配器 |
| `database/sql` + 自定义 Scanner | pgx 原生类型映射 | 项目建立时已选择 pgx | 所有 JSONB 操作直接 scan |

## Open Questions

1. **Go 常量包归属**
   - What we know: 需要 `TunnelTypeWireGuard` 和 `TunnelTypeProxy` 两个常量
   - What's unclear: 放在 `network` 包还是 `repository` 包
   - Recommendation: **放在 `network` 包**。理由：`ValidateEgressBinding` 是消费这些常量的核心逻辑所在地，`repository` 包只做数据存取不应承载业务语义。API handler 可以 import `network` 包使用常量。如果为了避免 handler 依赖 network 包，可以在 handler 层定义局部常量或直接用字符串字面量（因为 DB CHECK 约束是最终防线）。

2. **sanitizeProxyConfig 策略选择**
   - What we know: 需要清理 `password` 类字段
   - What's unclear: 是否需要递归清理嵌套对象中的 password（如 `tls.password`）
   - Recommendation: **仅清理顶层 `password` 字段**。当前白名单的 5 种协议中，`password` 都在顶层。`tls` 对象中不包含密码（只有 `server_name`、`enabled` 等）。如果未来有嵌套密码需求，再扩展。

## Environment Availability

Step 2.6: SKIPPED（无外部依赖。本阶段纯 Go 代码 + SQL migration 变更，所有依赖已在 go.mod 中，不引入新工具或服务。）

## Validation Architecture

### Test Framework

| Property | Value |
|----------|-------|
| Framework | Go 标准库 `testing` |
| Config file | 无独立配置文件，Go test 标准行为 |
| Quick run command | `go test ./internal/...` |
| Full suite command | `go test ./...` |

### Phase Requirements → Test Map

| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| DATA-01 | 创建出口 IP 时可选择 tunnel_type | integration (API handler) | `go test ./internal/controlplane/http/ -run TestCreateEgressIP -x` | ❌ Wave 0 |
| DATA-02 | proxy 类型存储 proxy_config JSONB | unit (repository) | `go test ./internal/store/repository/ -run TestEgressIPProxyConfig -x` | ❌ Wave 0 |
| DATA-03 | ValidateEgressBinding 按 tunnel_type 分支 | unit (network) | `go test ./internal/network/ -run TestValidateEgressBinding -x` | ❌ Wave 0 |
| DATA-04 | proxy_config 白名单+必需字段校验 | unit (handler/validator) | `go test ./internal/controlplane/http/ -run TestValidateProxyConfig -x` | ❌ Wave 0 |
| DATA-05 | 现有 WG 数据 migration 后不丢失 | manual (migration 执行验证) | N/A — 通过 migration SQL 的 DEFAULT 子句保证 | manual-only |

### Sampling Rate

- **Per task commit:** `go build ./...`（编译通过即可）
- **Per wave merge:** `go test ./...`
- **Phase gate:** Full suite green before `/gsd-verify-work`

### Wave 0 Gaps

- [ ] `internal/network/validate_test.go` — 覆盖 DATA-03（ValidateEgressBinding proxy 分支）
- [ ] `internal/controlplane/http/admin_egress_ips_test.go` — 覆盖 DATA-01, DATA-04（API handler 校验）

注意：由于 repository 层测试需要真实 PostgreSQL 连接，DATA-02 的测试可以通过 API handler 层间接覆盖，或标记为 integration test。

## Sources

### Primary (HIGH confidence)

- 项目源码 `internal/store/repository/models.go` — 当前 EgressIP 结构体（已读取）
- 项目源码 `internal/store/repository/queries.go` — 当前 SQL 查询（已读取，7+ 处需更新）
- 项目源码 `internal/network/types.go` — 当前 EgressConfig / TunnelSpec 定义（已读取）
- 项目源码 `internal/network/validate.go` — 当前 ValidateEgressBinding 逻辑（已读取）
- 项目源码 `internal/controlplane/http/admin_egress_ips.go` — 当前 API handler（已读取）
- 项目源码 `internal/runtime/tasks/worker.go` — repoValidator 适配层（已读取）
- 项目 `go.mod` — 确认 pgx v5.7.6, Go 1.25.7（已读取）
- `.planning/proxy-tunnel-plan.md` — 完整代理隧道设计文档（已读取）

### Secondary (MEDIUM confidence)

- pgx v5 文档 — `[]byte` / `json.RawMessage` 与 JSONB 的映射是 pgx v5 标准行为，项目中 `events.metadata` 已验证此模式可行（`json.Marshal` → `[]byte` → INSERT → SELECT → `json.Unmarshal`）
- PostgreSQL 文档 — `ALTER TABLE ADD COLUMN ... DEFAULT` 对现有行立即生效是 PostgreSQL 11+ 的标准行为（本项目使用 PostgreSQL 18.3）

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH — 不引入新依赖，全部使用项目已有技术
- Architecture: HIGH — CONTEXT.md 已锁定所有关键结构决策，变更路径明确
- Pitfalls: HIGH — 基于实际源码分析，每个 pitfall 都可定位到具体文件和行号

**Research date:** 2026-03-28
**Valid until:** 2026-04-28（稳定技术栈，无过期风险）
