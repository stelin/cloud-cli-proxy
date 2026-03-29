---
phase: 06-mvp
plan: 04
subsystem: infra, api, ui
tags: [slog, healthz, pgxpool, bootstrap, wireguard, zod, security]

requires:
  - phase: 06-mvp
    provides: "控制面 API 和后台管理界面基础"
provides:
  - "LOG_FORMAT=json 和 LOG_LEVEL 环境变量支持（控制面 + host-agent）"
  - "增强型 /healthz 端点含 database + agent 分组检查"
  - "显式 pgxpool 连接池配置和优雅关闭"
  - "bootstrap 脚本完善的 HTTP 错误码解析和中文提示"
  - "EgressIP API 响应中 WgPresharedKey 字段清除"
  - "出口 IP 表单前端 IPv4/endpoint/CIDR 格式校验"
affects: [deployment, monitoring, security]

tech-stack:
  added: []
  patterns: [newLogger() 提供可配置日志, sanitizeEgressIP 清除响应中的敏感字段]

key-files:
  created: []
  modified:
    - internal/controlplane/app/app.go
    - cmd/host-agent/main.go
    - internal/controlplane/http/router.go
    - internal/agentapi/client.go
    - internal/agent/server.go
    - deploy/bootstrap/cloud-bootstrap.sh
    - internal/controlplane/http/admin_egress_ips.go
    - web/admin/src/components/egress-ips/egress-ip-drawer.tsx

key-decisions:
  - "newLogger() 抽取为独立函数，控制面和 host-agent 共享相同逻辑"
  - "pgxpool 配置 MaxConns=10 MinConns=2，适合单宿主机 MVP 场景"
  - "/healthz 返回 degraded 而非 error，区分部分可用和全挂状态"
  - "bootstrap 脚本使用 curl -w '%{http_code}' 替代 --fail，保留响应体供错误码解析"
  - "sanitizeEgressIP 统一在 handler 层清除 WgPresharedKey，不改动 model 的 json tag"

patterns-established:
  - "newLogger(): LOG_FORMAT=json 时使用 JSONHandler，LOG_LEVEL 控制日志级别"
  - "sanitizeEgressIP: 所有返回 EgressIP 的 API handler 在 writeJSON 前调用"

requirements-completed: [ACCS-01, ADMN-03, ADMN-04]

duration: 5min
completed: 2026-03-27
---

# Phase 06 Plan 04: 终端/后台体验打磨与上线前安全、稳定性、可运维检查 Summary

**结构化日志 + healthz 分组检查 + pgxpool 显式配置 + bootstrap 错误码解析 + EgressIP 敏感字段清除 + 前端表单格式校验**

## Performance

- **Duration:** 5 min
- **Started:** 2026-03-27T17:39:27Z
- **Completed:** 2026-03-27T17:44:15Z
- **Tasks:** 3
- **Files modified:** 8

## Accomplishments
- 控制面和 host-agent 支持 LOG_FORMAT=json 结构化日志和 LOG_LEVEL 日志级别控制
- /healthz 增强为 database + agent 分组健康检查，返回 degraded/ok 状态
- pgxpool 显式配置 MaxConns=10, MinConns=2, 连接生命周期和空闲超时
- 控制面 Run() 退出时通过 defer db.Close() 释放连接池
- bootstrap 脚本三个 curl 调用全部替换为 -w '%{http_code}' 模式，支持 HTTP 4xx/5xx 错误码解析
- EgressIP API 所有响应（List/Get/Create/Update）通过 sanitizeEgressIP 清除 WgPresharedKey
- 出口 IP 表单添加 IPv4 格式校验、WG endpoint host:port 格式校验、WG peer address CIDR 格式校验

## Task Commits

Each task was committed atomically:

1. **Task 1: 后端运维加固 — 结构化日志 + healthz 增强 + 连接池 + 优雅关闭** - `b92257f` (feat)
2. **Task 2: 安全审查与终端体验打磨** - `c436c03` (fix)
3. **Task 3: 后台 UI 体验打磨** - `816b0ce` (feat)

## Files Created/Modified
- `internal/controlplane/app/app.go` - newLogger(), 显式 pgxpool 配置, defer db.Close()
- `cmd/host-agent/main.go` - newLogger() + slog.SetDefault()
- `internal/controlplane/http/router.go` - AgentHealthChecker 接口, /healthz 分组检查
- `internal/agentapi/client.go` - Ping() 方法用于 agent 健康检查
- `internal/agent/server.go` - /healthz 端点
- `deploy/bootstrap/cloud-bootstrap.sh` - HTTP 错误码解析, 中文提示, 重试建议
- `internal/controlplane/http/admin_egress_ips.go` - sanitizeEgressIP 清除 WgPresharedKey
- `web/admin/src/components/egress-ips/egress-ip-drawer.tsx` - IPv4/endpoint/CIDR 格式校验

## Decisions Made
- newLogger() 抽取为独立函数，控制面和 host-agent 共享相同逻辑
- pgxpool 配置 MaxConns=10 MinConns=2，适合单宿主机 MVP 场景
- /healthz 返回 degraded 而非 error，区分部分可用和全挂状态
- bootstrap 脚本使用 curl -w '%{http_code}' 替代 --fail，保留响应体供错误码解析
- sanitizeEgressIP 统一在 handler 层清除 WgPresharedKey，不改动 model 的 json tag

## Deviations from Plan

None - plan executed exactly as written.

## Security Audit Results (D-13)

已确认以下安全项：
- ✅ User 结构体不包含 PasswordHash（仅在 BootstrapUserAuth 中）
- ✅ AdminConfig.JWTSecret 不在任何 API 响应中
- ✅ EgressIP.WgPresharedKey — 本计划已修复，通过 sanitizeEgressIP 清除
- ✅ 日志中不打印密码（admin_users.go 只 log error，不 log 请求体）
- ✅ API 权限边界正确：/healthz 公开，bootstrap 凭证认证，admin 路由 JWT 保护

## Issues Encountered
- host-agent 有 `//go:build linux` 约束，macOS 上使用 `GOOS=linux go build` 编译验证

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness
- 后端运维加固、安全审查和体验打磨已完成
- 所有验证（go build + go vet + bash -n + tsc --noEmit）通过
- MVP 阶段所有计划执行完毕

## Self-Check: PASSED

All 8 modified files verified present. All 3 task commits (b92257f, c436c03, 816b0ce) verified in git log.

---
*Phase: 06-mvp*
*Completed: 2026-03-27*
