---
phase: 07-data-layer-typing
plan: 01
subsystem: database
tags: [postgres, migration, pgx, json, tunnel-type]

requires:
  - phase: 02-tunnel-egress
    provides: egress_ips 表和 WireGuard 列
provides:
  - tunnel_type 列（wireguard/proxy 枚举）
  - proxy_config JSONB 列
  - 扩展后的 EgressIP Go 模型（含 TunnelType + ProxyConfig）
  - 6 个更新后的 SQL 查询方法
affects: [07-02, 07-03, 08-singbox-provider, 09-frontend-form]

tech-stack:
  added: []
  patterns: [json.RawMessage 映射 JSONB NULL, defaultIfEmpty 保证默认值]

key-files:
  created:
    - internal/store/migrations/0004_proxy_tunnel.sql
  modified:
    - internal/store/repository/models.go
    - internal/store/repository/queries.go

key-decisions:
  - "proxy_config 直接 Scan 到 json.RawMessage，NULL 映射为 nil，不使用 COALESCE"
  - "tunnel_type 使用 NOT NULL DEFAULT 'wireguard'，现有行自动回填，无需 UPDATE"
  - "CreateEgressIP 和 UpdateEgressIP 通过 defaultIfEmpty 保证空值时默认为 wireguard"

patterns-established:
  - "JSONB 可空列映射：pgx 直接 Scan 到 json.RawMessage，NULL→nil"
  - "枚举列模式：TEXT + CHECK + DEFAULT，Go 侧 string 类型"

requirements-completed: [DATA-01, DATA-02, DATA-05]

duration: 2min
completed: 2026-03-28
---

# Phase 07 Plan 01: 数据层隧道类型扩展 Summary

**egress_ips 表新增 tunnel_type（wireguard/proxy CHECK 约束）和 proxy_config JSONB 列，Go 模型和全部 6 个 SQL 查询同步扩展**

## Performance

- **Duration:** 2 min
- **Started:** 2026-03-28T06:18:28Z
- **Completed:** 2026-03-28T06:20:47Z
- **Tasks:** 2
- **Files modified:** 3

## Accomplishments
- 创建 0004_proxy_tunnel.sql migration，为 egress_ips 表添加 tunnel_type 和 proxy_config 两列
- EgressIP、CreateEgressIPParams、UpdateEgressIPParams 三个结构体均已扩展 TunnelType 和 ProxyConfig 字段
- ListEgressIPs、GetEgressIP、GetEgressIPByHost、GetHostDetail、CreateEgressIP、UpdateEgressIP 六个方法的 SQL 和 Scan 全部更新
- `go build ./internal/store/...` 编译通过

## Task Commits

Each task was committed atomically:

1. **Task 1: 创建 0004_proxy_tunnel.sql 数据库 migration** - `d33365d` (feat)
2. **Task 2: 扩展 Repository 数据模型和 SQL 查询** - `e5286a6` (feat)

## Files Created/Modified
- `internal/store/migrations/0004_proxy_tunnel.sql` - tunnel_type + proxy_config 列的 DDL
- `internal/store/repository/models.go` - EgressIP / CreateEgressIPParams / UpdateEgressIPParams 扩展字段
- `internal/store/repository/queries.go` - 6 个 EgressIP 查询方法同步更新

## Decisions Made
- proxy_config 直接 Scan 到 `json.RawMessage`，NULL 自动映射为 nil，不使用 COALESCE（避免 WireGuard 行返回空 JSON 对象）
- tunnel_type 使用 `NOT NULL DEFAULT 'wireguard'`，现有行在 migration 后自动获得默认值，不需要额外 UPDATE 语句
- CREATE 和 UPDATE 操作通过 `defaultIfEmpty(params.TunnelType, "wireguard")` 保证空值时默认为 wireguard

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered

None

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness
- 数据层基础就绪，Plan 02（网络校验分支）和 Plan 03（API 适配）可以在此基础上进行
- tunnel_type CHECK 约束确保数据完整性，后续 Provider 工厂可据此路由

---
*Phase: 07-data-layer-typing*
*Completed: 2026-03-28*
