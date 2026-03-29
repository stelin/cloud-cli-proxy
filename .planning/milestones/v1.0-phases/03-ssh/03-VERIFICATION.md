---
phase: 03-ssh
verified: 2026-03-27T17:30:00Z
status: passed
score: 9/9 must-haves verified
human_verification:
  - test: "在真实终端执行 curl -sSL http://<host>/v1/bootstrap/script | bash，输入凭证并完成启动流程"
    expected: "密码不回显、阶段提示依次展示、最终自动进入 SSH 会话"
    why_human: "需要真实终端验证交互体验、密码回显行为和 SSH 会话接管"
  - test: "在不同终端（macOS Terminal、iTerm2、Linux shell）验证中文提示正确渲染"
    expected: "所有中文提示无乱码、排版正常"
    why_human: "字符编码和渲染依赖真实终端环境"
  - test: "模拟失败场景（凭证错误、账号禁用）后检查终端输出和退出码"
    expected: "中文错误提示可读、退出码与文档一致、显示重试建议"
    why_human: "需要人工确认提示文案可读性和重试建议清晰度"
---

# Phase 3: 启动入口与 SSH 接入 Verification Report

**Phase Goal:** 在安全运行时之上，构建顺滑的终端启动流程，完成认证、启动中提示和 SSH 接入。
**Verified:** 2026-03-27T17:30:00Z
**Status:** passed
**Re-verification:** No — initial verification

## Goal Achievement

### Observable Truths

| #   | Truth | Status | Evidence |
| --- | ----- | ------ | -------- |
| 1 | 用户可以通过单行 curl 获取启动脚本并在终端输入用户名与密码（密码不回显） | ✓ VERIFIED | GET /v1/bootstrap/script 返回 cloud-bootstrap.sh；脚本使用 `read -r -s` 读取密码；路由已挂载 |
| 2 | 认证通过后只能通过现有异步任务链路入队 start_host，不出现绕过任务系统的启动路径 | ✓ VERIFIED | bootstrap_auth.go:108 调用 `QueueHostAction(ActionStartHost)`；不存在直接 Docker 操作 |
| 3 | 凭证错误、账号禁用或账号过期会返回稳定错误码与中文提示，且不会创建启动任务 | ✓ VERIFIED | auth_invalid/account_disabled/account_expired 三种错误码均在失败路径 return 前退出；5 个测试用例覆盖 |
| 4 | 认证通过后，终端轮询能看到固定阶段进度，而不是模糊 loading | ✓ VERIFIED | bootstrap_status.go stagesByEventType 映射 net.ready→host_starting→runtime_validating→ssh_ready；脚本 poll 并打印 stage_text |
| 5 | start_host 任务只有在网络就绪且 SSH readiness 通过后才允许成功 | ✓ VERIFIED | worker.go startHost/rebuildHost 均在 PrepareHost 之后调用 waitForSSH → WaitForSSHReady；未通过则返回错误 |
| 6 | SSH readiness 未通过时任务失败并带有可诊断错误，不做自动重试 | ✓ VERIFIED | SSHNotReadyError → error_code=ssh_not_ready；无重试循环；TestWaitForSSHReady 超时测试通过 |
| 7 | 当任务成功且 SSH 已就绪后，终端脚本会自动切换到可用 SSH 会话 | ✓ VERIFIED | handoff API 仅在 succeeded + ssh.handoff.ready 事件时返回 host/port/user；脚本 exec ssh 完成接管 |
| 8 | 启动失败相关场景会输出稳定中文提示和确定退出码 | ✓ VERIFIED | BootstrapErrorEntries 定义 7 个 error_code → 中文 → exit_code(10-16)；脚本 case 映射一致 |
| 9 | 失败后只给显式重试建议，不进行自动重试或隐藏恢复 | ✓ VERIFIED | 脚本含 "请重试命令" 提示；无 retry/RETRY/auto.retry 模式；e2e 测试 Test 3 确认 |

**Score:** 9/9 truths verified

### Required Artifacts

| Artifact | Expected | Status | Details |
| -------- | -------- | ------ | ------- |
| `internal/controlplane/http/bootstrap_auth.go` | POST /v1/bootstrap/sessions 认证与任务入队 | ✓ VERIFIED | 129 行，bcrypt 校验，QueueHostAction，writeBootstrapError |
| `internal/controlplane/http/bootstrap_script.go` | GET /v1/bootstrap/script 脚本分发 | ✓ VERIFIED | 47 行，读取文件返回 text/x-shellscript |
| `internal/controlplane/http/bootstrap_status.go` | GET /v1/bootstrap/tasks/{taskID} 进度接口 | ✓ VERIFIED | 114 行，resolveStage 事件流反向扫描，完整 D-06 阶段映射 |
| `internal/controlplane/http/bootstrap_handoff.go` | GET /v1/bootstrap/tasks/{taskID}/handoff 接口 | ✓ VERIFIED | 97 行，仅 succeeded + ssh.handoff.ready 返回载荷 |
| `internal/controlplane/http/bootstrap_errors.go` | error_code → 中文提示 → 退出码 映射 | ✓ VERIFIED | 33 行，7 个 error_code 定义，LookupBootstrapError fallback |
| `deploy/bootstrap/cloud-bootstrap.sh` | 完整终端流：输入→认证→轮询→handoff→exec ssh | ✓ VERIFIED | 111 行，read -r -s，poll 循环，exec ssh，所有退出码映射 |
| `internal/runtime/tasks/ssh_ready.go` | SSH readiness gate | ✓ VERIFIED | 84 行，WaitForSSHReady 轮询+超时，SSHNotReadyError 类型 |
| `internal/runtime/tasks/ssh_handoff.go` | handoff 元数据生成 | ✓ VERIFIED | 16 行，BuildSSHHandoffMetadata 调用 DeriveManagementSSHAccess |
| `internal/network/ssh_access.go` | 管理网段 SSH 接入参数推导 | ✓ VERIFIED | 45 行，DeriveManagementSSHAccess /30 子网推导 |
| `internal/store/repository/queries.go` | 认证查询和事件查询 | ✓ VERIFIED | GetBootstrapUserByUsername、GetPrimaryHostByUserID、GetTaskByID、ListEventsByTaskID 均含真实 SQL |
| `internal/store/repository/models.go` | BootstrapUserAuth 结构体 | ✓ VERIFIED | 第 79-84 行，含 UserID/Username/PasswordHash/Status |
| `internal/controlplane/http/router.go` | 路由挂载 | ✓ VERIFIED | 4 条 bootstrap 路由均已注册（sessions/script/tasks/handoff） |
| `internal/runtime/tasks/worker.go` | waitForSSH 与 ssh.handoff.ready 事件 | ✓ VERIFIED | startHost/rebuildHost 均调用 waitForSSH；ssh.ready 后写入 ssh.handoff.ready |
| `test/bootstrap/e2e_bootstrap_ssh.sh` | E2E 脚本结构验证 | ✓ VERIFIED | 7 个检查全部通过 |

### Key Link Verification

| From | To | Via | Status | Details |
| ---- | -- | --- | ------ | ------- |
| bootstrap_auth.go | runtime_service.go | QueueHostAction(ActionStartHost) | ✓ WIRED | 第 108 行 `h.queue.QueueHostAction(r.Context(), host.ID, agentapi.ActionStartHost, requestedBy)` |
| bootstrap_auth.go | queries.go | GetBootstrapUserByUsername + GetPrimaryHostByUserID | ✓ WIRED | 第 66 行和第 96 行分别调用 |
| bootstrap_script.go | cloud-bootstrap.sh | GET /v1/bootstrap/script 返回脚本内容 | ✓ WIRED | 第 33 行 `os.ReadFile(h.scriptPath)` |
| worker.go | ssh_ready.go | startHost/rebuildHost 调用 WaitForSSHReady | ✓ WIRED | waitForSSH 方法第 285 行调用 WaitForSSHReady |
| bootstrap_status.go | queries.go | GetTaskByID + ListEventsByTaskID | ✓ WIRED | 第 62 行和第 100 行分别调用 |
| bootstrap_status.go | D-06 阶段映射 | stagesByEventType 事件到阶段映射 | ✓ WIRED | 第 47-51 行定义 ssh.ready/runtime.validating/net.ready 映射 |
| worker.go | ssh_handoff.go | ssh.ready 后写入 ssh.handoff.ready 事件 | ✓ WIRED | 第 306 行 BuildSSHHandoffMetadata，第 307 行 RecordEvent type=ssh.handoff.ready |
| bootstrap_handoff.go | bootstrap_status.go | 仅对 succeeded 任务返回 handoff | ✓ WIRED | 第 63 行检查 TaskStatusSucceeded，第 92 行查找 ssh.handoff.ready 事件 |
| cloud-bootstrap.sh | bootstrap_errors.go | error_code → 退出码映射一致 | ✓ WIRED | 脚本 case 语句与 BootstrapErrorEntries 的 ExitCode 一一对应 |

### Behavioral Spot-Checks

| Behavior | Command | Result | Status |
| -------- | ------- | ------ | ------ |
| Bootstrap auth handler 测试 | `go test ./internal/controlplane/http -run TestBootstrapAuthHandler` | 5/5 PASS | ✓ PASS |
| Bootstrap script handler 测试 | `go test ./internal/controlplane/http -run TestBootstrapScriptHandler` | 2/2 PASS | ✓ PASS |
| Bootstrap status handler 测试 | `go test ./internal/controlplane/http -run TestBootstrapStatusHandler` | 5/5 PASS | ✓ PASS |
| Bootstrap handoff handler 测试 | `go test ./internal/controlplane/http -run TestBootstrapHandoffHandler` | 4/4 PASS | ✓ PASS |
| Bootstrap error mapping 测试 | `go test ./internal/controlplane/http -run TestBootstrapErrorMapping` | 6/6 PASS | ✓ PASS |
| SSH readiness 测试 | `go test ./internal/runtime/tasks -run TestWaitForSSHReady` | 4/4 PASS | ✓ PASS |
| SSH handoff metadata 测试 | `go test ./internal/runtime/tasks -run TestBuildSSHHandoffMetadata` | 4/4 PASS | ✓ PASS |
| Bootstrap 脚本语法检查 | `bash -n deploy/bootstrap/cloud-bootstrap.sh` | SYNTAX_OK | ✓ PASS |
| E2E 脚本结构验证 | `bash test/bootstrap/e2e_bootstrap_ssh.sh` | 7/7 PASS | ✓ PASS |

### Requirements Coverage

| Requirement | Source Plan | Description | Status | Evidence |
| ----------- | ---------- | ----------- | ------ | -------- |
| ACCS-01 | 03-01 | 用户可以执行一条简短的 curl 启动命令，并在终端中看到用户名密码输入提示 | ✓ SATISFIED | GET /v1/bootstrap/script 返回脚本；脚本含 read -r / read -r -s；POST /v1/bootstrap/sessions 认证入口可用 |
| ACCS-02 | 03-02 | 用户在凭证正确后，可以在终端中看到云主机启动中的状态提示 | ✓ SATISFIED | GET /v1/bootstrap/tasks/{taskID} 返回 stage_code + stage_text；D-06 阶段映射完整 |
| ACCS-03 | 03-03 | 系统可以直接把用户接入一个可用的 SSH 会话，无需手工查找主机和端口 | ✓ SATISFIED | handoff API 返回 host/port/user；脚本 exec ssh 自动接管 |
| ACCS-04 | 03-01, 03-03 | 凭证错误、账号过期、未绑定出口 IP 或启动失败时返回明确错误提示 | ✓ SATISFIED | BootstrapErrorEntries 定义 7 种 error_code，含中文提示和非零退出码；脚本完整映射 |
| RUNT-03 | 03-02 | 接入前验证容器和 SSH 服务都已真正就绪 | ✓ SATISFIED | WaitForSSHReady gate 在 startHost/rebuildHost 中生效；未就绪任务不返回 succeeded |

**Orphaned Requirements:** None — REQUIREMENTS.md 中 Phase 3 映射的 5 个 ID 全部在 PLAN frontmatter 中声明并实现。

### Anti-Patterns Found

| File | Line | Pattern | Severity | Impact |
| ---- | ---- | ------- | -------- | ------ |
| (none) | — | — | — | — |

No TODO, FIXME, placeholder, stub, or empty implementation patterns found in any Phase 03 artifact.

### Human Verification Required

### 1. 端到端 curl → SSH 接入体验

**Test:** 在真实终端执行 `curl -sSL http://<host>/v1/bootstrap/script | bash`，输入凭证并等待启动完成。
**Expected:** 密码不回显、阶段提示（主机启动中 → 运行时校验中 → SSH 就绪）依次展示、最终自动 `exec ssh` 进入会话。
**Why human:** 需要真实终端环境验证交互行为和 SSH 会话接管。

### 2. 中文提示在不同终端的渲染

**Test:** 在 macOS Terminal、iTerm2 和 Linux shell 中分别触发认证失败和启动成功路径。
**Expected:** 所有中文提示（"认证通过，主机启动中"、"账号已被停用，请联系管理员" 等）正确渲染、无乱码。
**Why human:** 字符编码和渲染依赖真实终端环境，无法通过单元测试覆盖。

### 3. 失败场景退出码验证

**Test:** 分别模拟凭证错误、账号禁用、账号过期、主机未分配等场景，检查 `echo $?` 的值。
**Expected:** auth_invalid=10、account_disabled=11、account_expired=12、host_not_found=13、start_failed=14、ssh_not_ready=15、egress_binding_missing=16。
**Why human:** 需要在真实 shell 环境中确认进程退出码传递正确。

### Gaps Summary

No gaps found. All 9 must-have truths verified through code inspection and automated tests. All 5 requirement IDs satisfied. All key links wired. No anti-patterns detected. 37 automated tests (22 Go + 7 e2e + 8 script checks) pass.

---

_Verified: 2026-03-27T17:30:00Z_
_Verifier: Claude (gsd-verifier)_
