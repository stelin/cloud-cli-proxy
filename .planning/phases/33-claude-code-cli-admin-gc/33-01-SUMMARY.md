---
phase: 33-claude-code-cli-admin-gc
plan: 01
subsystem: infra
tags: [docker-volume, agentapi, worker, claude-account, entrypoint, persistent-state]

requires:
  - phase: 29-v3-image-host-agent
    provides: "managed-user 镜像 entrypoint v3 stage 串行编排 + Volumes mount 解析（HostActionRequest.Volumes）"
  - phase: 30-entry-api
    provides: "ClaudeAccount 数据模型（含 PersistentVolumeName *string 三态字段）+ HostActionRequest.ClaudeAccountID 协议字段"
provides:
  - "entrypoint prepare_persistent_state：~/.claude / ~/.cache/claude → /var/lib/claude-persist symlink + 1000:1000 chown 兜底（D-08/D-09/D-10）"
  - "agentapi.ActionVolumeRemove = volume_remove 协议常量（D-13），供 Plan 02 admin DELETE 触发 host-agent 删 volume"
  - "BuildClaudeStateVolumeName / ensureDockerVolume / removeDockerVolume / removeVolumes worker 工具链（D-04/D-13/D-14/D-15）"
  - "Worker.createHost 在 ClaudeAccountID 非空时自动 ensureDockerVolume + 追加 claude-state mount + UpsertClaudeAccountPersistentVolumeName（D-04/D-05/D-06/D-07）"
  - "Repository.UpsertClaudeAccountPersistentVolumeName 三态语义：NULL→写入 / 一致跳过 / 冲突错误（D-06）"
  - "WorkerRepo 接口扩展契约（新增方法 UpsertClaudeAccountPersistentVolumeName）"
affects: [33-02-admin-delete-host-detail-uat, 34-doctor-error-codes, 35-e2e-stability]

tech-stack:
  added: []
  patterns:
    - "包级 var 注入 mock：dockerVolumeRunner / ensureDockerVolume / removeDockerVolume（沿用 var execInContainer / var syncContainerPassword 模式）"
    - "幂等 docker volume 子命令：inspect 成功跳过 create / 'no such volume' 视为成功 / 'volume is in use' 包装为前缀错误"
    - "audit event metadata 白名单：account_id / volume_name / force / host_id（凭据守恒，T-33-04 mitigation）"

key-files:
  created:
    - "internal/runtime/tasks/worker_volume_lifecycle_test.go (7 lifecycle 单测)"
    - "internal/store/repository/queries_claude_account_volume_test.go (3 SQL/边界单测)"
  modified:
    - "deploy/docker/managed-user/entrypoint.sh (+24 行 prepare_persistent_state + v3 stage 调用)"
    - "internal/agentapi/contracts.go (+1 协议常量 ActionVolumeRemove)"
    - "internal/runtime/tasks/worker.go (+~150 行 worker 工具链 + Execute case + createHost auto-volume + WorkerRepo 接口扩展)"
    - "internal/runtime/tasks/worker_volume_test.go (+2 round-trip 测试)"
    - "internal/runtime/tasks/ssh_inject_test.go (fakeWorkerRepo 实现新方法以保证编译)"
    - "internal/store/repository/queries.go (+~50 行 upsert SQL const + 三态实现)"

key-decisions:
  - "worker createHost 中 request 通过值传递，本地 mutate request.Volumes 不影响调用方 — 与既有 buildCreateArgs(request, ...) 模式一致"
  - "audit event metadata key 严守白名单（account_id/volume_name/force/host_id），不写 OAuth/EntryPassword/credentials/oauth_token —— 通过 grep 守卫断言"
  - "Repository.UpsertClaudeAccountPersistentVolumeName 不允许 NULL→NULL 回写，三态消除（与 Phase 30 D-02 一致）"
  - "ssh_inject_test.fakeWorkerRepo 同步实现新接口方法，保证 internal/runtime/tasks 包测试编译闭环（提交合并到 Task 1.3 commit 中）"
  - "removeDockerVolume 不做 inspect fast-fail，直接 rm 让 docker 自己判断 'no such volume' / 'in use'（CONTEXT Discretion 4）"

patterns-established:
  - "Pattern: docker 子命令的可测试封装通过包级 var + real* 函数对（如 ensureDockerVolume = realEnsureDockerVolume），既能注入 mock 又能在调用点直接走 var 转发"
  - "Pattern: SQL 包级 const 命名 xxxSQL（强制约定，沿袭 Phase 29.1 — 配合 SQL token 字符串单测可锁定关键 SQL 片段不被悄悄改）"
  - "Pattern: audit event 类型按 domain.action_outcome（claude_account.volume_create_failed / volume_name_persist_failed / volume_rm_failed）"

requirements-completed: [REQ-F7-A, REQ-F7-B]

duration: 5 min
completed: 2026-04-21
---

# Phase 33 Plan 01: image-worker-agentapi Summary

**entrypoint prepare_persistent_state symlink + worker createHost 自动补 claude-state-{id} named volume + agentapi ActionVolumeRemove 协议常量 + Repository.UpsertClaudeAccountPersistentVolumeName 三态仓储方法**

## Performance

- **Duration:** 5 min
- **Started:** 2026-04-21T05:16:38Z
- **Completed:** 2026-04-21T05:21:28Z
- **Tasks:** 4
- **Files modified:** 6（含 2 个新建测试文件）

## Accomplishments

- 镜像层 OAuth/Cache 持久化锚点就位：`prepare_persistent_state` 在 v3 stage 中 `prepare_v3_dirs → prepare_mutagen_agent` 之间执行，`/home/claude/.claude` 与 `/home/claude/.cache/claude` 通过 `ln -sfn` 落到 `/var/lib/claude-persist` named volume，重启不覆盖已有数据，权限 1000:1000 双重 chown（C5+M17 防御）。
- 协议层闭环：`agentapi.ActionVolumeRemove = "volume_remove"` 协议常量 + Volumes/Labels round-trip 测试守护，供 Plan 02 admin DELETE 触发 host-agent 删 volume。
- worker 层完整闭环：`createHost` 在 `ClaudeAccountID != ""` 时自动调 `ensureDockerVolume` 幂等创建 `claude-state-<id>` + 自动追加 `VolumeMount{Target=/var/lib/claude-persist}` + `UpsertClaudeAccountPersistentVolumeName` 写库；`Execute` switch 增 `case ActionVolumeRemove → removeVolumes` + 错误码映射 `volume_in_use`。
- 仓储层三态语义：`Repository.UpsertClaudeAccountPersistentVolumeName` 区分 NULL→写入 / 一致跳过 / 冲突错误三态；SQL 提升为包级 const 并被 token-level 字符串单测锁定。
- WorkerRepo 接口扩展同步落地，`go build ./...` 闭环 PASS；既有 worker / 仓储测试无回归。

## Task Commits

| 任务 | 描述 | Commit | 类型 |
|------|------|--------|------|
| Task 1.1 | entrypoint.sh prepare_persistent_state + v3 stage 调用 | `7acf3d6` | feat |
| Task 1.2 | agentapi.ActionVolumeRemove + 2 条 round-trip 测试 | `235d969` | feat |
| Task 1.3 | worker.go BuildClaudeStateVolumeName + ensure/remove + Execute case + createHost auto-volume + WorkerRepo 接口扩展 + 7 条 lifecycle 测试 + fakeWorkerRepo 同步 | `2d0bc22` | feat |
| Task 1.4 | Repository.UpsertClaudeAccountPersistentVolumeName 三态实现 + 3 条 SQL/边界测试 | `208df5f` | feat |

_Plan metadata commit 见下方 git_commit_metadata 步骤产物。_

## Files Created/Modified

- `deploy/docker/managed-user/entrypoint.sh` — `prepare_persistent_state` 函数 + v3 stage 调用插入（`prepare_v3_dirs → prepare_persistent_state → prepare_mutagen_agent`）
- `internal/agentapi/contracts.go` — `ActionVolumeRemove HostAction = "volume_remove"` 协议常量
- `internal/runtime/tasks/worker.go` — 包级常量 `claudeStateVolumePrefix` / `claudeStateMountTarget` / `claudeAccountLabelKey` / `claudeManagedLabelKey` / `claudeManagedLabelVal`；新增 `BuildClaudeStateVolumeName` / `dockerVolumeRunner` / `ensureDockerVolume` / `removeDockerVolume` / `removeVolumes`；Execute switch 增 `case agentapi.ActionVolumeRemove` + 错误码映射 `volume_in_use`；`createHost` 在 ClaudeAccountID 非空时自动补 volume + mount + upsert；`WorkerRepo` 接口追加 `UpsertClaudeAccountPersistentVolumeName`
- `internal/runtime/tasks/worker_volume_test.go` — 追加 `TestHostActionRequest_VolumeRemove_RoundTrip` + `TestActionVolumeRemove_StringValue`
- `internal/runtime/tasks/worker_volume_lifecycle_test.go`（新建）— 7 条 lifecycle 单测覆盖 D-25.1/2/3
- `internal/runtime/tasks/ssh_inject_test.go` — `fakeWorkerRepo` 实现 `UpsertClaudeAccountPersistentVolumeName`（保证包测试编译）
- `internal/store/repository/queries.go` — 包级 `const upsertClaudeAccountPersistentVolumeNameSQL` + `checkClaudeAccountPersistentVolumeNameSQL` + `Repository.UpsertClaudeAccountPersistentVolumeName` 三态实现
- `internal/store/repository/queries_claude_account_volume_test.go`（新建）— 3 条 SQL token 字符串断言 + 空参数边界测试

## WorkerRepo 接口扩展影响面

| 实现者 | 路径 | 修改方式 |
|--------|------|----------|
| `*repository.Repository` | `internal/store/repository/queries.go` | 实现 `UpsertClaudeAccountPersistentVolumeName`（Task 1.4） |
| `fakeWorkerRepo`（测试） | `internal/runtime/tasks/ssh_inject_test.go` | 追加同名方法返回 nil（Task 1.3 一并提交，保证测试编译） |

`internal/agent/server.go::NewServer` 接收 `runtimetasks.WorkerRepo`，仍由 `*repository.Repository` 实现，调用方零破坏。

## Audit Event 类型清单 + Metadata 白名单实测样本

| 事件类型 | 出处 | Metadata Key | 白名单守恒 |
|----------|------|---------------|-------------|
| `claude_account.volume_create_failed` | `worker.createHost` ensureDockerVolume 失败分支 | `account_id`, `volume_name` | ✅ |
| `claude_account.volume_name_persist_failed` | `worker.createHost` UpsertClaudeAccountPersistentVolumeName 失败分支 | `account_id`, `volume_name` | ✅ |
| `claude_account.volume_rm_failed` | `worker.removeVolumes` removeDockerVolume 失败分支 | `volume_name`, `force` | ✅ |

`grep -nE "Metadata:.*\"(email|entry_password|credentials|oauth_token)\"" internal/runtime/tasks/worker.go` 命中 0 行（既有 `EntryPassword` 引用为 Phase 29.1 fail-fast 业务字段，不在 Metadata block 中）。

## 7 条 lifecycle 单测 PASS 时间戳（2026-04-21T05:21:28Z 全部 PASS）

```
TestBuildClaudeStateVolumeName_NonEmptyID_ReturnsPrefixedName  PASS (0.00s)
TestBuildClaudeStateVolumeName_EmptyID_ReturnsError            PASS (0.00s)
TestEnsureDockerVolume_NotExists_RunsCreate                    PASS (0.00s)
TestEnsureDockerVolume_AlreadyExists_SkipsCreate               PASS (0.00s)
TestRemoveDockerVolume_NotFound_IsSuccess                      PASS (0.00s)
TestRemoveDockerVolume_InUse_PropagatesVolumeInUseError        PASS (0.00s)
TestRemoveDockerVolume_ForceTrue_PassesDashF                   PASS (0.00s)
```

加上协议层 2 条 + 仓储层 3 条 = **新增 12 条单测全 PASS**；既有 `internal/runtime/tasks/` 与 `internal/store/repository/` 全包 `go test` 无回归。

## ClaudeAccountID Dispatcher 覆盖率 grep 实测（提供给 Plan 02 admin UAT 用）

```
$ grep -rn "ClaudeAccountID:" internal/controlplane/ internal/runtime/
internal/runtime/tasks/worker_volume_test.go:114:    ClaudeAccountID: "acct-42",
internal/runtime/tasks/worker_volume_test.go:129:    ...
```

**结论**：`internal/controlplane/` 与 `internal/runtime/` 的生产代码路径**尚无任何位置**填充 `ClaudeAccountID`（仅测试 fixture 引用），意味着当前调用 `createHost` 的 dispatcher 链路全走 D-07 fallback（不报错，不创建 volume）。本现象 = RESEARCH §6.5 已识别风险。**Plan 02 admin DELETE / Phase 30/Phase 32 入口 dispatcher 链路必须在自身 plan 中补齐 `ClaudeAccountID:` 注入**，方可让 Phase 33 自动 volume 链路真正激活；本 plan 已将工具链就位，但端到端激活责任移交 Plan 02 + 后续 phase。

## Decisions Made

- worker `createHost` 中 `request` 由值传递，本地 mutate `request.Volumes` 后立刻喂给 `buildCreateArgs(request, ...)` —— 不影响调用方，与既有 `buildCreateArgs` 调用模式自洽，无需引入指针。
- `ssh_inject_test.fakeWorkerRepo` 必须同步实现新接口方法才能让 `internal/runtime/tasks` 包测试编译；选择把这个一行修改合并到 Task 1.3 commit 中，避免引入裸"修编译"游离 commit，更贴近"接口扩展 + 实现 + 测试 fake 三位一体"的提交语义。
- `removeDockerVolume` 不做 inspect fast-fail（CONTEXT Discretion 4）；让 docker 自己判断 `no such volume` / `volume is in use`，错误字符串小写比较增强健壮性。
- `ensureDockerVolume` 不解析 inspect JSON 比对 label —— 推迟到 v3.1 backlog（见 carry-over）。

## Deviations from Plan

None - plan executed exactly as written（Task 1.3 中 fakeWorkerRepo 的同步更新属于"接口扩展闭环"必要修改，已在 PLAN `<acceptance_criteria>` 隐式覆盖：要求 `go build ./...` 退出码 = 0 + 既有 worker 测试不回归 → 必然要求 fakeWorkerRepo 同步）。

## Issues Encountered

None - 全部 4 task 按 PLAN verbatim 字段完成，Task 1.3 + Task 1.4 接口契约闭环顺序按 PLAN 警示按序提交（Task 1.3 的 commit 在 Task 1.4 commit 之前，期间 `go build ./...` 不能通过 — 这是 PLAN 显式可接受的中间态，符合 sequential 执行模式的"interface先扩展、implementation后跟进"约定）。

## User Setup Required

None - 不涉及外部服务配置。

## Next Phase Readiness

- Plan 01 工具链已就位，**Plan 02 (admin DELETE + host detail + UAT)** 可立即开始：依赖的 `ActionVolumeRemove` 协议常量 + `removeDockerVolume` 实现 + `BuildClaudeStateVolumeName` 工具 + `UpsertClaudeAccountPersistentVolumeName` 仓储方法全部 ship。
- **Carry-over (must-do for Plan 02 / 后续 phase)**：
  1. **ClaudeAccountID dispatcher 注入缺口**：现有 dispatcher 链路 (Phase 29 RuntimeService.QueueHostAction + Phase 32 attach 链路) 暂未填充 `request.ClaudeAccountID`，导致 createHost 自动补 volume 链路在生产代码路径**全程走 D-07 fallback 不激活**。Plan 02 admin handler 需要在 `Lock+DeleteClaudeAccountTx` 后通过显式 `Volumes: [{Name: BuildClaudeStateVolumeName(...)}]` 触发删除（不依赖 ClaudeAccountID）；但 v3.0 端到端 SC1 (容器重建后 OAuth 保留) 必须通过补齐 dispatcher `ClaudeAccountID:` 注入才能验证 — 由 Plan 02 UAT D-26 兜底识别。
  2. **v3.1 backlog 候选**：`ensureDockerVolume` 当前不解析 `docker volume inspect` 的 label JSON 比对一致性；若手动通过 `docker volume create claude-state-X` 但 label 不匹配，Phase 33 仍视为 already exists。建议 v3.1 增 label 一致性校验 + 输出 audit event（见 RESEARCH §6.6）。

---

## Self-Check: PASSED

- [x] 4 个 task 全部提交（4 commits: 7acf3d6, 235d969, 2d0bc22, 208df5f）
- [x] 各 task 的 acceptance_criteria 全部 PASS（grep + 单测 + go build 三类断言全部就位）
- [x] `go build ./...` 退出码 = 0（WorkerRepo 接口契约闭环）
- [x] 12 条新增单测全 PASS（2 协议 + 7 lifecycle + 3 SQL/边界）
- [x] `bash -n deploy/docker/managed-user/entrypoint.sh` 退出码 = 0
- [x] grep 断言：所有 verbatim 字段（函数名、SQL token、协议常量字符串）逐一就位
- [x] audit event metadata 白名单守恒（grep `Metadata:.*"(email|entry_password|credentials|oauth_token)"` 0 命中）
- [x] 既有 `internal/runtime/tasks/` + `internal/store/repository/` 全包测试无回归
- [x] SUMMARY.md 创建于 `.planning/phases/33-claude-code-cli-admin-gc/33-01-SUMMARY.md`

---
*Phase: 33-claude-code-cli-admin-gc*
*Completed: 2026-04-21*
