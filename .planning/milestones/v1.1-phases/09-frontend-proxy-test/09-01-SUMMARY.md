---
phase: 09-frontend-proxy-test
plan: 01
subsystem: api
tags: [proxy, socks5, http-connect, sing-box, egress-ip, react-query]

requires:
  - phase: 07-proxy-data-model
    provides: EgressIP model with tunnel_type and proxy_config fields
  - phase: 08-singbox-provider
    provides: sing-box binary in managed image, ProxySpec and outbound config patterns

provides:
  - POST /v1/admin/egress-ips/{ipID}/test endpoint with three-check probe suite
  - ProbeResult / ConnectivityCheckResult / EgressIPCheckResult / DNSLeakCheckResult Go types
  - Proxy dialer factory supporting SOCKS5, HTTP CONNECT, and sing-box local forwarding
  - Frontend TestResult interface and useTestEgressIP mutation hook
  - Expanded EgressIP TS interface with tunnel_type and proxy_config fields

affects: [09-02, 09-03, frontend-proxy-test]

tech-stack:
  added: [golang.org/x/net/proxy (direct usage)]
  patterns: [proxy dialer factory, sing-box local SOCKS5 inbound for protocol testing, multi-source IP detection]

key-files:
  created:
    - internal/controlplane/http/admin_egress_ip_probe.go
  modified:
    - internal/controlplane/http/router.go
    - web/admin/src/hooks/use-egress-ips.ts

key-decisions:
  - "SOCKS5 和 HTTP 协议通过 Go 标准库直接拨号，vmess/ss/trojan 通过临时 sing-box 本地 SOCKS5 转发"
  - "连通性使用 Google 204 检测，出口 IP 使用 ipify/ip.me/ifconfig.me 多源交叉验证"
  - "DNS 泄漏通过 ipleak.net JSON 端点验证代理 DNS 路径"

patterns-established:
  - "Proxy dialer factory: getProxyDialer 按协议类型分发，返回 contextDialer + cleanup"
  - "sing-box probe 模式: 临时 SOCKS5 inbound + 协议 outbound，5s 启动超时"

requirements-completed: [TEST-01, TEST-02, TEST-03, TEST-04]

duration: 4min
completed: 2026-03-28
---

# Phase 09 Plan 01: Proxy Test API + Frontend Types Summary

**代理测试 API 支持 SOCKS5/HTTP/vmess/ss/trojan 五种协议，返回连通性、出口 IP 匹配、DNS 泄漏三项检测结果，前端 TestResult 类型和 mutation hook 就绪**

## Performance

- **Duration:** 4 min
- **Started:** 2026-03-28T08:06:50Z
- **Completed:** 2026-03-28T08:11:21Z
- **Tasks:** 2
- **Files modified:** 3

## Accomplishments

- 实现 POST /v1/admin/egress-ips/{ipID}/test 端点，30 秒总超时，adminAuth 保护
- 代理拨号工厂支持 SOCKS5（golang.org/x/net/proxy）、HTTP CONNECT（自实现）、vmess/ss/trojan（sing-box 本地转发）
- 三项检测套件：连通性（Google 204）、出口 IP 匹配（ipify/ip.me/ifconfig.me 多源）、DNS 泄漏（ipleak.net）
- 前端 EgressIP 接口扩展 tunnel_type/proxy_config，TestResult 接口完整映射后端 ProbeResult

## Task Commits

Each task was committed atomically:

1. **Task 1: Go 代理测试 handler + 路由注册** - `3bf7f63` (feat)
2. **Task 2: 前端 EgressIP 类型扩展 + TestResult + useTestEgressIP hook** - `6bf2751` (feat)

## Files Created/Modified

- `internal/controlplane/http/admin_egress_ip_probe.go` - 代理测试 handler，包含 ProbeResult 类型、proxy dialer 工厂、三项检测函数
- `internal/controlplane/http/router.go` - 注册 POST /v1/admin/egress-ips/{ipID}/test 路由
- `web/admin/src/hooks/use-egress-ips.ts` - 扩展 EgressIP 接口，新增 TestResult 接口和 useTestEgressIP hook

## Decisions Made

- SOCKS5 和 HTTP 协议通过 Go 标准库直接拨号，不依赖外部二进制
- vmess/shadowsocks/trojan 通过临时 sing-box 实例本地 SOCKS5 转发测试，复用现有 sing-box 二进制
- 连通性检测使用 Google connectivitycheck 204 端点（低延迟、高可用）
- 出口 IP 使用三个独立源交叉验证，取第一个成功结果作为实际 IP
- DNS 泄漏通过 ipleak.net JSON 端点验证域名解析走代理路径

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered

None

## User Setup Required

None - no external service configuration required.

## Known Stubs

None - all data paths are wired to real API endpoints.

## Next Phase Readiness

- 后端代理测试 API 就绪，Plan 02 可构建前端测试 UI 组件
- TestResult 类型和 useTestEgressIP hook 已导出，Plan 02/03 可直接引用

---
*Phase: 09-frontend-proxy-test*
*Completed: 2026-03-28*
