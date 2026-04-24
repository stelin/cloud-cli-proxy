---
phase: 37-e2e-uat
plan: "04"
type: execute
subsystem: documentation
tags:
  - runbook
  - cold-promotion
  - pattern-g
  - operations
depends_on: []
provides: REQ-MOUNT-V31-15
affects:
  - docs/runbooks/v31-cold-promotion.md
tech-stack:
  added:
    - Markdown (Pattern G 运维手册规范)
  patterns:
    - Pattern G (头部元信息 + >=5 章节 + 快速诊断命令小节)
key-files:
  created:
    - docs/runbooks/v31-cold-promotion.md
  modified: []
decisions: []
metrics:
  duration_seconds: 189
  completed_date: "2026-04-24"
---

# Phase 37 Plan 04: 冷文件晋升机制运维手册 Summary

编写冷文件晋升机制运维手册 `docs/runbooks/v31-cold-promotion.md`，遵循 PATTERNS Pattern G 规范，覆盖原理图、启停、排障、协同边界、错误码反查五大方面，为运维人员提供完整的晋升机制操作指南。

## Tasks Executed

| Task | Name                                     | Commit  | Files Created                   |
|------|------------------------------------------|---------|---------------------------------|
| 1    | 撰写 v31-cold-promotion.md 运维手册         | fc9d3ca | docs/runbooks/v31-cold-promotion.md |

## Verification Results

| Check                              | Result |
|------------------------------------|--------|
| 文件存在                           | PASS   |
| ## 章节数 >= 5                     | PASS (6) |
| 快速诊断命令 存在                  | PASS   |
| MOUNT_PROMOTER_FAILED 覆盖         | PASS (3 处) |
| MOUNT_HOT_SYNC_FAILED 覆盖         | PASS (2 处) |
| MOUNT_SSHFS_FAILED 覆盖            | PASS (2 处) |
| MOUNT_SSHFS_DISCONNECTED 覆盖      | PASS (1 处) |
| MOUNT_MERGERFS_FAILED 覆盖         | PASS (1 处) |
| pgrep cold-promoter 命令存在       | PASS   |
| last-session.json jq 命令存在      | PASS (2 处) |
| docker exec ls 命令存在            | PASS   |
| doctor mount jq 命令存在           | PASS   |
| 头部元信息（适用版本/关联需求）      | PASS   |
| ASCII 时序图（数据流描述）          | PASS   |

## Chapter Structure

6 个 ## 章节，100% 覆盖计划要求的 5 大方面：

1. **概述与原理** — 冷文件晋升定义、ASCII 时序图 (cold sshfs → inotify → SFTP → hot → mergerfs)、关键组件表
2. **启动与关闭** — 自动启动、LIFO 自动回收、异常退出清理、CLOUD_CLAUDE_NO_PROMOTION=1 手动关闭
3. **晋升失败排障** — 5 种常见失败模式 + 诊断命令、熔断机制说明、inotify watch 限制调整
4. **与 mergerfs / hot_sync 协同** — 三层文件系统职责表、晋升文件生命周期、不与 hot_sync 轮询冲突、mergerfs 命中验证
5. **错误码反查** — 5 个 MOUNT_* 错误码 (Code + Severity + 触发 + 影响 + 修复)
6. **快速诊断命令** — 5 条可 copy-paste 命令 (pgrep / last-session.json jq / docker exec ls / doctor mount jq / explain)

## Deviations from Plan

无 - 计划已完全按原样执行。所有内容均来自 PLAN.md Task 1 action 块中指定的完整手册模板。

## Known Stubs

无 - 手册中所有内容均为实质性描述，无占位符、TODO、或未完成内容。

## Key Links Fulfilled

| Link                                              | Status |
|---------------------------------------------------|--------|
| §1 原理图 → cold_promoter.go (inotify → SFTP → hot → mergerfs) | VERIFIED |
| §5 错误码反查 → errcodes/mount.go (5 个 MOUNT_* Code + Message + NextAction) | VERIFIED |
