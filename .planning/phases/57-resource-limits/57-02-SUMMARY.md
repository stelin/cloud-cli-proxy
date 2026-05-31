---
phase: 57-resource-limits
plan: 02
subsystem: api
tags: [go, nethttp, pointer-semantics, patch, storage-opt, docker]

requires:
  - phase: 57-01
    provides: "*int/*float64 指针类型 + UpdateHostResources Repository 方法"
provides:
  - "Create 端点三态指针语义（nil=默认 / 0=无限制 / 正值=限制）"
  - "PatchResources handler（状态检查 + 范围验证 + 事件记录）"
  - "Worker --storage-opt 磁盘限制参数生成"
  - "移除 runtime_service 双层默认值兜底"
affects: [57-03-ui]

tech-stack:
  added: []
  patterns:
    - "API 层三态解析函数：Create（nil→默认）vs PATCH（nil→不修改）两种语义"
    - "Worker 层 0=不传参数（Docker 无限制），>0=传具体参数"

key-files:
  created: []
  modified:
    - internal/agentapi/contracts.go
    - internal/runtime/runtime_service.go
    - internal/controlplane/http/admin_hosts.go
    - internal/controlplane/http/router.go
    - internal/runtime/tasks/worker.go

key-decisions:
  - "HostActionRequest 资源字段保持非指针（int/float64），三态解析在 API 层完成"
  - "Worker 层移除 else 默认值分支：0 值不传 Docker 参数即为无限制"
  - "PATCH 范围验证：内存 128-262144 MB、CPU 0.1-64 核、磁盘 1-2048 GB"

requirements-completed:
  - RES-02
  - RES-04

duration: 20min
completed: 2026-05-31
---

# Phase 57 Plan 02: API 层 + Worker 层 Summary

**Create 端点三态指针语义、PatchResources handler（状态+范围验证）、Worker --storage-opt 磁盘限制**

## Performance

- **Duration:** ~20min
- **Tasks:** 3
- **Files modified:** 5

## Accomplishments

- contracts.go：HostActionRequest 新增 DiskLimitGB 字段
- runtime_service.go：移除 defaultIntIfZero/defaultFloatIfZero 双层默认值兜底，改用 ptrToInt/ptrToFloat 直接透传
- admin_hosts.go：Create body 资源字段改为指针类型（*int/*float64），resolveMemory/resolveCPU/resolveDisk 三态解析；新增 PatchResources handler（status 检查返回 409、范围验证、事件记录）、AdminHostStore 接口扩展
- router.go：注册 PATCH /v1/admin/hosts/{hostID}/resources 路由
- worker.go：移除内存/CPU 的 else 默认值分支，新增 DiskLimitGB > 0 时传 --storage-opt size=XG

## Task Commits

1. **Task 1: contracts.go + runtime_service.go** — `bc9dd66` (feat)
2. **Task 2: admin_hosts.go Create + PatchResources** — `e0b99fe` (feat)
3. **Task 3: router.go + worker.go** — `f21b6ea` (feat)

## Decisions Made

- Worker 移除 else 默认值分支：0 值不传 Docker 参数即为无限制，与移除的双层默认值兜底对齐
- Create 和 PATCH 的三态解析语义不同：Create 的 nil→默认值（4096/2.0/20），PATCH 的 nil→不修改该字段，因此用了两套解析函数

## Deviations from Plan

None — plan executed exactly as written.

## Issues Encountered

None

## Next Phase Readiness

后端 API + Worker 层完整就绪。Phase 57 三个计划全部完成，可进行端到端验证。

---
*Phase: 57-resource-limits*
*Completed: 2026-05-31*
