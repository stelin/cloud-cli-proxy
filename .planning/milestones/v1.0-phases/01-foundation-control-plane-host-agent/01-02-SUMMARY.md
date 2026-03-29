---
phase: 01-foundation-control-plane-host-agent
plan: "02"
subsystem: infra
tags: [docker, openssh, ubuntu, claude-code, image-lock]
requires:
  - phase: 01-01
    provides: control-plane development baseline and repository conventions
provides:
  - pinned managed-user image contract for Phase 1 lifecycle tasks
  - SSH-capable workspace container template with `claude code`
  - build and smoke-verification scripts for the managed image
affects: [host-agent, lifecycle-api, phase-3-ssh]
tech-stack:
  added: [Ubuntu 24.04, OpenSSH Server, Node.js, npm]
  patterns: [locked-image-metadata, workspace-home-persistence, shell-based-smoke-checks]
key-files:
  created:
    - deploy/docker/managed-user/Dockerfile
    - deploy/docker/managed-user/image.lock
    - deploy/docker/managed-user/build-managed-image.sh
    - scripts/verify-managed-image.sh
  modified:
    - deploy/docker/managed-user/sshd_config
    - deploy/docker/managed-user/entrypoint.sh
    - deploy/docker/managed-user/README.md
key-decisions:
  - "受管镜像固定使用 `/workspace` 作为默认用户主目录与持久化挂载点。"
  - "控制面与 host-agent 统一从 `image.lock` 读取镜像全名，而不是各自硬编码。"
  - "Phase 1 的镜像验证只覆盖 SSH 基础能力与内容完整性，不提前实现网络强约束。"
patterns-established:
  - "Pattern: 受管镜像通过 lock 文件暴露运行时契约。"
  - "Pattern: 镜像冒烟校验脚本只验证内容与入口，不直接做真实 SSH handoff。"
requirements-completed: [RUNT-01, RUNT-02]
duration: 1 min
completed: 2026-03-26
---

# Phase 01 Plan 02: 受管镜像模板 Summary

**固定镜像锁、SSH 工作环境和 `claude code` 预装的受管用户模板容器**

## Performance

- **Duration:** 1 min
- **Started:** 2026-03-26T16:50:46+08:00
- **Completed:** 2026-03-26T16:51:07+08:00
- **Tasks:** 3
- **Files modified:** 7

## Accomplishments
- 创建了基于 `ubuntu:24.04` 的受管用户镜像模板，内置 OpenSSH、常用 shell 工具和 `claude code`。
- 用 `image.lock` 明确锁定镜像全名、默认用户、主目录挂载点和默认重建模式。
- 提供了固定标签的构建脚本与镜像内容校验脚本，为 host-agent 后续拉起容器提供可重复基础。

## Task Commits

Each task was committed atomically:

1. **Task 1: 创建受管用户镜像与 SSH 启动脚本** - `f4bc265` (feat)
2. **Task 2: 锁定镜像版本并写清运行时契约** - `59e2a93` (docs)
3. **Task 3: 提供构建与冒烟校验脚本** - `3ac6d91` (chore)

**Plan metadata:** pending

## Files Created/Modified
- `deploy/docker/managed-user/Dockerfile` - 受管镜像定义与软件安装清单
- `deploy/docker/managed-user/sshd_config` - 固定 `Port 22` 与 `AuthorizedKeysFile` 契约
- `deploy/docker/managed-user/entrypoint.sh` - SSHD 启动入口与 host key 初始化
- `deploy/docker/managed-user/image.lock` - 镜像 pin、挂载点和重建模式
- `deploy/docker/managed-user/README.md` - 控制面与 host-agent 共享的镜像使用说明
- `deploy/docker/managed-user/build-managed-image.sh` - 固定 tag 的镜像构建脚本
- `scripts/verify-managed-image.sh` - `sshd`、`claude`、`workspace` 用户和 `/workspace` 校验脚本

## Decisions Made
- 镜像只做 Phase 1 需要的 SSH 与开发工具基础，不提前插入 WireGuard 或隧道逻辑。
- 通过 shell 脚本维持镜像验收闭环，降低未来 CI 接入成本。
- 将 `factory_reset_mode` 只定义为契约字段，避免在本阶段误实现 destructive 行为。

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered
- 本机 Docker daemon 不可连接，因此镜像构建和 `docker run` 级别验证尚未实际执行；已完成脚本语法检查与计划要求的内容级 `rg` 校验。

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness
- `image.lock` 已成为控制面与 host-agent 共享的单一镜像来源，Phase 1 生命周期任务可以直接消费。
- Phase 2 只需在此模板旁边扩展网络准备逻辑，无需重做 SSH 或工作区基础镜像。

---
*Phase: 01-foundation-control-plane-host-agent*
*Completed: 2026-03-26*
