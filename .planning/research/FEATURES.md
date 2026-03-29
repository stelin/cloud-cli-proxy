# Feature Landscape

**Domain:** v1.2 用户自助面板与 Bootstrap 重设计
**Researched:** 2026-03-28
**Confidence:** MEDIUM-HIGH (基于现有代码库分析 + 行业惯例 + 官方文档)

## Table Stakes

用户期望存在的功能。缺少 = 产品感觉不完整。

| Feature | Why Expected | Complexity | Dependencies | Notes |
|---------|--------------|------------|--------------|-------|
| 用户自助面板：查看自己的主机状态 | 用户需要知道自己的机器在不在线 | Low | 用户认证体系 | 现有 admin hosts API 已有数据，需要按 user_id 过滤的新端点 |
| 用户自助面板：查看绑定的出口 IP | 出口 IP 是产品核心承诺，用户必须能看到 | Low | 用户认证体系 | 只读展示，复用现有 BindingWithIP 模型 |
| 用户登录认证（区别于管理员） | 用户需要独立入口，不能用管理员凭证 | Med | DB: users 表已有 short_id + password_hash | 复用现有 bcrypt 密码体系，新增 user JWT 签发 |
| 角色路由分离：admin vs user 视图 | 同一 SPA 必须按角色展示不同内容 | Med | 用户登录认证 | TanStack Router beforeLoad + layout route 已有模式可套用 |
| Bootstrap 重设计：`curl domain/{short_id}` 入口 | v1.2 需求明确要求的新入口路径 | Med | 现有 entry handler 已实现 `/entry/{shortId}` | 已有骨架，需增强脚本内容 |
| 实时状态推送（Bootstrap 流程） | 用户等待启动时必须看到进度，否则以为卡死 | Med | 后端 SSE 端点 | 当前 bootstrap_status 是轮询，需改为 SSE 推送 |
| Claude 账号信息展示 | v1.2 需求明确要求用户可查看 | Low | Claude 账号模型（新表） | 只读展示，管理员绑定后用户侧可见 |

## Differentiators

让产品脱颖而出的功能。不是必须，但有价值。

| Feature | Value Proposition | Complexity | Dependencies | Notes |
|---------|-------------------|------------|--------------|-------|
| Bootstrap ASCII 艺术欢迎屏 | 品牌辨识度 + 专业感，curl 下来第一眼印象 | Low | 纯 shell 脚本 | 硬编码 ASCII art 比依赖 figlet 更可靠（不需要用户装 figlet） |
| KasmVNC 远程桌面内嵌 | 用户不用离开面板就能操作桌面环境 | Med-High | 现有 VNC proxy 已实现 WebSocket 转发 | admin_vnc_proxy.go 已有反向代理骨架，需适配用户认证路径 |
| 用户自助重建主机 | 减少管理员工单，用户自治能力 | Med | 用户认证 + 现有 rebuild task 流程 | 后端 rebuild action 已存在，需加用户权限校验 |
| 一用户多主机（多 Claude 账号） | 支撑"每个 Claude 账号对应一台主机"的商业模型 | High | DB schema 变更 + 新表 | 当前 1:1 模型需要改为 1:N，影响面较大 |
| Claude 账号 CRUD（管理侧） | 管理员集中管理 Claude 账号资产 | Med | 新表 + 新 API + 新前端页面 | 标准 CRUD，复用现有管理后台模式 |
| Bootstrap 自动 SSH 接入 | 启动完成后无缝进入 SSH，不需要用户手动连 | Low-Med | SSE 完成信号 + exec ssh | 当前 entry.go 已有 exec ssh 逻辑，需与 SSE 流整合 |

## Anti-Features

明确不要做的功能。

| Anti-Feature | Why Avoid | What to Do Instead |
|--------------|-----------|-------------------|
| 用户自选代理节点 | PROJECT.md 明确排除，安全和支持风险 | 管理员统一配置，用户只能查看 |
| 用户自定义镜像 | 削弱安全性和就绪性保证 | 统一受管镜像，由管理员控制模板 |
| Web Terminal（独立于 KasmVNC） | v1 约束只做 SSH，Web Terminal 是额外复杂度 | KasmVNC 已包含终端能力，不需要单独的 xterm.js |
| 用户申请/交接账号 | PROJECT.md 明确标注"流程未设计清楚，v1.2 暂不做" | 管理员手动操作 |
| 计费/套餐/余额 | 明确 out of scope | 纯手动管理到期时间 |
| 用户侧事件日志 | 过度暴露内部运维细节，用户不关心 | 只展示主机状态和关键时间点 |
| 在 bootstrap 脚本中依赖 figlet/toilet | 不能假设用户终端装了这些工具 | 硬编码 ASCII art 字符串直接 echo |
| 独立的用户前端应用 | 增加构建/部署/维护成本 | 同一 SPA 用 TanStack Router layout route 分角色 |

---

## 详细设计参考

### 1. 角色路由：Admin vs User 同一 SPA

**现状分析：**
- 当前 SPA 使用 TanStack Router `createFileRoute("/_dashboard")` + `beforeLoad` 做管理员认证守卫
- 认证状态存 `localStorage("admin_token")`，只有 `isAuthenticated()` 布尔判断，无角色概念
- 所有 dashboard 子路由（users/hosts/egress-ips/events/tasks）都在 `_dashboard` layout 下

**推荐方案：** TanStack Router 双 layout route

```
routes/
  __root.tsx              # 不变
  login.tsx               # 管理员登录（已有）
  portal-login.tsx        # 用户登录（新增）
  _dashboard.tsx          # 管理员 layout（已有，beforeLoad 检查 admin role）
  _dashboard/
    index.tsx             # 管理员首页（已有）
    users/...             # 管理员用户管理（已有）
    hosts/...             # 管理员主机管理（已有）
    ...
  _portal.tsx             # 用户 layout（新增，beforeLoad 检查 user role）
  _portal/
    index.tsx             # 用户首页：我的主机列表
    machines.$machineId.tsx  # 单台主机详情 + VNC 入口
    account.tsx           # Claude 账号信息
```

**为什么用双 layout 而不是条件渲染：**
- 管理员和用户的侧边栏、导航、权限边界完全不同
- TanStack Router 的 `beforeLoad` 天然支持这种模式，官方文档有 RBAC 指南
- 代码边界清晰，不会出现"管理员组件意外渲染给用户"的安全问题

**认证体系变更：**
- `lib/auth.ts` 需要从纯 token 存取扩展为 `{ token, role: 'admin' | 'user' }` 结构
- 后端需要新增 `POST /v1/portal/login` 端点，签发带 `role: "user"` 的 JWT
- 管理员和用户 JWT 使用同一个 secret 但 claims 不同，中间件按 role 分发

**Confidence:** HIGH -- TanStack Router 官方文档明确描述了 RBAC layout route 模式。

### 2. KasmVNC 远程桌面内嵌

**现状分析：**
- `admin_vnc_proxy.go` 已实现完整的反向代理：HTTP 请求走 `httputil.ReverseProxy`，WebSocket 走 hijack 双向拷贝
- 路径重写逻辑已处理 `/v1/admin/hosts/{hostID}/vnc/{path...}` 前缀剥离
- 容器 IP 通过 `docker inspect` 获取，KasmVNC 监听 6080 端口
- 当前只在 admin 路由下注册，受 `AdminAuthMiddleware` 保护

**推荐方案：** 复用现有代理，新增用户侧路由

1. **用户侧 VNC 端点：** `GET /v1/portal/machines/{machineId}/vnc/{path...}`
   - 受用户 JWT 中间件保护
   - 额外校验：该 machine 必须属于当前用户（防越权）
   - 底层复用同一个 VNC proxy handler（从 AdminVNCProxyHandler 提取通用逻辑）

2. **前端嵌入：** iframe 方案
   - KasmVNC 的 Web UI 本身就是一个完整的 HTML 应用，最适合 iframe 嵌入
   - iframe src 指向 `/v1/portal/machines/{machineId}/vnc/`
   - 设置 `allow="clipboard-read; clipboard-write"` 以支持剪贴板
   - iframe 内的 WebSocket 连接会自动走同一反向代理路径

3. **认证传递：** Cookie 方案优于 URL token
   - iframe 内的请求会自动携带同域 cookie
   - 用户登录后将 JWT 同时写入 cookie（`HttpOnly; SameSite=Strict; Path=/v1/portal`）
   - 避免 token 出现在 URL 中（安全风险 + 现代浏览器限制）

**Complexity:** Med-High -- 代理骨架已有，但 iframe 内的 WebSocket 升级、cookie 认证传递、剪贴板权限需要逐一调通。

**Confidence:** MEDIUM -- VNC 反向代理已验证可工作，iframe 嵌入是 KasmVNC 官方推荐模式，但具体的认证传递细节需要实际调试。

### 3. Bootstrap ASCII 艺术欢迎屏

**现状分析：**
- 当前有两套 bootstrap 入口：
  - `/v1/bootstrap/script`：读取 `deploy/bootstrap/cloud-bootstrap.sh` 静态文件
  - `/entry/{shortId}`：动态生成 shell 脚本（entry.go），包含认证 + SSH 接入
- v1.2 要求统一为 `curl domain/{short_id}` 入口

**推荐方案：** 服务端动态生成增强脚本

脚本输出流程：
```
1. 显示 ASCII 艺术 Logo（硬编码在脚本中，不依赖 figlet）
2. 显示 "Cloud CLI Proxy" 品牌文字
3. 显示用户 short_id 和连接信息
4. read -sp 交互输入密码
5. 调用 auth API
6. SSE 流式读取启动状态，带动画进度条
7. 启动完成后 exec ssh 自动接入
```

**ASCII 艺术实现：**
- 硬编码 ASCII art 字符串，不依赖任何外部工具
- 使用 ANSI 转义码着色（`\033[36m` 等），大部分现代终端都支持
- 保持总宽度 <= 60 字符，避免窄终端换行破坏效果
- 脚本开头用 `tput colors 2>/dev/null` 检测颜色支持，无色终端回退到纯文本

**进度动画：**
- 用 `while` 循环 + `curl -sN` 读取 SSE 流
- 每收到一个 stage 事件，用 `\r` 回车覆盖当前行
- 阶段文字：`正在启动主机...` -> `正在配置网络...` -> `正在验证连通性...` -> `SSH 就绪!`

**Confidence:** HIGH -- 纯 shell 脚本，无外部依赖，完全可控。

### 4. 多账号/多主机用户模型

**现状分析：**
- 当前 `users` 表与 `hosts` 表是隐式 1:1 关系（通过 `hosts.user_id` FK）
- `GetPrimaryHostByUserID` 函数名暗示已经预留了多主机的语义空间
- 没有 Claude 账号的概念，Claude Code 只是预装在镜像里

**推荐 Schema：**

```sql
-- 新增：Claude 账号表
CREATE TABLE claude_accounts (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    label       TEXT NOT NULL,                 -- 管理员可读标签
    email       TEXT NOT NULL,                 -- Claude 账号邮箱
    status      TEXT NOT NULL DEFAULT 'active', -- active / suspended / expired
    notes       TEXT,                          -- 管理员备注
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- hosts 表增加 Claude 账号关联
-- 关系链：user 1:N hosts, host 1:1 claude_account
ALTER TABLE hosts ADD COLUMN claude_account_id UUID REFERENCES claude_accounts(id);
```

**关系模型：**
```
User (1) --> (N) Host (1) --> (1) ClaudeAccount
```

**为什么不用多对多绑定表：**
- 需求明确说"每个账号对应一台独立主机"
- 一个 Claude 账号同时只会在一台机器上使用（否则会冲突）
- 直接在 hosts 上加 `claude_account_id` 最简单，且约束清晰

**影响面：**
- `GetPrimaryHostByUserID` -> `ListHostsByUserID`（返回列表）
- 前端用户面板：从"我的主机"单卡片 -> 主机列表
- Bootstrap 入口：需要支持选择哪台主机（或默认启动第一台）
- Admin hosts CRUD：新增 claude_account_id 字段

**Confidence:** MEDIUM -- 关系模型设计基于需求分析，但"一个 Claude 账号是否真的只对应一台主机"需要与业务方确认。

### 5. 实时任务状态推送（SSE）

**现状分析：**
- 当前 bootstrap 状态查询是 REST 轮询：`GET /v1/bootstrap/tasks/{taskID}`
- 返回 `task_status` + `stage_code` + `stage_text`
- 已有 `resolveStage` 函数通过事件列表推算当前阶段
- 前端/CLI 需要自己定时 poll

**推荐方案：** 新增 SSE 端点，与现有 REST 端点共存

```
GET /v1/bootstrap/tasks/{taskID}/stream
Accept: text/event-stream

event: stage
data: {"stage_code":"host_starting","stage_text":"主机启动中","task_status":"running"}

event: stage
data: {"stage_code":"runtime_validating","stage_text":"运行时校验中","task_status":"running"}

event: stage
data: {"stage_code":"ssh_ready","stage_text":"SSH 就绪","task_status":"succeeded"}

event: done
data: {"ssh_host":"...","ssh_port":...,"ssh_user":"..."}

event: error
data: {"error_code":"...","error_message":"..."}
```

**Go 服务端实现要点：**
- 设置 `Content-Type: text/event-stream`，`Cache-Control: no-cache`，`Connection: keep-alive`
- 用 `http.Flusher` 接口每次写完立即 flush
- 每 15 秒发 `:keepalive\n\n` 心跳防止代理/CDN 断连
- 内部用 1 秒间隔轮询 DB 事件表（而非 channel），保持实现简单
- 客户端断开时 `r.Context().Done()` 自动清理

**Bash 客户端消费：**
```bash
curl -sN "$BASE_URL/v1/bootstrap/tasks/$TASK_ID/stream" | while IFS= read -r line; do
  case "$line" in
    data:*) handle_data "${line#data: }" ;;
  esac
done
```

**为什么选 SSE 不选 WebSocket：**
- 启动状态是单向推送（server -> client），不需要双向
- curl 原生支持 SSE（`curl -N`），WebSocket 需要额外工具
- Go 标准库直接支持，无需引入 gorilla/websocket
- bootstrap 脚本是 bash，SSE 比 WebSocket 容易处理一个数量级

**Confidence:** HIGH -- Go 标准库 SSE 实现成熟，curl -N 消费 SSE 是已验证的模式。

---

## Feature Dependencies

```
用户登录认证 --> 角色路由分离 --> 用户自助面板
                                    |-->  查看主机状态
                                    |-->  查看出口 IP
                                    |-->  查看 Claude 账号
                                    |-->  重建主机
                                    \-->  KasmVNC 内嵌

Claude 账号模型 --> Claude 账号 CRUD（管理侧）
                --> hosts 表 schema 变更
                --> 用户面板 Claude 账号展示

SSE 端点 --> Bootstrap 重设计（实时进度）
          --> ASCII 艺术欢迎屏（整合在同一脚本中）

多主机模型 --> Bootstrap 入口需要处理"选哪台主机"
```

## MVP Recommendation

**Phase 1（基础层 -- 其他一切的前提）：**
1. 用户登录认证体系（新端点 + user JWT）
2. 角色路由分离（双 layout route）
3. 用户面板骨架（查看主机状态 + 出口 IP，只读）

**Phase 2（数据模型扩展）：**
4. Claude 账号表 + CRUD（管理侧）
5. hosts 表增加 claude_account_id
6. 多主机支持（ListHostsByUserID）
7. 用户面板展示 Claude 账号信息

**Phase 3（Bootstrap 重设计）：**
8. SSE 状态推送端点
9. 新 bootstrap 脚本（ASCII art + 密码输入 + SSE 进度 + auto SSH）
10. `curl domain/{short_id}` 统一入口

**Phase 4（增强体验）：**
11. KasmVNC iframe 内嵌（用户面板）
12. 用户自助重建主机

**Phase ordering rationale:**
- 认证和路由是一切用户侧功能的前提，必须先做
- Claude 账号模型独立于 UI，可以在面板骨架就绪后并行推进
- Bootstrap 重设计依赖 SSE，但不依赖用户面板，可以与 Phase 2 并行
- KasmVNC 内嵌复杂度最高且不是 table stakes，放最后

**Defer:**
- 用户申请/交接账号：PROJECT.md 明确排除
- 用户侧事件日志：anti-feature，过度暴露内部细节

## Sources

- [TanStack Router RBAC Guide](https://tanstack.com/router/v1/docs/framework/react/how-to/setup-rbac) -- 角色路由官方文档
- [TanStack Router Authenticated Routes](https://tanstack.com/router/v1/docs/framework/react/guide/authenticated-routes) -- 认证路由指南
- [KasmVNC Reverse Proxy Docs](https://kasmweb.com/kasmvnc/docs/master/how_to/reverse_proxy.html) -- VNC 反向代理配置
- [KasmVNC GitHub](https://github.com/kasmtech/KasmVNC) -- 功能特性参考
- [Kasm Iframe Embedding](https://docs.kasm.com/docs/how-to) -- iframe 嵌入指南
- [Go SSE Implementation](https://oneuptime.com/blog/post/2026-01-25-server-sent-events-streaming-go/view) -- Go SSE 实战参考
- [MDN Server-Sent Events](https://developer.mozilla.org/en-US/docs/Web/API/Server-sent_events/Using_server-sent_events) -- SSE 协议规范
- [curl SSE Consumption](https://jvns.ca/blog/2021/01/12/day-36--server-sent-events-are-cool--and-a-fun-bug/) -- curl 消费 SSE 实践
- 项目现有代码：`admin_vnc_proxy.go`、`entry.go`、`bootstrap_status.go`、`_dashboard.tsx`、`auth.ts`

---
*Feature research for: v1.2 用户自助面板与 Bootstrap 重设计*
*Researched: 2026-03-28*
