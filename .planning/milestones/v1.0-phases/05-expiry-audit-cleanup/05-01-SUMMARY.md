---
phase: 05-expiry-audit-cleanup
plan: 01
subsystem: infra
tags: [scheduler, expiry, postgres, migration, admin-api]

requires:
  - phase: 04-admin-panel
    provides: admin API framework, user/host management endpoints
provides:
  - DB migration with expires_at field, events.user_id column, and query indexes
  - Generic background scheduler framework (scheduler package)
  - Expiry scanner that marks expired users and stops their running hosts
  - Admin API for managing user expiry and reactivating expired users
  - Repository methods for expiry queries, event listing, host status, and stale task cleanup
affects: [05-expiry-audit-cleanup, admin-panel, controlplane]

tech-stack:
  added: []
  patterns: [background-scheduler-with-ticker, expiry-scan-idempotent, partial-index-for-scan]

key-files:
  created:
    - internal/store/migrations/0003_expiry_audit.sql
    - internal/controlplane/scheduler/scheduler.go
    - internal/controlplane/scheduler/expiry.go
  modified:
    - internal/store/repository/models.go
    - internal/store/repository/queries.go
    - internal/controlplane/http/admin_users.go
    - internal/controlplane/http/router.go
    - internal/controlplane/app/app.go

key-decisions:
  - "Scheduler uses per-job goroutine with time.Ticker and WaitGroup for graceful shutdown"
  - "Expiry scan default interval 60s, configurable via Config.ExpiryScanInterval"
  - "Partial index idx_users_expires_at_status for efficient expiry scan on active users with expires_at set"
  - "UpdateStatus handler allows expired→active transition without checking current status (target validation only)"

patterns-established:
  - "scheduler.Job pattern: Name + Interval + Fn(ctx) error for pluggable background tasks"
  - "Interface segregation: ExpiryStore and HostActionQueuer as narrow dependency interfaces"

requirements-completed: [LIFE-04, LIFE-05]

duration: 8min
completed: 2026-03-27
---

# Phase 5 Plan 01: Expiry/Audit Data Foundation & Scheduler Summary

**DB migration with expires_at/user_id fields, generic ticker-based scheduler, expiry scanner with auto-stop, and admin expiry API endpoints**

## Performance

- **Duration:** 8 min
- **Started:** 2026-03-27T12:00:00Z
- **Completed:** 2026-03-27T12:08:00Z
- **Tasks:** 2
- **Files modified:** 8

## Accomplishments
- Database migration adding expires_at to users, user_id to events, and comprehensive query indexes
- Generic background scheduler framework supporting multiple independent interval-based jobs
- Expiry scanner that atomically marks expired users and stops their running hosts via QueueHostAction
- Admin API endpoint for setting/clearing user expiry time with future-date validation
- Seven new repository methods: ListExpiredActiveUsers, UpdateUserExpiry, ListRunningHostsByUserID, ListRunningHosts, ListEvents, UpdateHostStatus, MarkStaleTasks

## Task Commits

Each task was committed atomically:

1. **Task 1: DB 迁移与 Repository 层扩展** - `6762c08` (feat)
2. **Task 2: 调度框架、到期扫描器与管理 API 到期支持** - `fba0efb` (feat)

## Files Created/Modified
- `internal/store/migrations/0003_expiry_audit.sql` - Adds expires_at, events.user_id, and query/scan indexes
- `internal/store/repository/models.go` - User.ExpiresAt, Event.UserID, ListEventsParams/Result types
- `internal/store/repository/queries.go` - Seven new query methods plus expires_at in all User queries
- `internal/controlplane/scheduler/scheduler.go` - Generic Job/Scheduler with ticker and graceful shutdown
- `internal/controlplane/scheduler/expiry.go` - ExpiryScanner with user.expired and host.stop.expired events
- `internal/controlplane/http/admin_users.go` - UpdateExpiry handler, UpdateUserExpiry in interface
- `internal/controlplane/http/router.go` - PUT /v1/admin/users/{userID}/expiry route
- `internal/controlplane/app/app.go` - Config fields, ExpiryScanner creation, scheduler startup in Run()

## Decisions Made
- Scheduler uses per-job goroutines with time.Ticker; WaitGroup ensures all goroutines complete before shutdown
- Default 60s expiry scan interval, configurable via ExpiryScanInterval
- Partial index on (expires_at, status) WHERE expires_at IS NOT NULL AND status = 'active' for scan efficiency
- UpdateStatus validates target status only (active/disabled), naturally allowing expired→active reactivation

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered
None

## User Setup Required
None - no external service configuration required.

## Next Phase Readiness
- Expiry data foundation and scheduler framework complete
- Ready for Plan 02 (reconciler, event audit API, and cleanup logic)
- ReconcileInterval config field already in place for Plan 02's reconciler job

---
*Phase: 05-expiry-audit-cleanup*
*Completed: 2026-03-27*
