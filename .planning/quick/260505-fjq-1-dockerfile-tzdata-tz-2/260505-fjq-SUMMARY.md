---
phase: quick
plan: 260505-fjq
subsystem: "容器镜像 / 前端管理后台"
tags: ["tzdata", "timezone", "dockerfile", "frontend"]
dependency_graph:
  requires: []
  provides: ["QUICK-TZ-01", "QUICK-TZ-02"]
  affects: ["claude-shell/docker/Dockerfile", "web/admin/src/components/hosts/create-host-dialog.tsx"]
tech-stack:
  added: []
  patterns: []
key-files:
  created: []
  modified:
    - claude-shell/docker/Dockerfile
    - web/admin/src/components/hosts/create-host-dialog.tsx
decisions: []
metrics:
  duration_seconds: 51
  completed_date: "2026-05-05T03:14:25Z"
---

# Phase quick Plan 260505-fjq: 时区修复（tzdata + 固定标准偏移）Summary

**One-liner:** 用户容器 Dockerfile 安装 tzdata 使 TZ 环境变量生效，前端时区选择列表改用固定标准偏移避免夏令时跳动。

## 任务执行记录

| # | 任务 | 类型 | 提交 | 文件 |
|---|------|------|------|------|
| 1 | Dockerfile 添加 tzdata 包 | auto | c794309 | claude-shell/docker/Dockerfile |
| 2 | 前端时区选择改为固定标准偏移 | auto | ed0f63d | web/admin/src/components/hosts/create-host-dialog.tsx |

## 变更详情

### Task 1: Dockerfile 添加 tzdata
- 在 `apt-get install -y --no-install-recommends` 列表末尾追加 `tzdata`
- 保持 `--no-install-recommends` 和 `rm -rf /var/lib/apt/lists/*` 不变
- 容器内 `TZ=America/Los_Angeles date` 等命令可正确解析时区

### Task 2: 前端固定标准偏移
- 删除 `getUTCOffset(tz: string)` 动态计算函数（原使用 `Intl.DateTimeFormat` + `timeZoneName: "shortOffset"`，夏令时导致偏移值随季节变化）
- `TIMEZONE_OPTIONS` 数组从 `{value, label}` 扩展为 `{value, label, offset}`，13 个时区全部使用标准偏移：
  - UTC-8: America/Los_Angeles
  - UTC-5: America/New_York
  - UTC-6: America/Chicago
  - UTC-7: America/Denver
  - UTC+0: Europe/London
  - UTC+1: Europe/Paris, Europe/Berlin
  - UTC+9: Asia/Tokyo, Asia/Seoul
  - UTC+8: Asia/Shanghai, Asia/Singapore
  - UTC+10: Australia/Sydney
  - UTC-10: Pacific/Honolulu
- SelectItem 渲染处将 `({getUTCOffset(tz.value)})` 替换为 `({tz.offset})`

## 验证结果

- `grep tzdata claude-shell/docker/Dockerfile` 命中且位于 apt-get install 行
- `grep getUTCOffset web/admin/src/components/hosts/create-host-dialog.tsx` 返回空（0 引用）
- `grep -E 'UTC[-+][0-9]+' web/admin/src/components/hosts/create-host-dialog.tsx` 命中 13 条固定偏移

## Deviations from Plan

无。计划执行完全按预期完成，无偏离。

## Auth Gates

无。

## Known Stubs

无。本次修改无占位符或硬编码空值。

## Self-Check: PASSED

- [x] `claude-shell/docker/Dockerfile` 已修改，tzdata 已加入
- [x] `web/admin/src/components/hosts/create-host-dialog.tsx` 已修改，getUTCOffset 已删除，固定偏移已就位
- [x] 提交 c794309 存在于 git 历史
- [x] 提交 ed0f63d 存在于 git 历史
