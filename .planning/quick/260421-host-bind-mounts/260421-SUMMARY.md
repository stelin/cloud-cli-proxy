---
phase: 260421-host-bind-mounts
plan: 01
subsystem: backend + frontend
tags: [host-mounts, bind-mount, admin-api]
dependency_graph:
  requires: []
  provides: [host_mounts_jsonb, bind_mount_api, mount_manager_ui]
  affects: [repository, agentapi, runtime, controlplane, admin-web]
tech_stack:
  added: []
  patterns: [jsonb-column, raw-message-scan, bind-mount-docker-args]
key_files:
  created:
    - internal/store/migrations/0015_host_mounts.sql
    - web/admin/src/components/hosts/mount-manager.tsx
  modified:
    - internal/store/repository/models.go
    - internal/store/repository/queries.go
    - internal/agentapi/contracts.go
    - internal/runtime/runtime_service.go
    - internal/runtime/tasks/worker.go
    - internal/controlplane/http/admin_hosts.go
    - internal/controlplane/http/router.go
    - web/admin/src/hooks/use-hosts.ts
    - web/admin/src/routes/_dashboard/hosts/$hostId.tsx
    - web/admin/src/components/hosts/create-host-dialog.tsx
decisions:
  - "使用 json.RawMessage 中间变量扫描 JSONB 列，再 Unmarshal 到 HostMounts 类型"
  - "Checkbox 组件不存在，改用原生 HTML input[type=checkbox]"
  - "保留路径警告仅前端提示不阻断，管理员知情即可"
metrics:
  duration: "12m"
  completed: "2026-04-29"
  tasks_completed: 2
  tasks_total: 2
---

# 宿主机路径挂载管理 260421-01 Summary

宿主机路径挂载全链路实现：DB JSONB 列 + 仓储查询 + Agent API 契约 + Runtime 映射 + Worker Docker --mount type=bind 参数 + Admin API PUT 端点 + 前端挂载管理组件与详情页/创建对话框接入。

## 完成的任务

| Task | 名称 | Commit | 关键文件 |
|------|------|--------|----------|
| 1 | 后端全链路 -- 迁移、模型、仓储、契约、Runtime、Worker、Admin API | 2e910f3 | 0015_host_mounts.sql, models.go, queries.go, contracts.go, runtime_service.go, worker.go, admin_hosts.go, router.go |
| 2 | 前端接入 -- 类型 Hooks、挂载管理组件、详情页、创建对话框 | ee98afa | use-hosts.ts, mount-manager.tsx, $hostId.tsx, create-host-dialog.tsx |

## 实现细节

### DB Migration
- `0015_host_mounts.sql`: `ALTER TABLE hosts ADD COLUMN IF NOT EXISTS host_mounts JSONB NOT NULL DEFAULT '[]'`
- 空数组 `[]` 为默认值，现有主机无需迁移

### Repository
- `HostMount` 结构体 (Source, Target, ReadOnly) + `HostMounts` 类型 (`[]HostMount`)
- `Host` 和 `UpsertHostParams` 均追加 `HostMounts` 字段
- 所有 Host 读查询 (getHostSQL, listHostsSQL, listHostsByUserIDSQL, listRunningHostsSQL, listRunningHostsByUserIDSQL, listHostsWithUsernameSQL, getHostWithClaudeAccountSQL, GetPrimaryHostByUserID) 追加 `host_mounts` 列扫描
- 使用 `json.RawMessage` 中间变量扫描 JSONB，再 Unmarshal 到 `HostMounts`
- 新增 `UpdateHostMounts(ctx, hostID, HostMounts)` 方法

### Agent API
- 新增 `BindMount` 结构体 (Source, Target, ReadOnly)
- `HostActionRequest` 追加 `BindMounts []BindMount`

### Runtime Service
- `QueueHostAction` 中将 `host.HostMounts` 逐字段映射到 `request.BindMounts`

### Worker
- `buildCreateArgs` 在 Volumes 循环后新增 BindMounts 循环
- 校验 source/target 非空且为绝对路径
- `os.MkdirAll(source)` 自动创建宿主机目录
- 生成 `--mount type=bind,src={source},dst={target}[,readonly]`

### Admin API
- `AdminHostStore` 接口追加 `UpdateHostMounts` 方法
- Create handler 请求体追加 `host_mounts` 字段
- 新增 `UpdateMounts` handler: PUT `/v1/admin/hosts/{hostID}/mounts`
  - 校验 source/target 非空且为绝对路径
  - 审计事件 `admin.host.update_mounts`

### 前端
- `HostMount` 接口 + `useUpdateHostMounts` hook
- `useCreateHost` payload 追加 `host_mounts` 可选字段
- 新建 `MountManager` 组件: 列表展示/添加/删除/保存，运行中禁用，保留路径警告
- 详情页出口 IP 绑定 Card 后新增挂载路径 Card
- 创建对话框新增可选挂载路径区域

## 偏差记录

**Rule 1 - Bug**: Checkbox 组件不存在
- **发现于:** Task 2
- **问题:** `@/components/ui/checkbox` 未安装
- **修复:** 改用原生 HTML `input[type=checkbox]` + Tailwind 样式
- **文件:** web/admin/src/components/hosts/mount-manager.tsx
- **Commit:** ee98afa

## 已知 Stubs

无 -- 所有功能已完整接入。

## Self-Check: PASSED

- 0015_host_mounts.sql 存在且含 host_mounts JSONB 列
- models.go 含 HostMount 结构体 + HostMounts 类型 + Host.HostMounts + UpsertHostParams.HostMounts
- queries.go 含 UpdateHostMounts 方法，所有 Host 读查询 SELECT host_mounts
- contracts.go 含 BindMount 结构体 + HostActionRequest.BindMounts
- runtime_service.go QueueHostAction 中 host.HostMounts 映射到 request.BindMounts
- worker.go buildCreateArgs 中 BindMounts 循环生成 --mount type=bind 参数
- admin_hosts.go 含 UpdateMounts handler + AdminHostStore 接口新增方法
- router.go 注册 PUT /v1/admin/hosts/{hostID}/mounts
- use-hosts.ts 含 HostMount 接口 + useUpdateHostMounts hook
- mount-manager.tsx 存在，导出 MountManager 组件
- 详情页含挂载路径 Card
- 创建对话框含挂载路径区域
- go build ./... 通过
- npm run build 通过
- Commit 2e910f3 存在
- Commit ee98afa 存在
