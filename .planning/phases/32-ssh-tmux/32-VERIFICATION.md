---
phase: 32-ssh-tmux
verified: 2026-04-20T00:00:00Z
status: gaps_found
score: 6/12 must-haves verified (code-level)
overrides_applied: 0
re_verification:
  previous_status: none
  previous_score: null
  gaps_closed: []
  gaps_remaining: []
  regressions: []
gaps:
  - truth: "SC5 (REQ-F3-B) 断网时本地键入以灰色未确认样式显示，重连后按序提交"
    status: failed
    reason: "BufferedStdin 在 runClaudeWithSession 内未与 Reconnector 共享 atomic.Int32 state —— pTYAttachOnce 用独立局部 state 恒为 StateConnected，Plan 01 单测虽证明 ringBuf + 灰色 echo + Flush 自身逻辑正确，但在端到端流程中从未被触发为 Reconnecting 分支。灰色未确认样式永远不会出现；reconnect 期间用户键入直接进入 pipeW 被阻塞 / 丢弃。与 Plan 02 SUMMARY decisions[0] / Plan 03 SUMMARY Deviations[2] 自述的 carry-over 一致。"
    artifacts:
      - path: "internal/cloudclaude/session.go"
        issue: "pTYAttachOnce line 724-726：`var state atomic.Int32; state.Store(int32(StateConnected)); NewBufferedStdin(os.Stdin, &state, ...)` —— 该 state 变量从未被 Reconnector 写入，永远为 Connected。Plan 02 计划提到的 `Reconnector.RegisterStateListener` / `BufferedStdin.SetReconnector` 接口补强始终未实现（grep 确认两个方法均不存在）。"
    missing:
      - "runClaudePTYWithReconnect 构造 Reconnector 时应先创建共享 atomic.Int32（或直接使用 reconnector.StateAddr() — 该 getter 已由 Plan 01 暴露），并在第一次 pTYAttachOnce 之前把 BufferedStdin 的 state 指针指向同一个 atomic；reconnect 成功的 onReconnected 回调中调 bs.Flush() 把 ringBuf 写回新 session.Stdin。"
      - "或采用 Plan 02 PLAN <interfaces> 中 Q4-RESOLVED 方案：在 reconnect.go 补 Reconnector.RegisterStateListener(listener func(ConnState))、input_buffer.go 补 BufferedStdin.SetReconnector(r *Reconnector)，在 pTYAttachOnce 中 bs.SetReconnector(reconnector) 一行挂接。"
      - "补充集成级单测（fake conn.Wait 返回 io.EOF 触发 reconnect → 在断网期用 io.Pipe 喂 'abc' 到 BufferedStdin → assert echo 含 ansiGray + ringBuf 非空 → reconnect 成功后 pipeR 读到 'abc'）。"

  - truth: "SC11 (REQ-F5-D) 同一 claude_account 第二个 cloud-claude 启动时 Mutagen sync 不重复创建（账号级单例锁）"
    status: failed
    reason: "AcquireSyncLock 实现和 ssh.go 注入闭包已就位，sync_lock 自身单测 + 集成测试 TestIntegration_Phase32_SyncLockMutexes 能证明锁语义正确；但 mount_strategy.MountWorkspace 函数体内从未调用 mountCfg.SyncSessionLock(accountID) —— Phase 31 D-31 仅在 MountConfig 上预留了字段，落地调用缺失。Plan 03 SUMMARY decisions[3] 自述为 carry-over。结果：真实多端启动场景下 flock 永远不会被触发，两端 cloud-claude 仍会并行 mutagen create / sync，M15 双写风险未被实际防御。"
    artifacts:
      - path: "internal/cloudclaude/mount_strategy.go"
        issue: "rg 确认 MountWorkspace / mountMutagen 代码路径无任何 `SyncSessionLock(` 调用；字段仅用于文档和结构体定义。"
      - path: "internal/cloudclaude/ssh.go"
        issue: "闭包 mountCfg.SyncSessionLock = func(...) { AcquireSyncLock(connA, ...) } 被注入，但下游调用方从不 invoke — 属典型 orphan wiring。"
    missing:
      - "mount_strategy.MountWorkspace 在 mountMutagen 之前调用 release, lockErr := mountCfg.SyncSessionLock(mountCfg.ClaudeAccountID)；errors.Is(lockErr, ErrSyncLocked) → 强制 ActualMode=ModeSSHFSOnly + 在 DowngradeChain 追加 DowngradeStep{Reason: \"sync_locked\", From: intended, To: \"sshfs_only\"} + 写 last-session.json；成功分支注册 defer release() 直到 cloud-claude 退出。"
      - "相应单测或集成测试：mount_strategy_test.go 增加 fake SyncSessionLock 返回 ErrSyncLocked 的 case，断言 ActualMode 与 DowngradeChain。"
      - "可选：ssh.go 闭包已置 mountCfg.IsSecondaryClient = true，Phase 31 mount_strategy 调用后该字段会正确透传给 SessionConfig；因此仅需补 mount_strategy 侧的 invoke，不需要改 Plan 02 / Plan 03 其余链路。"

deferred: []

human_verification:
  - test: "30s 网络抖动 UAT（SC2 / REQ-F4-A / BASE-03）"
    expected: "cloud-claude 进入 tmux 后运行 claude → `docker network disconnect <ctr> bridge` 持续 30s → `docker network connect <ctr> bridge` → cloud-claude 自动 reconnect 成功；tmux capture-pane 对比前后，claude 进程 PID 不变、scrollback buffer 完整、无丢字。"
    why_human: "TestIntegration_Phase32_NetworkDisconnect30s 已落框架但 t.Skip（docker network 权限 + 端到端 PTY 交互工程量超本 plan）；RESEARCH §Test Matrix 明确留 Phase 35 真机 UAT。"
  - test: "pkill -SIGHUP sshd 后 tmux 存活 UAT（SC3 / C7）"
    expected: "`ssh container 'tmux new -d -s test; sleep 1; pkill -SIGHUP sshd'` 执行后，本端重新 ssh 进容器 `tmux attach -t test` 必须成功；session 窗口内容保留。"
    why_human: "integration_test.go TestIntegration_Phase32_TmuxSurvivesSighupSshd / TestIntegration_Phase32_PgrepNoSystemdLogind 需要实际 docker fixture（scripts/test-fixture-up.sh 需本地 docker）；CI 环境不保证权限，TestMain 会 os.Exit(0) 优雅 skip。手测链：run fixture → go test -tags=integration -run TestIntegration_Phase32_TmuxSurvives -v。"
  - test: "多端 banner 第二行 UAT（SC9 / REQ-F5-A / REQ-F5-B）"
    expected: "同一 shortID 容器内启两个 cloud-claude 进程：第二端 banner 必须含 `✓ 已 attach 到会话 claude-<id>`，紧跟第二行 `（另 1 个会话正在共享：<第一端 hostname> / 刚刚活跃）`；hostname 来自 /workspace/.cloud-claude/clients/<pid>.json 文件注册表的 hostname 字段，活跃时间来自 tmux list-clients client_activity。"
    why_human: "printAttachBanner + readClientHostnames 代码路径在 session.go 存在且单测覆盖纯函数层（parseTmuxListClients / formatBannerSecondLine / decideTakeOverClientCount），但 writeClientFile 是在容器内远程执行 printf '%s' > .json 的异步 goroutine；真端到端验收需要两个活 ssh session + 两个 tmux client + 注册表文件落盘时序 —— 属 docker UAT 范畴。"
  - test: "账号级单例锁 docker UAT（SC11 / REQ-F5-D 端到端）"
    expected: "**前提：Gap #2（mount_strategy 未调 SyncSessionLock）闭合后**，容器内同一 claude_account 启两个 cloud-claude：第一个 `docker exec` 容器 `ls /tmp/cloud-claude/locks/` 应见 `sync-<account>.lock`；`ps aux | grep 'sleep infinity'` 应只见 1 个；第二个 cloud-claude 启动时 stderr 必含 `[SESSION_SYNC_LOCKED]` + last-session.json `client_role=\"secondary\"` + mergerfs runtime branch 无 full 档位（仅 sshfs）。"
    why_human: "TestIntegration_Phase32_SyncLockMutexes 已能证明 AcquireSyncLock 自身的锁语义，但端到端多端 cloud-claude 启动（含 mount_strategy 降级 + last-session 写入 + sshfs-only 视图）需要完整 docker 环境 + 先闭合 Gap #2。"
  - test: "sshfs 抖动 30s 不挂死 UAT（SC12 / C3）"
    expected: "启动 cloud-claude 进 full 档位 → 容器内 `docker network disconnect` 30s → ls /workspace 不 hang；sshfs_watcher 在 15s 内把 cold branch 从 mergerfs 摘除（MOUNT_SSHFS_DISCONNECTED 警告可见）；网络恢复后 mergerfs 自动接回或手动重挂成功。"
    why_human: "sshfs_watcher 在 Phase 31 ship，Phase 32 只做联合验收；需要完整挂载链（sshfs + mergerfs + mutagen）+ docker network 权限。"
---

# Phase 32: SSH-Tmux Verification Report

**Phase Goal:** 把 SSH 会话从"断线重来"升级为"30s 抖动无感知 + 进程不丢 + 多端共享 attach"；交付 F3（SSH 弱网容忍 + 自动重连）+ F4（tmux 默认包装 + 会话恢复）+ F5（多端同账号 attach），并在容器侧账号粒度强制 Mutagen 单例锁；必须防御 C3（sshfs 抖动级联）+ C7（systemd-logind 杀 tmux）。
**Verified:** 2026-04-20
**Status:** `gaps_found`
**Re-verification:** No — initial verification

## Goal Achievement

### Observable Truths — 12 Success Criteria

| #   | Truth (SC)                                                | Status          | Evidence                                                                                                                                                                                                                                  |
| --- | --------------------------------------------------------- | --------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| 1   | SC1  — `ServerAliveInterval < 15s` 启动失败 + 明确错误       | ✓ VERIFIED      | `cmd/cloud-claude/main.go:363-366` 启动期校验 `KeepAliveInterval > 0 && < 15s` → `errcodes.Format(SESSION_KEEPALIVE_TOO_AGGRESSIVE, ...)` + `os.Exit(exitConfigError=4)`；`keepalive.go::RunKeepAlive` 入口再次 defensive check；errcodes session.go:9-14 已注册 Fatal Severity。 |
| 2   | SC2  — 断网 30s 内 cloud-claude 不退出 / claude 不丢 / tmux 可 attach      | ? UNCERTAIN     | 代码路径 = tmux new-session -A 包装 + RunKeepAlive + Reconnector 骨架完整；但端到端"进程不丢 + buffer 完整"需要 docker network disconnect UAT。见 human_verification#1。                                                             |
| 3   | SC3  — `pkill -SIGHUP sshd` 后 `tmux attach -t test` 成功        | ? UNCERTAIN     | `TestIntegration_Phase32_TmuxSurvivesSighupSshd` + `TestIntegration_Phase32_PgrepNoSystemdLogind` 代码就位；需实际 docker fixture 跑 `-tags=integration`。见 human_verification#2。                                           |
| 4   | SC4  — 退避序列 `[1s,2s,4s,8s,30s]` + 不重新弹密码                    | ✓ VERIFIED      | `reconnect.go:30-37` `var backoffSeq = []time.Duration{1s,2s,4s,8s,30s}`；`TestBackoffSeq` 单测逐元素断言；Reconnector.Run 循环调 `sshConnect(r.cfg)` 复用启动期已缓存 `SSHConfig.Password`；grep 确认无 askpass / stdin 重读。      |
| 5   | SC5  — 断网灰色未确认 echo + 重连后按序提交 / 无丢字 / 无乱序        | ✗ FAILED        | **carry-over gap #1 确认**：`session.go::pTYAttachOnce:724-726` 使用独立局部 `atomic.Int32`（恒为 StateConnected），Plan 01 的 BufferedStdin Reconnecting 分支永不激活；Plan 02 SUMMARY decisions[0] / Plan 03 SUMMARY Deviations[2] 均已自述。 |
| 6   | SC6  — 重连失败 prompt 两要素（原因 + 下一步）                         | ✓ VERIFIED      | `errcodes/net.go:33-46` NET_RECONNECT_GAVE_UP 注册 `Message="重连失败（已重试 %d 次，耗时 %s）"` + `NextAction="请检查网络后重新运行 cloud-claude，或运行 cloud-claude doctor 诊断"`；`reconnect.go::FormatGiveUpMessage` 调用 `errcodes.Format` 自动 prefix [code] + 建议段。    |
| 7   | SC7  — 容器内 tmux 不可用：不阻塞启动 + banner `[!] 容器内 tmux 不可用...`   | ✓ VERIFIED      | `ssh.go::ConnectAndRunClaudeV3` 在 OAuth 检查后 `available, _, reason := DetectTmux(connA)`；false 分支 `errcodes.Format(SESSION_TMUX_UNAVAILABLE, reason)` + `return runClaude(...)` 走 v2.0 裸路径，不 `os.Exit`；errcodes session.go Severity=Warn。    |
| 8   | SC8  — `cloud-claude sessions ls` / `attach <name>` 列表 + 切换        | ✓ VERIFIED      | `cmd/cloud-claude/sessions.go::newSessionsCmd` 注册 `ls` / `attach`；`session.go::RunSessionsLs` 远程 `tmux list-sessions -F ...` + tabwriter 表格；`RunSessionsAttach` 远程 `tmux has-session` 校验 → 失败 [SESSION_NOT_FOUND] + `ExitConfigError(=4)`。 |
| 9   | SC9  — 多端默认共享 attach + 第二端 banner `（另 N 个会话正在共享）`          | ✓ VERIFIED (code) / ? HUMAN (UAT) | `printAttachBanner` + `readClientHostnames` + `writeClientFile`（/workspace/.cloud-claude/clients/<pid>.json）全链路已实装；`tmux new-session -A` 确保默认不踢人（rg 确认模板字面值）。端到端 "另 N 个会话共享" 文字输出需 docker 双端 UAT。见 human_verification#3。 |
| 10  | SC10 — `--new-session` 生成 `claude-<short_id>` / `--take-over` 通知其它端 | ✓ VERIFIED      | `main.go:245-275` 剥离 `--new-session`/`--take-over` + `GenerateShortSessionID()` 调用；`session.go::performTakeOver` 实现 list-clients/display-message/sleep 3/detach-client -a + `errcodes.Format(SESSION_TAKEOVER_NOTIFIED, ...)`。            |
| 11  | SC11 — 账号级 Mutagen 单例锁（后连端只读 sshfs / mergerfs 视图）         | ✗ FAILED        | **carry-over gap #2 确认**：`sync_lock.go::AcquireSyncLock` + ssh.go 闭包注入已就位，但 `mount_strategy.go::MountWorkspace` 函数体内 **从未调用** `mountCfg.SyncSessionLock(accountID)`（grep 验证）；锁实际未生效。Plan 03 SUMMARY decisions[3] 自述 carry-over。       |
| 12  | SC12 — sshfs 抖动 30s 后 mergerfs 不整体挂死（与 Phase 31 联合）          | ? UNCERTAIN     | sshfs_watcher 在 Phase 31 ship 不由本 phase 重写；`TestIntegration_Phase32_NetworkDisconnect30s` 仅落框架并 t.Skip；需 docker + 真拔网 UAT。见 human_verification#5。                                                                     |

**Score:** 6 / 12 must-haves VERIFIED code-level (SC1, SC4, SC6, SC7, SC8, SC10)
**FAILED:** 2 (SC5, SC11 — carry-over gaps)
**UNCERTAIN / 需人工：** 4 (SC2, SC3, SC9 端到端部分, SC12)

### Deferred Items

无。所有未通过项要么属 code-level 真实 gap（SC5 / SC11），要么属 UAT 性质（由 human_verification 兜底），均不匹配"later milestone phase 吸收"判据。

### Required Artifacts (per-plan must_haves artifacts)

| Artifact                                                    | Plan    | Status           | Details                                                                                                                      |
| ----------------------------------------------------------- | ------- | ---------------- | ---------------------------------------------------------------------------------------------------------------------------- |
| `internal/cloudclaude/keepalive.go` + 三平台 build-tag       | 01      | ✓ VERIFIED       | RunKeepAlive + sendKeepaliveWithTimeout + ConfigureTCPKeepAlive；`//go:build linux` / `darwin` / `!linux && !darwin` 三文件就位。 |
| `internal/cloudclaude/reconnect.go`                         | 01      | ✓ VERIFIED       | backoffSeq / Reconnector / renderDisconnectStatus / FormatGiveUpMessage / StateAddr 全部实装。                              |
| `internal/cloudclaude/input_buffer.go`                      | 01      | ⚠️ ORPHANED      | 代码本身 + 单测齐备；但 runClaudeWithSession 路径未共享 state（local atomic） → 实际生产路径退化为 stdin 直传（SC5 ✗）。       |
| `internal/cloudclaude/errcodes/{codes,session,net}.go`      | 01      | ✓ VERIFIED       | 10 条新码（7 SESSION_* + 3 NET_*）全部注册，codes_test.go PASS。                                                            |
| `internal/cloudclaude/last_session.go` TmuxSession/ClientRole/ReconnectCount | 01      | ✓ VERIFIED       | 3 字段 omitempty；`TestLastSessionSnapshot_NewFieldsRoundTrip` + `_OmitemptyForEmpty` PASS。                              |
| `internal/cloudclaude/session.go` (867 LOC)                 | 02      | ✓ VERIFIED       | DetectTmux / buildTmuxSessionName / runClaudeWithSession / pTYAttachOnce / writeClientFile / performTakeOver / RunSessionsLs 等全部实装；22 条 session_test.go PASS。 |
| `cmd/cloud-claude/sessions.go`                              | 02      | ✓ VERIFIED       | newSessionsCmd + ls / attach + connectForSessions；`go run ... sessions --help` 能看到 ls / attach 节点。                   |
| `internal/cloudclaude/sync_lock.go`                         | 03      | ⚠️ ORPHANED      | AcquireSyncLock + ErrSyncLocked + parseLastInt 全部实装；但 mount_strategy 路径永不调用 → 生产场景锁不生效（SC11 ✗）。      |
| `internal/cloudclaude/integration_test.go` 6 个 TestIntegration_Phase32_* | 03      | ✓ VERIFIED (build) / ? HUMAN (run) | 6 个用例函数名就位，`go build -tags=integration` PASS；执行需 docker fixture。                                             |

### Key Link Verification

| From                                  | To                                    | Via                                                              | Status          | Details                                                                                                                                           |
| ------------------------------------- | ------------------------------------- | ---------------------------------------------------------------- | --------------- | ------------------------------------------------------------------------------------------------------------------------------------------------- |
| `ssh.go::sshConnect`                  | `keepalive.go::ConfigureTCPKeepAlive` | 拨号成功后立即调用，best-effort                                    | ✓ WIRED         | grep: `ConfigureTCPKeepAlive(tc, 15*time.Second)` 在 sshConnect 内调用；错误仅 stderr warning 不 return。                                           |
| `reconnect.go::Reconnector`           | `ssh.go::sshConnect`                  | 复用 SSHConfig 重新拨号                                            | ✓ WIRED         | Reconnector.Run 内 `sshConnect(r.cfg)` — r.cfg 含启动期缓存 Password。                                                                              |
| `input_buffer.go::BufferedStdin`      | `reconnect.go::Reconnector`           | 共享 atomic.Int32 state 指针 / SetReconnector listener             | ✗ NOT_WIRED     | **SC5 的根因**：pTYAttachOnce 使用独立局部 state；RegisterStateListener / SetReconnector 接口 Plan 02 显式 defer，从未实装。                         |
| `ssh.go::ConnectAndRunClaudeV3`       | `session.go::DetectTmux`              | OAuth 检查后调用，结果决定走 runClaudeWithSession 或 runClaude      | ✓ WIRED         | ssh.go OAuth 后 `DetectTmux(connA)` 调用可见；true → runClaudeWithSession，false → fallback runClaude + SESSION_TMUX_UNAVAILABLE。                  |
| `cmd/cloud-claude/sessions.go`        | `internal/cloudclaude/RunSessionsLs/Attach` | cobra RunE → 业务 helper                                           | ✓ WIRED         | `runSessionsLs` / `runSessionsAttach` 分别委托 `cloudclaude.RunSessionsLs` / `RunSessionsAttach`。                                                   |
| `ssh.go::ConnectAndRunClaudeV3`       | `sync_lock.go::AcquireSyncLock`       | MountWorkspace 调用之前把闭包注入 mountCfg.SyncSessionLock         | ✓ WIRED (注入)   | ssh.go:95-109 闭包赋值到 `mountCfg.SyncSessionLock`；但下一跳断裂。                                                                                 |
| `mount_strategy.go::MountWorkspace`   | `mountCfg.SyncSessionLock` 闭包        | mountMutagen 之前调用拿锁，失败 errors.Is(err, ErrSyncLocked) 降级 | ✗ NOT_WIRED     | **SC11 的根因**：MountWorkspace 函数体内 grep 无 `SyncSessionLock(` 调用 — Phase 31 D-31 预留字段未落地 invoke。                                     |
| `session.go::writeLastSessionTmuxField` | `last_session.go::ClientRole`       | role 由 SessionConfig.IsSecondaryClient 决定 primary/secondary     | ✓ WIRED         | session.go 内 `role := "primary"; if sessionCfg.IsSecondaryClient { role = "secondary" }`；IsSecondaryClient 来自 mountCfg。Gap 仅在 mountCfg.IsSecondaryClient 永远不被置位（因 SyncSessionLock 没被调），但 wiring 本身正确。 |

### Data-Flow Trace (Level 4)

| Artifact / 数据目标                             | Data Variable         | Source                                                  | Produces Real Data  | Status          |
| ----------------------------------------------- | --------------------- | ------------------------------------------------------- | ------------------- | --------------- |
| BufferedStdin.ringBuf → SSH session.Stdin (SC5)  | ringBuf bytes         | pTYAttachOnce 内局部 atomic.Int32 恒为 Connected         | 否（永远走直传分支） | ✗ DISCONNECTED  |
| mount_strategy 降级路径 → last-session client_role=secondary (SC11) | IsSecondaryClient     | mountCfg.SyncSessionLock 闭包在 ErrSyncLocked 时置位     | 否（闭包永不被调用） | ✗ DISCONNECTED  |
| banner 第二行 hostname (SC9)                     | readClientHostnames map | /workspace/.cloud-claude/clients/<pid>.json 文件注册表 | 是（writeClientFile 写入 + readClientHostnames 读） | ✓ FLOWING (代码路径) — 端到端需 docker UAT 验证实际值落盘 |
| Reconnector renderDisconnectStatus (SC4 UX)      | disconnectStart       | Reconnector.Run 入口 `r.disconnectStart.Store(time.Now().UnixNano())` | 是                   | ✓ FLOWING      |
| keepalive 拦截 io.EOF → reconnect 路径 (SC2 底层) | conn.Close() 发起     | RunKeepAlive 连续 countMax 次超时 → conn.Close()        | 是                   | ✓ FLOWING      |

### Behavioral Spot-Checks

| Behavior                                                              | Command                                                                                   | Result                     | Status   |
| --------------------------------------------------------------------- | ----------------------------------------------------------------------------------------- | -------------------------- | -------- |
| 单元测试全套 PASS（Plan 01/02/03 单测 + Phase 31 既有）                    | `go test ./internal/cloudclaude/... -count=1 -short`                                      | ok internal/cloudclaude 0.974s / errcodes 0.311s | ✓ PASS |
| `cloud-claude sessions` 子命令注册成功                                  | `go run ./cmd/cloud-claude sessions --help 2>&1 \| grep -E "ls\|attach"`                  | (Plan 02 SUMMARY 实测 PASS) | ✓ PASS |
| backoffSeq 序列正确                                                    | `TestBackoffSeq` (reconnect_test.go)                                                      | PASS                       | ✓ PASS |
| errcodes 唯一性 + NextAction ≤ 80 runes + 命名正则                       | `go test ./internal/cloudclaude/errcodes/... -count=1`                                    | PASS                       | ✓ PASS |
| 30s 抖动 / pkill sshd / 多端 banner / docker 锁 UAT                     | `go test -tags=integration -run TestIntegration_Phase32_NetworkDisconnect30s ...`         | t.Skip（无 docker / short） | ? SKIP → human_verification |

### Requirements Coverage

| Requirement                 | Source Plan | Description (from REQUIREMENTS.md)                                                         | Status (code-level) | Evidence                                                                                      |
| --------------------------- | ----------- | ------------------------------------------------------------------------------------------ | ------------------- | --------------------------------------------------------------------------------------------- |
| REQ-F3-A                    | 01          | SSH KeepAlive 间隔 < 15s 启动失败                                                            | ✓ SATISFIED         | SC1 evidence                                                                                  |
| REQ-F3-B                    | 01          | 断网期间本地键入灰色未确认样式，重连后按序提交                                                  | ✗ BLOCKED           | SC5 — BufferedStdin 未与 Reconnector 共享 state                                               |
| REQ-F3-C                    | 01          | 重连失败 prompt 必须显示原因 + 下一步操作                                                      | ✓ SATISFIED         | SC6 — NET_RECONNECT_GAVE_UP Message + NextAction 已注册 + FormatGiveUpMessage 调用           |
| REQ-F3-D                    | 01          | 退避 1s→2s→4s→8s→30s + 不弹密码                                                              | ✓ SATISFIED         | SC4                                                                                           |
| REQ-F4-A                    | 02          | 30s 抖动内 tmux / claude 进程不丢                                                            | ? NEEDS HUMAN       | SC2 UAT                                                                                       |
| REQ-F4-B                    | 02          | `cloud-claude sessions ls / attach <name>`                                                 | ✓ SATISFIED         | SC8                                                                                           |
| REQ-F4-C                    | 02          | 容器内 tmux 不可用场景降级不阻塞启动                                                          | ✓ SATISFIED         | SC7                                                                                           |
| REQ-F5-A                    | 02          | 多端默认共享 attach（不踢人）                                                                | ✓ SATISFIED (code)  | SC9 — `tmux new-session -A` + printAttachBanner                                               |
| REQ-F5-B                    | 02          | 第二端 banner 显示其它会话 hostname + 活跃时间                                                | ✓ SATISFIED (code) / ? HUMAN (UAT) | SC9 — writeClientFile/readClientHostnames 已实装；端到端文字渲染需 docker UAT                |
| REQ-F5-C                    | 02          | `--new-session` / `--take-over`                                                            | ✓ SATISFIED         | SC10 — flag 剥离 + GenerateShortSessionID + performTakeOver                                  |
| REQ-F5-D                    | 03          | 账号级 Mutagen 单例锁                                                                        | ✗ BLOCKED           | SC11 — mount_strategy 未调 SyncSessionLock 闭包                                               |

无 orphaned requirement：Phase 32 REQUIREMENTS.md 绑定的 11 条 REQ-F3/F4/F5-* 全部出现在三个 plan 的 `requirements:` frontmatter 中。

### Anti-Patterns Found

本次 verifier 未独立扫描代码（Phase 32 已有 32-REVIEW.md 覆盖 5 Warning + 7 Info）。REVIEW.md findings 分类：

| File                                             | Line        | Pattern                                      | Severity | Impact                                                                                                           |
| ------------------------------------------------ | ----------- | -------------------------------------------- | -------- | ---------------------------------------------------------------------------------------------------------------- |
| internal/cloudclaude/session.go                  | 703-712     | SIGWINCH goroutine 泄漏（channel 不 close）     | ⚠️ Warning | 每次 reconnect +1 泄漏；不直接 block 任何 SC，但长时间重连会累积。归 REVIEW.md WR-01。                              |
| internal/cloudclaude/session.go                  | 614-764     | `*registryPid` data race（主 goroutine 与异步 writeClientFile）   | ⚠️ Warning | 极端 race 时可能漏 removeClientFile → 孤儿 registry entry。归 REVIEW.md WR-02；SC9 banner 在 race 场景显示 unknown-host。 |
| internal/cloudclaude/session.go + input_buffer.go | 多处         | 多个 BufferedStdin.Run goroutine 并发读 os.Stdin | ⚠️ Warning | reconnect 后旧 goroutine 不退（Read 阻塞 syscall） → 新一轮会与旧 goroutine 抢字节。归 REVIEW.md WR-03；与 SC5 gap 一起需重构。|
| internal/cloudclaude/input_buffer.go             | 90-141      | grayOpen / localEcho 无 mutex 跨 goroutine    | ⚠️ Warning | 当前 Flush 未被调用（因 SC5 gap），暂未爆；修 SC5 时必须一并加锁。归 REVIEW.md WR-04。                             |
| internal/cloudclaude/session.go                  | 200-209     | `\|\| exec <wrapCmd字面值>` fallback 对 cd builtin 失败 | ⚠️ Warning | fallback 路径不可用；主路径 `command -v tmux` 通过时无感。归 REVIEW.md WR-05。                                      |
| internal/cloudclaude/sync_lock.go                | 63          | lockPath 未校验 accountID 路径穿越             | ℹ️ Info   | accountID 来自 gateway（受信）；深度防御建议。REVIEW.md IN-01。                                                   |
| internal/cloudclaude/last_session.go             | 67-74       | WriteLastSession 固定 .tmp 后缀               | ℹ️ Info   | 目前串行写无 race；未来并发写需加锁。IN-02。                                                                    |
| internal/cloudclaude/keepalive.go                | 51-67       | `time.After` 不可取消                         | ℹ️ Info   | 15s 周期，实际 goroutine 泄漏可忽略；IN-03。                                                                    |
| internal/cloudclaude/session.go                  | 671-673     | `pTYAttachOnce` `(int, error, error)` 两 error 易混淆 | ℹ️ Info   | 可读性；IN-04。                                                                                                |
| internal/cloudclaude/session.go                  | 881-883     | RunSessionsAttach 用 ExitConfigError=4 表示 session not found | ℹ️ Info   | 语义不精确；IN-05。                                                                                            |
| cmd/cloud-claude/sessions.go                     | 58-75       | os.Exit 前 defer 不执行                       | ℹ️ Info   | IN-06。                                                                                                        |
| internal/cloudclaude/errcodes/net.go             | 31-36       | NET_RECONNECT_BACKOFF 注册但未实际 Format 输出  | ℹ️ Info   | IN-07；用于 SC6 上游日志可见性不损伤。                                                                          |

### Human Verification Required

见 YAML frontmatter `human_verification:` —— 5 项：

1. **30s 网络抖动 UAT（SC2 / BASE-03）** — docker network disconnect/connect + tmux capture-pane 前后对比；Phase 35 真机兜底。
2. **pkill -SIGHUP sshd 存活 UAT（SC3 / C7）** — docker fixture 跑 `TestIntegration_Phase32_TmuxSurvivesSighupSshd` + `TestIntegration_Phase32_PgrepNoSystemdLogind`。
3. **多端 banner 第二行 UAT（SC9 / REQ-F5-B）** — 同一 shortID 两端 cloud-claude，第二端 stderr 必须含 `（另 1 个会话正在共享：<hostname> / 刚刚活跃）`。
4. **账号级单例锁端到端 UAT（SC11 / REQ-F5-D）** — **先闭合 Gap #2** — 后两端启动 + ls /tmp lockfile + ps sleep infinity + stderr SESSION_SYNC_LOCKED + last-session secondary。
5. **sshfs 抖动 30s 不挂死 UAT（SC12 / C3）** — 完整挂载链 + docker network 权限 + mergerfs runtime branch 观察。

### Gaps Summary

**两个真实 gap 阻挡 phase goal 闭环，均属 Plan 02/03 已自述的 carry-over，非新发现：**

1. **Gap #1 — SC5 (REQ-F3-B) 本地键入灰色未确认 echo 从未在端到端路径激活**
   - Plan 01 的 BufferedStdin 代码 + 单测齐备；Plan 02 为避免改 Plan 01 接口（RegisterStateListener / SetReconnector / StateAddr 共享）而在 pTYAttachOnce 用独立局部 atomic.Int32 恒为 StateConnected，导致 ringBuf + ansiGray echo 分支从未被触发。
   - **闭合建议**：最小改动是改 `session.go::pTYAttachOnce` 用 `reconnector.StateAddr()`（Plan 01 已暴露该 getter）直接共享；或按 Plan 02 PLAN 原始 Q4-RESOLVED 方案补 RegisterStateListener + SetReconnector。同时需要补 WR-03（多 goroutine 读 os.Stdin 竞争）修复：让 BufferedStdin 跨多次 attach 单例复用。

2. **Gap #2 — SC11 (REQ-F5-D) mount_strategy.MountWorkspace 未调用 SyncSessionLock 闭包**
   - sync_lock.go 实现 + ssh.go 注入闭包 + 集成测试 TestIntegration_Phase32_SyncLockMutexes 均齐备；但 Phase 31 D-31 在 MountConfig 预留 SyncSessionLock 字段后，MountWorkspace 函数体从未真正调用，Plan 03 遵守"不重写 mount_strategy" 指令未修复。后果：生产多端启动场景锁永远不触发，M15 双写风险未实际防御。
   - **闭合建议**：在 mount_strategy.go::MountWorkspace 内（mountMutagen 尝试之前）调用：
     ```go
     release, lockErr := mountCfg.SyncSessionLock(mountCfg.ClaudeAccountID)
     if errors.Is(lockErr, cloudclaude.ErrSyncLocked) {
         // 强制 ModeSSHFSOnly + DowngradeChain 追加 sync_locked
     } else if lockErr != nil {
         // 其它错误传递
     } else {
         defer release()
     }
     ```
     并在现有 mount_strategy_test.go 补 fake SyncSessionLock 返回 ErrSyncLocked 的 case。

**其余可观察 SC（SC2/SC3/SC9-UAT/SC12）需要 docker UAT**，不能在代码扫描层 close；建议与 Gap #2 一并在 docker fixture 上跑 `TestIntegration_Phase32_*` 套件闭环。

**建议执行**：`/gsd-plan-phase 32 --gaps` 基于 frontmatter gaps 生成两个补洞 plan（或合并为一个 plan），优先闭合 Gap #2（小改动、端到端影响大）再 Gap #1（需要跨 Plan 01/02 接口协同、影响 SC5 单条）。

---

_Verified: 2026-04-20T00:00:00Z_
_Verifier: Claude (gsd-verifier)_
