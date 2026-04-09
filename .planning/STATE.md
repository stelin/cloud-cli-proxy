# Project State

## Project Reference

See: .planning/PROJECT.md (updated 2026-04-09)

**Core value:** 单一二进制替换 claude 命令，透明启动 Docker 容器运行 Claude Code，所有网络流量走代理出口，设备指纹完全伪装
**Current focus:** Phase 17 — 镜像与 Entrypoint 基线

## Current Position

Phase: 17 of 23 (镜像与 Entrypoint 基线)
Plan: 0 of 0 in current phase
Status: Ready to plan
Last activity: 2026-04-09 — v1.3 roadmap created

Progress: [░░░░░░░░░░] 0%

## Performance Metrics

**Velocity:**
- Total plans completed: 0 (v1.3)
- Average duration: —
- Total execution time: 0 hours

**By Phase:**

| Phase | Plans | Total | Avg/Plan |
|-------|-------|-------|----------|
| — | — | — | — |

**Recent Trend:**
- Last 5 plans: —
- Trend: —

*Updated after each plan completion*

## Accumulated Context

### Decisions

Decisions are logged in PROJECT.md Key Decisions table.
Recent decisions affecting current work:

- [v1.3 research]: 网络架构采用 bridge + 容器内 sing-box tun + route-based split tunneling
- [v1.3 research]: Claude Code 安装走官方 curl 脚本（Bun standalone），不依赖 npm 或 spoof.js
- [v1.3 research]: /proc 伪造优先用 docker run -v 注入，避免容器内 mount 或滥用 --privileged
- [v1.3 research]: CLI 先用 docker run 子进程打通，后续可收敛到 Docker SDK

### Pending Todos

None yet.

### Blockers/Concerns

- [research]: Docker Desktop vs Linux Engine 的 host-gateway、nft 语义差异需在 Phase 18 验证
- [research]: garble 与 Docker client 依赖反射的组合需在 Phase 23 验证

## Session Continuity

Last session: 2026-04-09
Stopped at: Phase 17 context gathered
Resume file: .planning/phases/17-image-entrypoint-baseline/17-CONTEXT.md
