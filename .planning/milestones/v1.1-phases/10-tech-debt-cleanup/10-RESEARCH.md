# Phase 10: 技术债务清理 - Research

**Researched:** 2026-03-28
**Domain:** Go 后端运行时清理 + React 前端状态持久化
**Confidence:** HIGH

## Summary

本阶段修复 v1.1 里程碑审计发现的 4 项非阻塞技术债务，分为后端 2 项和前端 2 项。后端侧，`stopHost` 仅执行 `docker stop` 而不调用 `CleanupHost`，导致宿主机侧 `mgmt-*` veth 可能遗留；代理测试 API 的 vmess/ss/trojan 路径依赖 `sing-box` 二进制但控制面容器未预装。前端侧，测试结果仅存于 React 组件的 `useState<Map>` 中，页面刷新即丢失；WireGuard 类型出口 IP 的测试按钮无任何前置判断，点击后后端返回 `status: "error"` 但前端未做友好引导。

所有 4 项修复范围明确、代码路径已定位、不涉及架构变更。

**Primary recommendation:** 4 项修复均为精确的代码补丁，后端在 `worker.go` 和 `admin_egress_ip_probe.go` 各加数行，前端在列表页加 `localStorage` 持久化和 `tunnel_type` 判断。

## Standard Stack

### Core

本阶段不引入新依赖，全部使用现有技术栈。

| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| Go 标准库 `os/exec` | Go 1.26.1 | `exec.LookPath("sing-box")` 探测二进制可用性 | 标准库，无需额外依赖 |
| React `useState` + `useEffect` | React 19.2 | 组件挂载时从 localStorage 恢复状态 | 已在项目中使用 |
| `localStorage` (Web API) | — | 持久化测试结果 Map | 浏览器原生 API，项目已有使用先例（`admin_token`） |

### Alternatives Considered

| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| localStorage 持久化 | 后端新增 `last_test_result` 列 | 数据一致性更好但开发量大，不适合技术债务修复阶段 |
| `exec.LookPath` 预检 | 启动时环境检查 + 全局变量 | 更优雅但过度设计，`LookPath` 在调用时检查足够 |

## Architecture Patterns

### 修复 1: stopHost 调用 CleanupHost

**现状：**
- `rebuildHost`（worker.go:228–231）在删除容器后显式调用 `w.provider.CleanupHost(ctx, spec)`
- `stopHost`（worker.go:195–204）只执行 `docker stop`，不调用任何网络清理
- `RoutingProvider.CleanupHost` 同时调用 `TunnelProvider.CleanupHost` 和 `SingBoxProvider.CleanupHost`（防御性清理）
- 两个 provider 的 `CleanupHost` 都删除 `mgmt-{hostID[:8]}` veth，TunnelProvider 还删 `wg-*`，SingBoxProvider 还 `pkill sing-box`

**修复模式：** 在 `stopHost` 成功 `docker stop` 后追加 `CleanupHost` 调用。

```go
// internal/runtime/tasks/worker.go — stopHost
func (w *Worker) stopHost(ctx context.Context, request agentapi.HostActionRequest) error {
    containerName := firstNonEmpty(request.ContainerName, containerNameForHost(request.HostID))
    if exists, err := w.containerExists(ctx, containerName); err != nil {
        return err
    } else if !exists {
        return nil
    }

    if err := w.runDocker(ctx, "stop", containerName); err != nil {
        return err
    }

    // 清理宿主机侧网络残留（mgmt veth、wg 接口、sing-box 进程）
    if err := w.provider.CleanupHost(ctx, network.HostNetworkSpec{HostID: request.HostID}); err != nil {
        return fmt.Errorf("cleanup host network after stop: %w", err)
    }

    return nil
}
```

**关键细节：**
- `RoutingProvider.CleanupHost` 始终返回 `nil`（子实现把错误记入日志但不 propagate）
- 即使出错也不影响 stop 语义，但用 `fmt.Errorf` 包装可让上层知晓
- 容器 stop 后 PID namespace 已停，`SingBoxProvider.CleanupHost` 中的 `pkill` 会无效但不报错
- mgmt veth 是真正需要清理的资源

### 修复 2: sing-box 降级处理

**现状：**
- `getProxyDialer` 对 vmess/shadowsocks/trojan 调用 `startLocalSingBox`
- `startLocalSingBox` 通过 `exec.CommandContext(ctx, "sing-box", ...)` 启动，找不到二进制时返回 `"sing-box binary not found, cannot test %s protocol: %w"` 错误
- 该错误被 `TestProxy` 捕获，返回 `ProbeResult{Status: "error", Message: "无法建立代理连接: ..."}`
- 控制面 Dockerfile 尚未创建（compose 引用 `deploy/docker/control-plane/Dockerfile` 但文件不存在）
- 受管用户镜像（`deploy/docker/managed-user/Dockerfile`）预装 sing-box 1.13.3

**推荐降级模式：** 在 `getProxyDialer` 的 vmess/ss/trojan 分支入口处使用 `exec.LookPath("sing-box")` 预检，若找不到则返回结构化错误消息，明确告知"控制面环境未安装 sing-box，此协议测试需要 sing-box"，而非模糊的"无法建立代理连接"。

```go
case "vmess", "shadowsocks", "trojan":
    if _, lookErr := exec.LookPath("sing-box"); lookErr != nil {
        return nil, nil, fmt.Errorf("sing-box 未安装，无法测试 %s 协议（需在控制面环境安装 sing-box）", outboundType)
    }
    localPort, singboxCleanup, startErr := startLocalSingBox(ctx, proxyConfig)
    // ...
```

**补充：** 若要在容器化部署中支持 vmess/ss/trojan 测试，需在控制面 Dockerfile 中预装 sing-box（与受管镜像相同方式）。但此项属于"可选增强"，最低要求是降级处理——给出明确提示而非静默失败。

### 修复 3: 测试结果 localStorage 持久化

**现状（`web/admin/src/routes/_dashboard/egress-ips/index.tsx`）：**
- `testResults` 为 `useState<Map<string, TestResult>>(new Map())`
- 仅在 `handleTest` 的 `onSuccess` 中 `setTestResults`
- 页面刷新后 Map 清空，所有状态指示器回到灰色"未测试"

**推荐模式：** 利用 `localStorage` + JSON 序列化 `Map.entries()`。

```typescript
const STORAGE_KEY = "egress-ip-test-results";

function loadTestResults(): Map<string, TestResult> {
  try {
    const raw = localStorage.getItem(STORAGE_KEY);
    if (raw) return new Map(JSON.parse(raw));
  } catch { /* ignore */ }
  return new Map();
}

function saveTestResults(results: Map<string, TestResult>) {
  localStorage.setItem(STORAGE_KEY, JSON.stringify([...results.entries()]));
}
```

在 `EgressIPsPage` 中：
- 初始值改为 `useState<Map<string, TestResult>>(loadTestResults)`
- 在 `handleTest` 的 `onSuccess` 中，`setTestResults` 后同步写入 localStorage

**注意：** `Map` 不能直接 `JSON.stringify`，需转为 `[...map.entries()]` 数组再序列化；读取时 `new Map(JSON.parse(raw))` 即可恢复。项目中已有 `localStorage` 使用先例（`admin_token` 存取在 `lib/auth.ts`）。

### 修复 4: WireGuard 测试禁用提示

**现状：**
- 前端 `handleTest(ip)` 不检查 `ip.tunnel_type`，直接发请求
- 后端 `TestProxy` 在检测到 `tunnel_type != proxy` 时返回 `ProbeResult{Status: "error", Message: "WireGuard 类型出口 IP 在容器启动时自动验证，不支持手动测试"}`
- 但前端收到后在列表页显示灰色圆点（因 `status === "error"` 未被单独处理，与"未测试"同色），弹窗显示"测试错误"

**推荐模式 A（前端禁用 + toast 提示，推荐）：**
```typescript
function handleTest(ip: EgressIP) {
  if (ip.tunnel_type !== "proxy") {
    toast.info("WireGuard 类型出口 IP 在容器启动时自动验证，不支持手动测试");
    return;
  }
  // ... 原有逻辑
}
```
同时在 `DropdownMenuItem` 上为 WireGuard 类型添加 `disabled` 或视觉提示。

**推荐模式 B（隐藏测试菜单项）：**
```typescript
{ip.tunnel_type === "proxy" && (
  <DropdownMenuItem onClick={() => handleTest(ip)} disabled={testingId === ip.id}>
    ...
  </DropdownMenuItem>
)}
```

两种方案均可，模式 A 更明确告知用户原因，模式 B 更简洁。建议使用模式 A（禁用 + 提示），因为测试按钮的可见性让管理员知道功能存在但不适用于当前类型。

### Anti-Patterns to Avoid
- **不要在 `stopHost` 里重复 `rebuildHost` 的全部逻辑**：只需追加 `CleanupHost` 调用，不需要 `rm -f`、`PrepareHost` 等步骤
- **不要用 `sessionStorage` 替代 `localStorage`**：`sessionStorage` 关闭标签页即丢失，达不到"刷新后可恢复"的目标
- **不要为 WireGuard 测试做假结果**：不要伪造一个 `passed` 结果来掩盖不可测的事实

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| 检测 sing-box 可用性 | 自定义 PATH 搜索 | `exec.LookPath("sing-box")` | Go 标准库，处理 PATH 和可执行权限 |
| Map JSON 序列化 | 自定义序列化器 | `[...map.entries()]` + `JSON.parse` | JavaScript 内置模式 |
| veth 存在性检查 | 自定义 netlink 探测 | `netlink.LinkByName` → err → skip | 已有模式（CleanupHost 内部） |

## Common Pitfalls

### Pitfall 1: CleanupHost 在容器已停时的 sing-box pkill 行为
**What goes wrong:** `SingBoxProvider.CleanupHost` 尝试 `docker inspect` 获取 PID 然后 `nsenter` 进入容器 namespace 执行 `pkill`，但容器已停止时 PID 为 0 或 inspect 失败。
**Why it happens:** `CleanupHost` 设计为防御性调用，不知道容器当前状态。
**How to avoid:** 代码中已有 `pidStr != "0"` 的守卫检查，且错误只记 warn 日志不 propagate。`stopHost` 后调用 `CleanupHost` 是安全的——sing-box 进程已随容器 stop 而退出，mgmt veth 清理才是关键目标。
**Warning signs:** 日志中出现 `"failed to kill sing-box process"` warn 但不影响功能。

### Pitfall 2: localStorage 存储容量和数据过期
**What goes wrong:** localStorage 每个域名约 5-10MB，但测试结果数据极小（每条约 500 bytes），不会溢出。
**Why it happens:** 理论风险。
**How to avoid:** 每条 TestResult 数据量小，即使 1000 个出口 IP 也仅约 500KB。可选：仅保留最近 N 条，或在成功保存后清理已删除 IP 的陈旧数据。
**Warning signs:** `localStorage.setItem` 抛 `QuotaExceededError`。

### Pitfall 3: Map 序列化/反序列化类型丢失
**What goes wrong:** `JSON.parse` 后的 Map 值丢失 TypeScript 类型信息，可能导致 `tested_at` 不再是合法时间字符串。
**Why it happens:** JSON 序列化丢失类型元数据。
**How to avoid:** 读取时加基本的 `try-catch`；`tested_at` 本身存储为 ISO 8601 字符串，`formatDate` 会在 `new Date(dateStr)` 时处理。当前已使用字符串类型，无需额外转换。
**Warning signs:** 列表页测试时间显示 "Invalid Date"。

### Pitfall 4: exec.LookPath 在容器环境中的行为
**What goes wrong:** `exec.LookPath` 依赖容器内的 `PATH` 环境变量，若 sing-box 安装路径不在 PATH 中会误判为不可用。
**Why it happens:** Dockerfile 用 `install -m 0755` 安装到 `/usr/local/bin/sing-box`，该路径通常在 PATH 中。
**How to avoid:** 确认控制面 Dockerfile 中 sing-box 安装路径在默认 PATH 中（`/usr/local/bin` 是标准 PATH 目录）。
**Warning signs:** 日志中出现"sing-box 未安装"但实际已安装。

## Code Examples

### stopHost 添加 CleanupHost（基于现有代码）

```go
// Source: internal/runtime/tasks/worker.go:195-204 (现有) → 修改后
func (w *Worker) stopHost(ctx context.Context, request agentapi.HostActionRequest) error {
	containerName := firstNonEmpty(request.ContainerName, containerNameForHost(request.HostID))
	if exists, err := w.containerExists(ctx, containerName); err != nil {
		return err
	} else if !exists {
		return nil
	}

	if err := w.runDocker(ctx, "stop", containerName); err != nil {
		return err
	}

	if err := w.provider.CleanupHost(ctx, network.HostNetworkSpec{HostID: request.HostID}); err != nil {
		return fmt.Errorf("cleanup host network after stop: %w", err)
	}

	return nil
}
```

### sing-box LookPath 预检（基于现有代码）

```go
// Source: internal/controlplane/http/admin_egress_ip_probe.go:106-121 (现有) → 修改后
case "vmess", "shadowsocks", "trojan":
	if _, lookErr := exec.LookPath("sing-box"); lookErr != nil {
		return nil, nil, fmt.Errorf("sing-box 未安装，无法测试 %s 协议（需在控制面环境安装 sing-box）", outboundType)
	}
	localPort, singboxCleanup, startErr := startLocalSingBox(ctx, proxyConfig)
	if startErr != nil {
		return nil, nil, startErr
	}
	// ... 后续不变
```

### localStorage 持久化 Map（前端）

```typescript
// Source: web/admin/src/routes/_dashboard/egress-ips/index.tsx

const TEST_RESULTS_KEY = "egress-ip-test-results";

function loadTestResults(): Map<string, TestResult> {
  try {
    const raw = localStorage.getItem(TEST_RESULTS_KEY);
    if (raw) return new Map(JSON.parse(raw));
  } catch {
    // corrupt data — start fresh
  }
  return new Map();
}

function saveTestResults(results: Map<string, TestResult>) {
  try {
    localStorage.setItem(TEST_RESULTS_KEY, JSON.stringify([...results.entries()]));
  } catch {
    // quota exceeded — ignore
  }
}

// 在 EgressIPsPage 中：
const [testResults, setTestResults] = useState<Map<string, TestResult>>(loadTestResults);

// handleTest onSuccess:
onSuccess: (result) => {
  setTestResults((prev) => {
    const next = new Map(prev).set(ip.id, result);
    saveTestResults(next);
    return next;
  });
  setTestDialogResult(result);
  setTestingId(null);
},
```

### WireGuard 测试禁用提示（前端）

```typescript
// 在 handleTest 开头添加：
function handleTest(ip: EgressIP) {
  if (ip.tunnel_type !== "proxy") {
    toast.info("WireGuard 类型出口 IP 在容器启动时自动验证，不支持手动测试");
    return;
  }
  setTestingId(ip.id);
  testMutation.mutate(ip.id, { /* ... */ });
}

// DropdownMenuItem 添加 disabled：
<DropdownMenuItem
  onClick={() => handleTest(ip)}
  disabled={testingId === ip.id || ip.tunnel_type !== "proxy"}
>
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| 容器 stop 不清理网络 | stop 后调用 CleanupHost | Phase 10 | 消除 mgmt veth 遗留 |
| sing-box 缺失时静默失败 | `exec.LookPath` 预检 + 明确错误信息 | Phase 10 | 降低排障成本 |
| 测试结果仅在内存 | localStorage 持久化 | Phase 10 | 刷新后可恢复 |
| WireGuard 测试无前端拦截 | 前端禁用 + toast 提示 | Phase 10 | 避免无效请求和困惑 |

## Open Questions

1. **控制面 Dockerfile 是否需要在本阶段创建/补全？**
   - What we know: `docker-compose.yml` 引用 `deploy/docker/control-plane/Dockerfile` 但文件不存在。如果要在容器化部署中支持 vmess/ss/trojan 测试，需要在此 Dockerfile 中预装 sing-box。
   - What's unclear: 是否在本阶段创建 Dockerfile，还是留到后续
   - Recommendation: 本阶段只做"降级处理"（`LookPath` 预检 + 明确提示），Dockerfile 预装 sing-box 可作为可选增强项。成功标准 2 要求"sing-box 可用或降级处理"，降级处理即满足。

2. **localStorage 是否需要设置 TTL/清理策略？**
   - What we know: 测试结果数据量小（单条约 500B），即使不清理也不会溢出
   - What's unclear: 旧的（已删除的）出口 IP 的测试结果是否需要清理
   - Recommendation: 首版不需要 TTL，数据量可控。如有需要，可在后续版本加"只保留当前存在的 IP 的结果"的清理逻辑。

## Validation Architecture

### Test Framework
| Property | Value |
|----------|-------|
| Framework | Go `testing` (后端) |
| Config file | 无独立配置，使用 `go test` 默认 |
| Quick run command | `go test ./internal/...` |
| Full suite command | `go test ./...` |

### Phase Requirements → Test Map
| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| SC-1 | stopHost 后无 mgmt veth 遗留 | manual (需真实 Docker + netns) | — | ❌ manual-only |
| SC-2 | 控制面 sing-box 降级处理 | unit | `go test ./internal/controlplane/http/ -run TestProbe -x` | ❌ Wave 0 |
| SC-3 | 测试结果 localStorage 持久化 | manual (需浏览器) | — | ❌ manual-only |
| SC-4 | WireGuard 测试禁用/提示 | manual (需浏览器) | — | ❌ manual-only |

### Sampling Rate
- **Per task commit:** `go test ./internal/...`
- **Per wave merge:** `go test ./...`
- **Phase gate:** Full suite green + 4 项人工验证

### Wave 0 Gaps
- SC-2 可在 `admin_egress_ips_test.go` 中添加单元测试（mock store 返回 WireGuard 类型 IP 验证响应）
- SC-1/3/4 依赖运行时环境，仅可人工验证

## Sources

### Primary (HIGH confidence)
- 代码审查: `internal/runtime/tasks/worker.go` — stopHost/rebuildHost/CleanupHost 调用链
- 代码审查: `internal/network/routing_provider_linux.go` — RoutingProvider.CleanupHost 实现
- 代码审查: `internal/network/tunnel_provider_linux.go` — TunnelProvider.CleanupHost（mgmt veth 清理）
- 代码审查: `internal/network/singbox_provider_linux.go` — SingBoxProvider.CleanupHost（mgmt veth + pkill）
- 代码审查: `internal/controlplane/http/admin_egress_ip_probe.go` — TestProxy/getProxyDialer/startLocalSingBox
- 代码审查: `web/admin/src/routes/_dashboard/egress-ips/index.tsx` — testResults Map/handleTest/UI
- 代码审查: `web/admin/src/hooks/use-egress-ips.ts` — TestResult 类型定义
- v1.1 里程碑审计报告: `.planning/v1.1-MILESTONE-AUDIT.md`

### Secondary (MEDIUM confidence)
- Go `exec.LookPath` 文档 — 标准库 API，行为稳定
- MDN `localStorage` API — 浏览器标准 API

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH - 无新依赖，全部使用现有技术栈
- Architecture: HIGH - 4 项修复代码路径已完整定位和验证
- Pitfalls: HIGH - 基于实际代码分析，边界情况明确

**Research date:** 2026-03-28
**Valid until:** 2026-04-28（稳定，代码路径不太可能变化）
