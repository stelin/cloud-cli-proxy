---
phase: quick
plan: 260506-urq
subsystem: managed-user-image
tags: [dockerfile, claude-code, symlink, install.sh]
dependency_graph:
  requires: []
  provides: [FIX-01]
  affects: [deploy/docker/managed-user/Dockerfile]
tech_stack:
  added: []
  patterns: [cp -fL 跟随符号链接复制真实二进制]
key_files:
  created: []
  modified:
    - deploy/docker/managed-user/Dockerfile
decisions: []
metrics:
  duration: "2 minutes"
  completed_date: "2026-05-06"
---

# Phase quick Plan 260506-urq: 修复 Dockerfile 中 claude 安装产物为软链导致容器重建后断裂

**一句话总结：** 将 install.sh 安装路径的 `mv` 改为 `cp -fL`，确保 `/usr/local/bin/claude` 是独立的 ELF 文件，容器重建后不再因软链目标丢失而断裂。

## 执行摘要

| 任务 | 名称 | Commit | 文件 |
|------|------|--------|------|
| 1 | 修复 install.sh 路径的软链复制逻辑 | `28d18e7` | `deploy/docker/managed-user/Dockerfile` |
| 2 | 验证 Dockerfile 语法和逻辑完整性 | — | — |

## 变更详情

### `deploy/docker/managed-user/Dockerfile`

- **第 155 行**：`mv "${CLAUDE_BIN}" /usr/local/bin/claude` → `cp -fL "${CLAUDE_BIN}" /usr/local/bin/claude`
- `-L` 选项强制 `cp` 跟随符号链接，复制真实的目标文件内容
- `-f` 选项强制覆盖已存在的目标文件
- 保留 `chmod +x` 和 `claude --version` 验证

## 验证结果

| 检查项 | 结果 |
|--------|------|
| `grep -n "cp -fL" Dockerfile` | 命中第 155 行 |
| `grep -n "mv.*claude" Dockerfile` | 仅命中第 144 行（GitHub fallback，正确） |
| GitHub release fallback 路径 | 未改动，仍使用 `mv` |
| `claude --version` 验证 | 保留在第 158 行 |
| 反斜杠续行符 | 完整无遗漏 |

## 偏差记录

无 — 计划按预期执行，无偏差。

## 已知 Stub

无。

## Self-Check: PASSED

- [x] 修改的文件存在：`deploy/docker/managed-user/Dockerfile`
- [x] Commit 存在：`28d18e7`
- [x] 无绝对路径泄露
- [x] 无敏感信息写入
