---
phase: 08-singboxprovider
plan: 01
subsystem: network
tags: [sing-box, tun, proxy, dns-hijack, docker]

requires:
  - phase: 07-proxy-data-model
    provides: ProxySpec 和 EgressConfig 类型定义、validateProxyBinding 填充逻辑
provides:
  - sing-box 配置结构体和 JSON 生成函数 (buildSingBoxConfig)
  - outbound JSON 合并函数 (buildOutbound)，自动清理 dns_server 并注入 tag/bind_interface
  - 代理服务器地址提取函数 (extractProxyServer)，支持域名解析
  - 受管镜像预装 sing-box v1.13.3 二进制和 /etc/sing-box 配置目录
affects: [08-02, 08-03, singbox-provider, firewall-proxy]

tech-stack:
  added: [sing-box v1.13.3]
  patterns: [sing-box JSON 配置生成、outbound 合并注入模式]

key-files:
  created: [internal/network/singbox_config.go]
  modified: [deploy/docker/managed-user/Dockerfile]

key-decisions:
  - "使用 Go map[string]any 序列化 tun inbound 和 direct outbound，保持灵活性"
  - "extractProxyServer 使用 net.ResolveIPAddr 解析域名，确保 host route 使用 IP 地址"
  - "sing-box 版本通过 Dockerfile ARG 管理，便于后续升级"

patterns-established:
  - "sing-box 配置生成：结构体模板 + 用户 outbound JSON 合并"
  - "dns_server 字段清理：从 ProxySpec.OutboundConfig 中删除非 sing-box 字段"

requirements-completed: [SING-05, SING-01, SING-02]

duration: 2min
completed: 2026-03-28
---

# Phase 08 Plan 01: sing-box 配置生成与受管镜像预装 Summary

**sing-box 配置结构体 + JSON 生成函数（tun inbound / proxy outbound / DNS hijack）及受管镜像 v1.13.3 二进制预装**

## Performance

- **Duration:** 2 min
- **Started:** 2026-03-28T07:11:20Z
- **Completed:** 2026-03-28T07:12:57Z
- **Tasks:** 2
- **Files modified:** 2

## Accomplishments
- 完整的 sing-box 配置生成管线：tun inbound（auto_route + strict_route）、proxy outbound（bind_interface=mgmt0）、DNS hijack 路由规则
- buildOutbound 自动清理 db 中的 dns_server 字段并注入 tag/bind_interface
- extractProxyServer 支持 IP 和域名解析，为后续 host route 和 nftables 白名单提供地址
- 受管镜像 Dockerfile 预装 sing-box v1.13.3，版本通过 ARG 管理

## Task Commits

Each task was committed atomically:

1. **Task 1: sing-box 配置结构体与生成函数** - `dfa2b53` (feat)
2. **Task 2: 受管镜像预装 sing-box 二进制** - `65b2c90` (feat, 已在前序执行中完成)

## Files Created/Modified
- `internal/network/singbox_config.go` - sing-box 配置结构体（singBoxConfig 等）和三个核心函数
- `deploy/docker/managed-user/Dockerfile` - 预装 sing-box v1.13.3 二进制和 /etc/sing-box 配置目录

## Decisions Made
- 使用 `address` 字段（非已废弃的 `inet4_address`），符合 sing-box v1.12+ 规范
- TUN 地址选择 `172.18.0.1/30`，不与管理 veth 子网 `10.99.0.0/16` 冲突
- DNS strategy 设为 `ipv4_only`，与项目 IPv4-only 约束一致
- tun stack 选择 `system`，容器环境下最简单高效

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered
None

## User Setup Required
None - no external service configuration required.

## Next Phase Readiness
- sing-box 配置生成函数已就绪，08-02（SingBoxProvider 进程管理和防火墙）可直接调用 buildSingBoxConfig 和 extractProxyServer
- Dockerfile 已预装 sing-box 二进制，容器启动后无需运行时下载

## Self-Check: PASSED

---
*Phase: 08-singboxprovider*
*Completed: 2026-03-28*
