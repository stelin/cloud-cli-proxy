---
plan_id: "43-01"
phase: "43-vscode-portforwarding-e2e"
status: "complete"
completed_at: "2026-05-08"
duration_estimate: "15min"
---

# Plan 43-01: UAT 脚本端口转发场景补齐

## What Changed

扩展 `tests/scripts/uat-vscode-remote-ssh.sh` UAT 脚本，从 6 个场景增加到 9 个场景：

| 场景 | 名称 | 类型 | 描述 |
|------|------|------|------|
| 7 | direct-tcpip 端口转发验证 | 新增 | 通过 SSH -L 建立 direct-tcpip 隧道，curl 访问容器内 HTTP 服务验证连通性 |
| 8 | 端口转发出口 IP 验证 | 新增 | 通过 SSH 隧道访问容器内返回 client IP 的服务，验证流量路由 |
| 9 | 安全拒绝验证 | 新增 | 尝试转发到 10.99.1.1，验证 isForbiddenTarget 拦截 |

新增辅助组件：
- `--ssh-port=PORT` CLI 参数
- `detect_ssh_port()` 辅助函数
- SSH 隧道进程清理（`_UAT_SSH_PIDS` + `cleanup_ssh_tunnels` + trap EXIT）

## Key Files Modified

- `tests/scripts/uat-vscode-remote-ssh.sh` — +286 行，3 个新场景函数 + 辅助函数

## Self-Check: PASSED

- [x] bash -n 语法检查通过
- [x] --dry-run 模式输出 9 个场景
- [x] 新场景遵循现有 scenario 函数模式
- [x] exit code 语义不变（0=PASS, 1=FAIL, 2=SKIP）
- [x] SSH 隧道清理通过 trap EXIT 正确注册

## Decisions

- 端口转发场景使用 `ssh -L` 触发 direct-tcpip channel（与 VS Code Remote-SSH 真实路径一致）
- 安全拒绝测试目标使用 10.99.1.1:22（管理网段）
- SSH 隧道 PID 管理使用字符串拼接（`_UAT_SSH_PIDS`），避免 bash 数组在 `set -u` 下的 unbound variable 问题

## Commit

`feat(43-01): extend UAT script with port forwarding scenarios`
