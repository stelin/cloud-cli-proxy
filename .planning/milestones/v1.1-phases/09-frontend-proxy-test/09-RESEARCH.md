# Phase 9: 前端适配与代理测试 - Research

**Researched:** 2026-03-28
**Domain:** React 动态表单 + Go 代理测试 API
**Confidence:** HIGH

## Summary

Phase 9 覆盖两个独立但关联的领域：前端动态表单改造（UI-01 到 UI-05）和后端代理测试 API（TEST-01 到 TEST-04）。

前端方面，现有的 `EgressIPDrawer` 使用 react-hook-form + zod + shadcn/ui Sheet 实现静态 WireGuard 字段表单。需要将其改造为根据 `tunnel_type` 动态切换字段组，并在代理协议模式下根据协议类型（socks/vmess/shadowsocks/trojan/http）进一步动态渲染专用字段。同时需要实现表单模式与 JSON 编辑模式之间的双向切换。所有这些都可以用现有技术栈（react-hook-form watch + zod superRefine + 条件渲染）实现，不需要引入新依赖。

后端方面，需要新增 `POST /v1/admin/egress-ips/{ipID}/test` 端点，执行三项检测：连通性（HTTP 204）、出口 IP 匹配（ipify/ip.me）、DNS 泄漏检测。SOCKS5 协议可直接使用 `golang.org/x/net/proxy.SOCKS5()` 建立代理连接；非 SOCKS5 协议需要临时启动 sing-box 进程作为本地 SOCKS5 转发再测试。

**Primary recommendation:** 前端用 react-hook-form 的 `watch()` + 条件渲染实现动态表单，不引入额外表单库；后端用 `golang.org/x/net/proxy` 实现 SOCKS5 代理测试，非 SOCKS5 协议通过 sing-box 本地转发测试。

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions

- **D-01:** 在 Drawer 表单中使用 Select 下拉选择隧道类型（WireGuard / 代理协议），放在基础信息（标签、IP 地址、提供商）和协议配置字段之间
- **D-02:** 选择 WireGuard 时显示现有 wg_* 字段，选择代理协议时显示协议类型选择器 + 对应字段
- **D-03:** 创建模式下默认选中 WireGuard（向后兼容，与 Phase 7 D-19 一致）
- **D-04:** 编辑模式下根据已保存的 tunnel_type 回填，并相应显示对应字段组
- **D-05:** 选择"代理协议"后，先通过 Select 下拉选择协议类型（socks / vmess / shadowsocks / trojan / http），SOCKS5 默认选中
- **D-06:** 各协议字段映射（socks: server,port,username,password; vmess: server,port,uuid,security,alter_id,tls; shadowsocks: server,port,method,password; trojan: server,port,password,tls; http: server,port,username,password,tls）
- **D-07:** server 和 port 作为通用字段始终显示在协议专有字段之前
- **D-08:** 密码类字段使用 type="password" 且带可见性切换按钮
- **D-09:** 在代理协议配置区域提供"表单模式" / "JSON 模式"切换按钮
- **D-10:** JSON 模式使用 monospace textarea（不引入 Monaco/CodeMirror 依赖），提供语法错误提示
- **D-11:** 表单模式 → JSON 模式时，将当前表单值序列化为 sing-box outbound JSON 填入编辑器
- **D-12:** JSON 模式 → 表单模式时，尝试解析 JSON 回填表单字段；解析失败则提示用户
- **D-13:** 提交时统一使用 JSON 格式的 proxy_config 发送到后端
- **D-14:** 列表页每行操作菜单新增"测试"按钮
- **D-15:** 点击测试后显示 loading 状态（按钮变为 spinner），API 调用 `POST /v1/admin/egress-ips/{id}/test`
- **D-16:** 测试结果在 Dialog 中展示详情：连通性（pass/fail + 延迟）、出口 IP 匹配（预期 vs 实际）、DNS 泄漏（检测到的 DNS 服务器列表）
- **D-17:** 测试结果总状态使用颜色编码：全部通过=绿色、部分失败=黄色、全部失败=红色
- **D-18:** 后端测试 API 设置 30 秒总超时（TEST-04）
- **D-19:** 出口 IP 列表表格新增"隧道类型"列，使用 Badge 区分（WireGuard 蓝色 / 代理 紫色）
- **D-20:** 新增"测试状态"列，显示最近一次测试结果的状态圆点（绿色=通过 / 红色=失败 / 灰色=未测试）
- **D-21:** 状态圆点带 Tooltip 显示测试时间
- **D-22:** `EgressIP` 接口新增 `tunnel_type: string`、`proxy_config: Record<string, unknown> | null` 字段
- **D-23:** 新增 `useTestEgressIP` mutation hook，调用测试 API
- **D-24:** 新增 `TestResult` 接口，包含 status、tested_at、results（connectivity/egress_ip/dns_leak 子项）

### Claude's Discretion

- Zod schema 的具体组织方式（一个大 schema 带 discriminatedUnion vs 多个小 schema）
- 表单组件的内部拆分策略（单文件 vs 拆分为 WgFields / ProxyFields 子组件）
- 测试结果 Dialog 的具体布局细节
- DNS 泄漏检测的具体实现方式（后端如何判定 DNS 泄漏）

### Deferred Ideas (OUT OF SCOPE)

None — discussion stayed within phase scope
</user_constraints>

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| TEST-01 | 管理员可以对任意出口 IP 执行一键测试，验证连通性（HTTP 204 检测） | Go `golang.org/x/net/proxy` SOCKS5 dialer + http.Client 代理连接；非 SOCKS5 通过 sing-box 本地转发 |
| TEST-02 | 测试包含出口 IP 匹配检测（通过代理请求 ipify/ip.me/ifconfig.me，对比预期 IP） | HTTP GET 通过代理 dialer 请求外部 IP 检测服务，比对 EgressIP.IPAddress |
| TEST-03 | 测试包含 DNS 泄漏检测（验证 DNS 查询走代理路径） | 通过代理请求 DNS 泄漏检测 API（如 ipleak.net/json），检查返回的 DNS 服务器是否为宿主机本地 DNS |
| TEST-04 | 测试 API 设置合理超时（总超时 30s） | Go context.WithTimeout(ctx, 30*time.Second) 控制整体超时 |
| UI-01 | 出口 IP 创建/编辑表单根据 tunnel_type 切换显示 WireGuard 字段或代理协议字段 | react-hook-form watch("tunnel_type") + 条件渲染，zod superRefine 按类型分支校验 |
| UI-02 | 代理协议模式下支持按协议类型动态渲染对应字段 | watch("proxy_protocol") + 协议字段映射表驱动渲染，每个协议独立字段组件 |
| UI-03 | 提供高级 JSON 编辑模式，直接编辑 sing-box outbound 配置 | monospace textarea + JSON.parse 验证 + 双向序列化/反序列化 |
| UI-04 | 出口 IP 列表页/详情页提供测试按钮，显示 loading → 结果 | DropdownMenuItem 触发 useTestEgressIP mutation，Dialog 展示结果 |
| UI-05 | 出口 IP 列表页用状态指示器显示最近一次测试结果 | 表格新增列，彩色圆点 + Tooltip 显示测试时间 |
</phase_requirements>

## Standard Stack

### Core

| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| react-hook-form | 7.72.0 | 表单状态管理、动态字段控制 | 项目现有选型，watch() API 天然支持条件渲染 |
| zod | 4.3.6 | 表单校验 schema | 项目现有选型，superRefine 支持复杂条件校验 |
| @hookform/resolvers | 5.2.2 | zod 与 react-hook-form 桥接 | 项目现有选型 |
| @tanstack/react-query | 5.95.2 | 异步状态管理（测试 API mutation） | 项目现有选型，useMutation 处理测试请求 |
| radix-ui | 1.4.3 | UI 原语（Dialog/Select/Tooltip） | 项目现有选型 (shadcn/ui) |
| lucide-react | 1.7.0 | 图标库（Loader2/Check/X/Eye/EyeOff 等） | 项目现有选型 |
| sonner | 2.0.0 | Toast 通知 | 项目现有选型 |
| golang.org/x/net/proxy | 0.33.0 | SOCKS5 代理 dialer | go.mod 中已存在，提供 SOCKS5() 函数 |

### Supporting

| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| tailwindcss | 4.1.x | 样式 | 项目现有选型，所有 UI 样式 |
| class-variance-authority | 0.7.x | Badge 变体等组件样式 | 已在 Badge 组件中使用 |

### Alternatives Considered

| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| monospace textarea | Monaco Editor / CodeMirror | D-10 明确锁定不引入重型编辑器依赖，textarea 完全满足需求 |
| superRefine | z.discriminatedUnion | discriminatedUnion 要求所有分支共享一个 discriminator 字段，但表单有两层嵌套选择（tunnel_type → proxy_protocol），superRefine 更灵活 |

**Installation:** 不需要安装新依赖。所有需要的库已在 package.json 和 go.mod 中。

## Architecture Patterns

### 推荐前端组件结构

```
web/admin/src/
├── components/egress-ips/
│   ├── egress-ip-drawer.tsx          # 主 Drawer（改造现有文件）
│   ├── proxy-fields.tsx              # 代理协议表单字段（新建，Claude's Discretion）
│   ├── wg-fields.tsx                 # WireGuard 表单字段（抽取自 drawer，可选）
│   └── test-result-dialog.tsx        # 测试结果 Dialog（新建）
├── hooks/
│   └── use-egress-ips.ts             # 扩展 EgressIP 接口 + 新增 useTestEgressIP
└── routes/_dashboard/egress-ips/
    └── index.tsx                     # 列表页（改造现有文件）
```

### Pattern 1: 条件表单渲染（react-hook-form watch）

**What:** 用 `watch("tunnel_type")` 驱动字段组的条件渲染，切换时重置无关字段
**When to use:** tunnel_type 选择改变时，显示/隐藏对应字段组

```typescript
const tunnelType = form.watch("tunnel_type");
const proxyProtocol = form.watch("proxy_protocol");

// 切换 tunnel_type 时重置代理相关字段
useEffect(() => {
  if (tunnelType === "wireguard") {
    form.setValue("proxy_config", null);
    form.setValue("proxy_protocol", undefined);
  }
}, [tunnelType]);

// 渲染
{tunnelType === "wireguard" && <WgFields form={form} />}
{tunnelType === "proxy" && <ProxyFields form={form} protocol={proxyProtocol} />}
```

### Pattern 2: Zod 条件校验（superRefine）

**What:** 用 superRefine 实现按 tunnel_type 分支的动态校验规则
**When to use:** 提交时只校验当前活动的字段组

```typescript
const formSchema = z.object({
  label: z.string().min(1),
  ip_address: z.string().regex(/^(\d{1,3}\.){3}\d{1,3}$/),
  provider: z.string().default("manual"),
  tunnel_type: z.enum(["wireguard", "proxy"]),
  // WG fields
  wg_endpoint: z.string().optional(),
  wg_public_key: z.string().optional(),
  // ... other wg fields
  // Proxy fields
  proxy_protocol: z.enum(["socks", "vmess", "shadowsocks", "trojan", "http"]).optional(),
  proxy_server: z.string().optional(),
  proxy_port: z.coerce.number().optional(),
  // ... per-protocol fields
  proxy_config_json: z.string().optional(), // JSON mode raw text
  edit_mode: z.enum(["form", "json"]).default("form"),
}).superRefine((data, ctx) => {
  if (data.tunnel_type === "proxy" && data.edit_mode === "form") {
    if (!data.proxy_server) ctx.addIssue({ code: "custom", path: ["proxy_server"], message: "服务器地址不能为空" });
    if (!data.proxy_port) ctx.addIssue({ code: "custom", path: ["proxy_port"], message: "端口不能为空" });
    // per-protocol validation...
  }
  if (data.tunnel_type === "proxy" && data.edit_mode === "json") {
    try { JSON.parse(data.proxy_config_json ?? ""); }
    catch { ctx.addIssue({ code: "custom", path: ["proxy_config_json"], message: "JSON 格式不正确" }); }
  }
});
```

### Pattern 3: 表单值 ↔ JSON 双向转换

**What:** 表单模式和 JSON 模式之间的数据同步
**When to use:** 用户点击"表单模式"/"JSON 模式"切换按钮时

```typescript
function formValuesToProxyConfig(values: FormValues): Record<string, unknown> {
  const config: Record<string, unknown> = {
    type: values.proxy_protocol,
    server: values.proxy_server,
    server_port: values.proxy_port,
  };
  switch (values.proxy_protocol) {
    case "socks":
      if (values.proxy_username) config.username = values.proxy_username;
      if (values.proxy_password) config.password = values.proxy_password;
      config.version = "5";
      break;
    case "vmess":
      config.uuid = values.proxy_uuid;
      config.security = values.proxy_security || "auto";
      config.alter_id = values.proxy_alter_id ?? 0;
      if (values.proxy_tls) config.tls = { enabled: true, server_name: values.proxy_server_name || values.proxy_server };
      break;
    // ... other protocols
  }
  return config;
}

function proxyConfigToFormValues(config: Record<string, unknown>): Partial<FormValues> {
  const type = config.type as string;
  return {
    proxy_protocol: type,
    proxy_server: config.server as string,
    proxy_port: config.server_port as number,
    // ... per-protocol fields
  };
}
```

### Pattern 4: 测试 API Mutation + Dialog 展示

**What:** TanStack Query mutation 触发测试，结果显示在 Dialog 中
**When to use:** 用户点击列表页"测试"按钮时

```typescript
// Hook
export function useTestEgressIP() {
  return useMutation({
    mutationFn: (ipId: string) =>
      apiFetch<TestResult>(`/egress-ips/${ipId}/test`, { method: "POST" }),
  });
}

// 组件
const testMutation = useTestEgressIP();
const [testResult, setTestResult] = useState<TestResult | null>(null);

function handleTest(ipId: string) {
  testMutation.mutate(ipId, {
    onSuccess: (data) => setTestResult(data),
    onError: () => toast.error("测试失败"),
  });
}
```

### Pattern 5: Go 代理测试 Handler

**What:** 后端测试端点通过代理连接执行三项检测
**When to use:** `POST /v1/admin/egress-ips/{ipID}/test` 请求处理

```go
func (h *AdminEgressIPsHandler) Test() nethttp.Handler {
    return nethttp.HandlerFunc(func(w nethttp.ResponseWriter, r *nethttp.Request) {
        ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
        defer cancel()

        ipID := r.PathValue("ipID")
        ip, err := h.store.GetEgressIP(ctx, ipID)
        // ... error handling ...

        dialer, cleanup, err := h.getProxyDialer(ctx, ip)
        if cleanup != nil { defer cleanup() }
        // ... error handling ...

        httpClient := &nethttp.Client{
            Transport: &nethttp.Transport{DialContext: dialer.DialContext},
            Timeout:   25 * time.Second,
        }

        result := TestResult{TestedAt: time.Now().UTC()}
        result.Results.Connectivity = testConnectivity(ctx, httpClient)
        result.Results.EgressIP = testEgressIP(ctx, httpClient, ip.IPAddress)
        result.Results.DNSLeak = testDNSLeak(ctx, httpClient)
        // ... compute overall status ...

        writeJSON(w, nethttp.StatusOK, result)
    })
}
```

### Anti-Patterns to Avoid

- **不要为每个协议写独立的 Drawer 组件** — 用一个 Drawer + 条件渲染，避免大量重复代码
- **不要在 JSON 模式下跳过校验** — 即使是原始 JSON，也必须通过 JSON.parse + 结构校验后才能提交
- **不要用 useState 管理表单状态** — react-hook-form 已经管理了所有表单状态，额外 useState 会导致不一致
- **不要在测试 API 中做阻塞性等待** — 所有 HTTP 请求都必须带 context timeout，避免 goroutine 泄漏
- **不要信任前端发回的脱敏密码** — API 响应中 password 被替换为 "***"，编辑时前端需要区分"用户未修改密码"和"用户输入了新密码"

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| SOCKS5 代理连接 | 手动实现 SOCKS5 握手协议 | `golang.org/x/net/proxy.SOCKS5()` | SOCKS5 协议有认证协商、域名解析等细节，标准库已处理 |
| 表单状态管理 | 手动 useState + onChange | react-hook-form | 已有成熟的表单库在用，处理了验证、脏检查、提交等 |
| JSON 格式校验 | 手动写解析器 | `JSON.parse()` + try/catch | 浏览器内置 JSON 解析器已足够 |
| 非 SOCKS5 代理转 SOCKS5 | 自己实现 vmess/ss/trojan 客户端 | sing-box 本地进程临时转发 | 项目已依赖 sing-box，它原生支持所有协议转 SOCKS5 |
| Toast 通知 | 自定义通知系统 | sonner | 项目已有 |

## Common Pitfalls

### Pitfall 1: 密码字段脱敏冲突

**What goes wrong:** API 返回 `proxy_config.password = "***"`（脱敏），编辑时如果直接回填到表单再提交，后端会存储 "***" 作为密码
**Why it happens:** `sanitizeProxyConfig()` 在 Get/List 响应中替换密码为 "***"
**How to avoid:** 编辑模式下密码字段使用 placeholder 提示"不修改则留空"，提交时如果密码值为空或 "***" 则不包含在 proxy_config 中；或者后端在更新时检测到 "***" 则保留原值
**Warning signs:** 编辑出口 IP 后代理连接失败

### Pitfall 2: 隧道类型切换时的表单残留

**What goes wrong:** 用户先填了 WireGuard 字段，切换到代理协议后提交，WG 字段值仍在 payload 中
**Why it happens:** react-hook-form 不会自动清除隐藏字段的值
**How to avoid:** 在 `onSubmit` 中根据 `tunnel_type` 清理无关字段（WG 时设 proxy_config = null，代理时清空 wg_* 字段）；后端已有此逻辑（admin_egress_ips.go L192-197）
**Warning signs:** 创建 proxy 类型 IP 但 wg_endpoint 非空

### Pitfall 3: 测试 API 超时导致连接泄漏

**What goes wrong:** 代理不可达时，HTTP 请求挂起直到 30s 超时，但底层 TCP 连接可能不会被立即关闭
**Why it happens:** Go 的 `net/http` Transport 默认保持连接池
**How to avoid:** 每次测试创建独立的 `http.Transport`（设置 `DisableKeepAlives: true`），并在 handler 返回前显式调用 `transport.CloseIdleConnections()`
**Warning signs:** 测试后宿主机残留大量 TIME_WAIT 连接

### Pitfall 4: sing-box 临时进程僵尸

**What goes wrong:** 对非 SOCKS5 协议测试时启动的临时 sing-box 进程没有被正确清理
**Why it happens:** 进程 Start() 后如果测试中途 panic 或超时，defer cleanup 可能没机会执行
**How to avoid:** 使用 `exec.CommandContext(ctx, ...)` 确保 context 取消时进程被 kill；cleanup 函数中先 Kill() 再 Wait()
**Warning signs:** ps aux | grep sing-box 显示多个僵尸进程

### Pitfall 5: 表单模式 ↔ JSON 模式切换时数据丢失

**What goes wrong:** 用户在 JSON 模式编辑了自定义字段，切换回表单模式时这些字段丢失
**Why it happens:** 表单模式只映射已知字段（server/port/username 等），JSON 中的自定义/高级字段没有对应表单项
**How to avoid:** 切换到表单模式时，如果 JSON 包含未映射字段，弹出警告"以下字段将丢失：xxx"；或保留原始 JSON 作为 fallback
**Warning signs:** 用户切换模式后提交，发现部分配置消失

### Pitfall 6: DNS 泄漏检测误判

**What goes wrong:** DNS 泄漏检测因外部 API 不可用而误报为"通过"
**Why it happens:** 如果 DNS 泄漏检测 API（如 ipleak.net）本身不可达，跳过检测可能被视为通过
**How to avoid:** DNS 泄漏检测失败时标记为 `status: "error"`（而非 pass/fail），让管理员知道检测未能执行
**Warning signs:** 所有出口 IP 的 DNS 泄漏检测都显示通过

## Code Examples

### 已验证的现有模式

#### react-hook-form + zod 表单（现有 egress-ip-drawer.tsx 模式）

```typescript
// Source: web/admin/src/components/egress-ips/egress-ip-drawer.tsx L29-58
const formSchema = z.object({
  label: z.string().min(1, "标签不能为空"),
  ip_address: z.string().min(1).regex(/^(\d{1,3}\.){3}\d{1,3}$/),
  // ... fields
});

const form = useForm<FormValues>({
  resolver: zodResolver(formSchema),
  defaultValues: { /* ... */ },
});
```

#### TanStack Query mutation（现有 CRUD hooks 模式）

```typescript
// Source: web/admin/src/hooks/use-egress-ips.ts L35-48
export function useCreateEgressIP() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (data: Partial<EgressIP>) =>
      apiFetch<{ egress_ip: EgressIP }>("/egress-ips", {
        method: "POST",
        body: JSON.stringify(data),
      }),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["egress-ips"] });
    },
  });
}
```

#### Go handler 模式（现有 CRUD handler 模式）

```go
// Source: internal/controlplane/http/admin_egress_ips.go L154-238
func (h *AdminEgressIPsHandler) Create() nethttp.Handler {
    return nethttp.HandlerFunc(func(w nethttp.ResponseWriter, r *nethttp.Request) {
        var req createEgressIPRequest
        if err := json.NewDecoder(r.Body).Decode(&req); err != nil { /* ... */ }
        // validate, create, record event, respond
    })
}
```

#### Go SOCKS5 代理 dialer

```go
// Source: golang.org/x/net/proxy (go.mod 已包含)
import "golang.org/x/net/proxy"

auth := &proxy.Auth{User: username, Password: password}
dialer, err := proxy.SOCKS5("tcp", "socks-server:1080", auth, proxy.Direct)

// 创建 HTTP client 使用代理
transport := &http.Transport{
    DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
        return dialer.(proxy.ContextDialer).DialContext(ctx, network, addr)
    },
    DisableKeepAlives: true,
}
client := &http.Client{Transport: transport, Timeout: 10 * time.Second}
```

### 测试结果数据结构

```typescript
// 前端 TypeScript 接口（遵循 proxy-tunnel-plan.md §9）
interface TestResult {
  status: "passed" | "partial" | "failed" | "error";
  tested_at: string;
  results: {
    connectivity: {
      status: "pass" | "fail" | "error";
      latency_ms?: number;
      error?: string;
    };
    egress_ip: {
      status: "pass" | "fail" | "error";
      expected: string;
      actual?: string;
      sources?: Record<string, string>;
      error?: string;
    };
    dns_leak: {
      status: "pass" | "fail" | "error";
      dns_servers_detected?: string[];
      local_dns_leaked?: boolean;
      error?: string;
    };
  };
}
```

```go
// 后端 Go 结构
type TestResult struct {
    Status   string     `json:"status"`
    TestedAt time.Time  `json:"tested_at"`
    Results  struct {
        Connectivity ConnectivityResult `json:"connectivity"`
        EgressIP     EgressIPResult     `json:"egress_ip"`
        DNSLeak      DNSLeakResult      `json:"dns_leak"`
    } `json:"results"`
}

type ConnectivityResult struct {
    Status    string `json:"status"`
    LatencyMS int64  `json:"latency_ms,omitempty"`
    Error     string `json:"error,omitempty"`
}

type EgressIPResult struct {
    Status   string            `json:"status"`
    Expected string            `json:"expected"`
    Actual   string            `json:"actual,omitempty"`
    Sources  map[string]string `json:"sources,omitempty"`
    Error    string            `json:"error,omitempty"`
}

type DNSLeakResult struct {
    Status            string   `json:"status"`
    DNSServersDetected []string `json:"dns_servers_detected,omitempty"`
    LocalDNSLeaked    bool     `json:"local_dns_leaked,omitempty"`
    Error             string   `json:"error,omitempty"`
}
```

### sing-box 本地 SOCKS5 转发配置（非 SOCKS5 协议测试用）

```json
{
  "log": { "level": "error" },
  "inbounds": [{
    "type": "socks",
    "tag": "socks-in",
    "listen": "127.0.0.1",
    "listen_port": 0
  }],
  "outbounds": [
    { "...proxy_config..." }
  ]
}
```

## UI 组件可用性

### 已有组件（可直接使用）

| 组件 | 用途 |
|------|------|
| Sheet (SheetContent/SheetHeader) | Drawer 表单容器 |
| Select (SelectTrigger/SelectContent/SelectItem) | 隧道类型、协议类型下拉 |
| Input | 文本输入字段 |
| Label | 表单标签 |
| Button | 操作按钮 |
| Badge | 隧道类型标记（WireGuard / 代理） |
| Dialog (DialogContent/DialogHeader/DialogTitle) | 测试结果弹窗 |
| Tooltip (TooltipProvider/TooltipTrigger/TooltipContent) | 测试状态圆点提示 |
| DropdownMenu (DropdownMenuItem) | 列表页操作菜单 |
| Separator | 表单分节线 |
| Table | 列表页表格 |
| AlertDialog | 确认弹窗（已有） |

### 缺少的组件（需新增）

| 组件 | 用途 | 方案 |
|------|------|------|
| Textarea | JSON 编辑模式的多行文本框 | 用 shadcn/ui 的 Textarea 组件（或直接用原生 `<textarea>` 加 Tailwind 样式，因为只需要 monospace + 基本样式） |

**Textarea 最简实现：** D-10 锁定不引入 Monaco/CodeMirror，一个 monospace `<textarea>` 配合 Tailwind 类即可。不需要安装 shadcn/ui Textarea 组件，直接用原生 HTML textarea + className 足够。

```tsx
<textarea
  className="min-h-[200px] w-full rounded-md border border-input bg-background px-3 py-2 font-mono text-sm"
  value={jsonText}
  onChange={(e) => setJsonText(e.target.value)}
/>
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| zod 3.x z.discriminatedUnion | zod 4.x 改进的 discriminatedUnion 和 superRefine | zod 4.0 (2025) | superRefine 仍然是条件校验的最佳选择 |
| react-hook-form 旧版 watch | react-hook-form 7.x watch() + useFormContext | 7.x | watch 性能优化，支持更细粒度的订阅 |
| 手动 fetch + state | @tanstack/react-query 5.x | 5.0 (2024) | mutation 状态管理更成熟 |

## Open Questions

1. **密码字段编辑时的处理策略**
   - What we know: API Get/List 返回 `proxy_config.password = "***"`（脱敏）
   - What's unclear: 编辑时前端如何区分"保留原密码"和"清空密码"
   - Recommendation: 密码字段使用 placeholder "不修改则留空"，空值或 "***" 提交时不包含 password 字段。后端 Update handler 需要新增合并逻辑：当请求中 proxy_config.password 缺失或为 "***" 时，保留数据库中的原密码值。这属于 Agent's Discretion 范围，但计划中需要明确处理。

2. **DNS 泄漏检测的具体外部 API 选择**
   - What we know: proxy-tunnel-plan.md 提到 dnsleaktest.com/api 和 ipleak.net/json
   - What's unclear: 这些 API 的可用性和限流策略
   - Recommendation: 优先使用多个源并行检测，任一源返回即可。失败时标记为 error 而非假定通过。

3. **测试结果持久化**
   - What we know: D-20/D-21 要求列表页展示最近测试结果和时间
   - What's unclear: CONTEXT.md 未明确是否需要数据库存储测试结果
   - Recommendation: v1 可以将最近一次测试结果作为 API 响应的一部分返回（存储在 EgressIP 记录中的 JSON 列或新增列），也可以纯前端内存缓存（刷新后丢失）。建议使用简单的数据库列 `last_test_result JSONB` + `last_tested_at TIMESTAMPTZ`，这样刷新后测试结果仍然可见。

## Environment Availability

| Dependency | Required By | Available | Version | Fallback |
|------------|------------|-----------|---------|----------|
| Node.js | 前端构建 | ✓ | (项目已运行) | — |
| Go | 后端测试 API | ✓ | 1.26.1 | — |
| golang.org/x/net/proxy | SOCKS5 dialer | ✓ | 0.33.0 (go.mod indirect) | — |
| sing-box | 非 SOCKS5 协议测试转发 | ✓ (宿主机/容器内) | — | 仅支持 SOCKS5 直接测试 |
| 外部 IP 检测服务 (ipify/ip.me) | 出口 IP 匹配检测 | ✓ (公共 API) | — | 多源冗余 |

**Missing dependencies with no fallback:** 无

**Missing dependencies with fallback:** 无

## Validation Architecture

### Test Framework

| Property | Value |
|----------|-------|
| Framework | Vitest (前端) / Go test (后端) |
| Config file | 未配置 vitest.config.ts — Wave 0 任务 |
| Quick run command | `cd web/admin && npx vitest run --reporter=verbose` (前端) / `go test ./internal/controlplane/http/... -run TestEgressIP -v` (后端) |
| Full suite command | `cd web/admin && npx vitest run` (前端) / `go test ./... -v` (后端) |

### Phase Requirements → Test Map

| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| TEST-01 | 连通性检测 (HTTP 204) | unit | `go test ./internal/controlplane/http/... -run TestConnectivity -v` | ❌ Wave 0 |
| TEST-02 | 出口 IP 匹配检测 | unit | `go test ./internal/controlplane/http/... -run TestEgressIPMatch -v` | ❌ Wave 0 |
| TEST-03 | DNS 泄漏检测 | unit | `go test ./internal/controlplane/http/... -run TestDNSLeak -v` | ❌ Wave 0 |
| TEST-04 | 30 秒超时 | unit | `go test ./internal/controlplane/http/... -run TestTimeout -v` | ❌ Wave 0 |
| UI-01 | 隧道类型切换 | manual | 手动验证表单字段切换 | — |
| UI-02 | 协议类型动态渲染 | manual | 手动验证各协议字段 | — |
| UI-03 | JSON 编辑模式 | manual | 手动验证 JSON ↔ 表单切换 | — |
| UI-04 | 测试按钮和结果展示 | manual | 手动验证测试流程 | — |
| UI-05 | 状态指示器 | manual | 手动验证列表页状态圆点 | — |

### Sampling Rate

- **Per task commit:** `go test ./internal/controlplane/http/... -v -count=1`
- **Per wave merge:** `go test ./... -v`
- **Phase gate:** Full suite green before `/gsd-verify-work`

### Wave 0 Gaps

- [ ] 后端测试 API 单元测试文件（mock proxy server 验证三项检测逻辑）
- [ ] Vitest 配置（如果前端需要单元测试动态表单逻辑）

## Sources

### Primary (HIGH confidence)

- 项目源代码 — `web/admin/src/components/egress-ips/egress-ip-drawer.tsx` (现有表单模式)
- 项目源代码 — `web/admin/src/hooks/use-egress-ips.ts` (现有 hooks 模式)
- 项目源代码 — `internal/controlplane/http/admin_egress_ips.go` (现有 handler 模式)
- 项目源代码 — `internal/network/singbox_config.go` (sing-box 配置构建)
- 项目源代码 — `internal/network/types.go` (数据模型)
- 项目源代码 — `go.mod` (依赖版本确认)
- 项目源代码 — `web/admin/package.json` (前端依赖版本确认)
- `.planning/proxy-tunnel-plan.md` §7 / §9 (字段映射表和测试 API 设计)
- `golang.org/x/net/proxy` go doc 输出 (SOCKS5 API 确认)

### Secondary (MEDIUM confidence)

- npm registry 版本查询（react-hook-form 7.72.0, zod 4.3.6 等当前版本确认）

### Tertiary (LOW confidence)

- 无

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH - 全部使用项目已有依赖，版本通过 npm view 和 go.mod 确认
- Architecture: HIGH - 基于现有代码模式的自然扩展，所有模式已在项目中验证
- Pitfalls: HIGH - 基于对现有代码的直接分析（如 sanitizeProxyConfig 导致的密码脱敏问题）

**Research date:** 2026-03-28
**Valid until:** 2026-04-28 (30 days — 项目依赖稳定)
