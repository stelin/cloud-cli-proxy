---
phase: 01-foundation-control-plane-host-agent
plan: "01"
subsystem: api
tags: [go, postgres, pgx, docker-compose, migrations]
requires: []
provides:
  - control-plane service bootstrap with migration-on-start behavior
  - PostgreSQL schema for users, hosts, egress IPs, tasks, and events
  - single-host development compose stack for postgres plus control-plane
affects: [runtime, host-agent, lifecycle-api, phase-2-networking]
tech-stack:
  added: [Go, pgx/v5, PostgreSQL 18.3, Docker Compose]
  patterns: [stdlib-http-control-plane, startup-migrations, repository-layer]
key-files:
  created:
    - cmd/control-plane/main.go
    - internal/controlplane/app/app.go
    - internal/controlplane/http/router.go
    - internal/store/migrations/0001_initial.sql
    - deploy/compose/control-plane.dev.yml
  modified:
    - go.mod
    - go.sum
    - internal/store/repository/queries.go
    - internal/store/repository/models.go
    - internal/store/migrator/migrator.go
    - deploy/docker/control-plane/Dockerfile
key-decisions:
  - "控制面优先使用 Go 标准库 `net/http`，先把 HTTP 骨架和宿主机边界落稳。"
  - "数据库 migration 在 control-plane 启动时执行，避免额外的 Phase 1 运维步骤。"
  - "Compose 构建上下文指向仓库根目录，以保证 `deploy/compose` 下的文件能稳定构建控制面镜像。"
patterns-established:
  - "Pattern: 控制面通过 repository 访问数据库，不直接在 HTTP 层拼 SQL。"
  - "Pattern: 启动入口先迁移数据库，再暴露 API 端口。"
requirements-completed: [RUNT-01]
duration: 3 min
completed: 2026-03-26
---

# Phase 01 Plan 01: 控制面基础盘 Summary

**基于 Go 标准库的 control-plane 启动骨架、PostgreSQL 核心 schema 与单宿主机开发编排**

## Performance

- **Duration:** 3 min
- **Started:** 2026-03-26T16:46:15+08:00
- **Completed:** 2026-03-26T16:49:33+08:00
- **Tasks:** 3
- **Files modified:** 11

## Accomplishments
- 交付了可读取 `CONTROL_PLANE_ADDR`、`DATABASE_URL` 并在启动前运行 `RunMigrations` 的 control-plane 入口。
- 定义了 Phase 1 需要的六张 PostgreSQL 核心表、索引和 repository 契约。
- 补齐了 `postgres + control-plane` 的开发 compose 与 Dockerfile，让控制面具备单宿主机联调入口。

## Task Commits

Each task was committed atomically:

1. **Task 1: 初始化 Go 控制面服务骨架** - `9effa1c` (feat)
2. **Task 2: 定义 PostgreSQL 核心 schema 与仓储契约** - `ccc5d77` (feat)
3. **Task 3: 补齐单宿主机开发部署布局** - `67ebe10` (feat)

**Plan metadata:** pending

## Files Created/Modified
- `cmd/control-plane/main.go` - 控制面唯一启动入口与环境变量读取
- `internal/controlplane/app/app.go` - 组装 logger、repository、migrator 与 router
- `internal/controlplane/http/router.go` - 健康检查、用户/主机列表与 `/v1/tasks` handler 插槽
- `internal/store/migrations/0001_initial.sql` - 用户、主机、出口 IP、绑定、任务、事件 schema
- `internal/store/repository/queries.go` - Phase 1 repository 查询与写入契约
- `deploy/docker/control-plane/Dockerfile` - 控制面构建镜像
- `deploy/compose/control-plane.dev.yml` - 本地 postgres + control-plane 编排

## Decisions Made
- 使用 `pgxpool` 直接连接 PostgreSQL，先保持依赖面精简。
- `tasks` 表直接保留 `error_code`、`error_message`、`last_error_summary`，为后续 agent 回写留出完整错误链路。
- 在 compose 中只编排 `postgres` 与 `control-plane`，不提前塞入 host-agent 或网络组件。

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] 调整 compose 构建上下文到仓库根目录**
- **Found during:** Task 3（补齐单宿主机开发部署布局）
- **Issue:** `deploy/compose` 下的 compose 文件如果直接使用相对本目录的 `build.context: .`，无法稳定访问仓库根目录源码来构建控制面镜像。
- **Fix:** 将构建上下文指向仓库根目录，同时保留 `deploy/docker/control-plane/Dockerfile` 作为镜像定义入口。
- **Files modified:** `deploy/compose/control-plane.dev.yml`
- **Verification:** `docker compose -f deploy/compose/control-plane.dev.yml config`
- **Committed in:** `67ebe10`

---

**Total deviations:** 1 auto-fixed (1 blocking)
**Impact on plan:** 仅修正本地构建路径，未扩大范围，反而让 compose 配置更可执行。

## Issues Encountered
- 本机 Docker daemon 不可连接，导致计划要求的 `docker compose up ... && psql ... && curl /healthz` 实机验证无法完成；已完成 compose 配置解析校验与代码编译校验。

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness
- 受管镜像与 host-agent 可以直接复用本计划建立的 schema、repository 和 control-plane 启动骨架。
- 若要重做实机 compose 验证，需要先在 Linux 或可用的 Docker 守护进程环境中启动容器。

---
*Phase: 01-foundation-control-plane-host-agent*
*Completed: 2026-03-26*
