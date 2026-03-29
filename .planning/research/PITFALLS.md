# Domain Pitfalls

**Domain:** v1.2 用户自助面板与 Bootstrap 重设计 — 为现有 Go+React+Docker 平台添加用户认证、角色视图、KasmVNC 代理、Bootstrap URL 重设计、多账号数据模型
**Researched:** 2026-03-28
**Confidence:** HIGH（基于项目代码审查 + 官方文档 + 社区经验）

---

## Critical Pitfalls

导致需要重写或产生严重安全/可用性问题的错误。

### Pitfall 1: 用户 JWT 与管理员 JWT 共享签名密钥导致越权

**What goes wrong:** 当前管理员 JWT 使用单一 `JWTSecret` 签名，Subject 固定为 `"admin"`。如果用户 JWT 也复用同一密钥，且中间件仅检查 token 有效性而不检查 role claim，那么持有用户 token 的人可以直接访问 `/v1/admin/*` 端点。

**Why it happens:** 现有 `AdminAuthMiddleware` 只做了 HMAC 签名验证和过期检查，没有检查 `Subject` 或任何 role claim。这是 v1.0 合理的简化——因为只有管理员有 token。但新增用户认证后，如果不同时修改中间件，两种 token 将相互通用。

**Consequences:** 任何普通用户都能执行管理员操作：删除用户、解绑出口 IP、停止他人主机、查看所有事件日志。

**Prevention:**
1. 用户和管理员使用**不同的签名密钥**，或使用不同的 `Issuer`/`Audience` claim，且在各自中间件中严格校验
2. `AdminAuthMiddleware` 必须增加 `Subject == "admin"` 或 `role == "admin"` 检查
3. 新建 `UserAuthMiddleware`，只接受 `role == "user"` 的 token
4. 添加测试：用用户 token 请求 admin 端点，断言返回 403

**Detection:** 在 CI 中加入越权测试；代码审查时检查所有 auth 中间件是否有 role 断言。

**Phase:** 必须在用户认证体系阶段**第一个任务**就解决。

---

### Pitfall 2: 前端角色路由仅做 UI 隐藏，不做后端鉴权

**What goes wrong:** React SPA 中通过 `role` 判断显示管理员导航还是用户面板，但后端 API 没有对应的 per-user 数据隔离。用户通过 DevTools 或直接 curl 调用管理员 API 即可越权。

**Why it happens:** 前端路由保护给人一种"已经做了权限控制"的错觉。现有 `apiFetch` 硬编码 `API_BASE = "/v1/admin"`，如果用户端也用同一个 fetch 工具调用用户端点（如 `/v1/user/...`），很容易在重构时把 admin 端点暴露给用户请求路径。

**Consequences:** 前端权限形同虚设，任何有 token 的用户都能操作非己资源。

**Prevention:**
1. 后端是唯一的权限边界：每个 handler 必须从 JWT 中提取 `user_id`，查询时用 `WHERE user_id = $1` 过滤
2. 用户端点（`/v1/user/*`）和管理员端点（`/v1/admin/*`）使用不同的中间件栈
3. 前端角色路由只是 UX 优化，不是安全措施
4. 测试矩阵：用户 A 的 token 不能查到用户 B 的主机/账号

**Detection:** API 测试用两个不同用户的 token 交叉请求对方资源。

**Phase:** 用户自助面板 API 设计阶段。

---

### Pitfall 3: Bootstrap URL `/{short_id}` 与现有路由冲突

**What goes wrong:** 将 bootstrap 入口从 `/v1/bootstrap/script` 改为 `/{short_id}` 后，`short_id` 可能与现有路径冲突：`/healthz`、`/v1`、`/login`、`/entry/{shortId}`、前端 SPA 的 HTML5 History 路径（`/hosts`、`/users` 等）。

**Why it happens:** Go `net/http.ServeMux` 的路由匹配规则是最长前缀匹配。`/{short_id}` 是一个 catch-all 通配符，如果注册不当会吞掉本应匹配其他 handler 的请求。此外，SPA 需要一个 catch-all 来返回 `index.html`，两个 catch-all 会冲突。

**Consequences:** 用户访问 `curl domain/abc123` 可能触发错误的 handler；或者管理后台 SPA 路由全部 404；或者 healthz 探针失败导致监控误报。

**Prevention:**
1. `short_id` 格式限制为纯字母数字且固定长度（如 6-8 位），在路由注册时用正则或前置检查区分
2. 具体路由（`/healthz`、`/v1/*`、`/login`）总是优先于 `/{short_id}` 注册
3. 在 `/{short_id}` handler 内部，先查库确认 short_id 存在，不存在则 fallback 到 SPA 或 404
4. 建立一个 **保留词表**（healthz、v1、admin、login、api、static 等），创建用户时校验 short_id 不在其中
5. 考虑使用 `/s/{short_id}` 或 `/go/{short_id}` 前缀，彻底避免冲突——虽然 URL 稍长但省去大量边界问题

**Detection:** 集成测试覆盖所有现有路由 + short_id 路由的优先级正确性。

**Phase:** Bootstrap 重设计阶段，在路由改造的**第一步**就要确定方案。

---

### Pitfall 4: 数据模型从"一用户一主机"到"一用户多账号多主机"的破坏性变更

**What goes wrong:** 当前 `hosts` 表有 `UNIQUE (user_id, slot_key)` 约束，且代码中大量使用 `GetPrimaryHostByUserID` 这类假设用户只有一台主机的查询。新增 Claude 账号模型后，一个用户可能有多台主机（每个 Claude 账号对应一台），所有"用户 -> 唯一主机"的假设都会崩塌。

**Why it happens:** v1.0 合理简化为"一用户一主机"，代码中这个假设已经深入到 bootstrap 流程、entry handler、前端面板。迁移时如果只加了新表而没有全面审查旧代码，旧路径会在多主机场景下 panic 或返回错误数据。

**Consequences:**
- `GetPrimaryHostByUserID` 在用户有多台主机时可能返回错误的那台
- Bootstrap 脚本中 `ssh_port: 2222` 硬编码，多主机时端口冲突
- 出口 IP 绑定逻辑假设一个 host binding 就够了，多主机需要多绑定

**Prevention:**
1. 引入 `claude_accounts` 表作为中间层：`users -> claude_accounts -> hosts`
2. 保留 `GetPrimaryHostByUserID` 但明确加 `LIMIT 1 ORDER BY created_at ASC` 兼容旧路径
3. 新接口一律使用 `account_id` 或 `host_id` 定位主机，不再通过 user_id 隐式推断
4. 迁移脚本必须处理现有数据：为每个现有 user+host 对自动创建一条 claude_account 记录
5. SSH 端口分配必须从硬编码改为动态分配（从端口池分配或按 host_id 映射）

**Detection:** 迁移后运行现有 76 个测试，确认全部通过；新增多账号场景测试。

**Phase:** 数据模型变更阶段，必须先于用户面板和 bootstrap 改造。

---

### Pitfall 5: KasmVNC WebSocket 代理超时和连接中断

**What goes wrong:** 当前 VNC 代理实现使用 `net.DialTimeout` 5 秒连接 + 原始 TCP 双向拷贝（`io.Copy`），没有设置读写超时和 keep-alive。KasmVNC 官方文档要求代理层设置至少 1800 秒（30 分钟）的读写超时。用户闲置超过 Go 默认 TCP 超时后连接会静默断开。

**Why it happens:** 现有实现是 admin-only 的简单代理，偶尔用用没问题。但作为用户自助面板的核心功能暴露给所有用户后，长时间连接、高并发、网络抖动都会放大超时问题。

**Consequences:** 用户 VNC 桌面会话在几分钟空闲后断开，无法重连；或者大量僵死 goroutine（`io.Copy` 永远阻塞）耗尽服务器资源。

**Prevention:**
1. 在 WebSocket hijack 后，对 `clientConn` 和 `targetConn` 都设置 `SetDeadline`，并用 goroutine 定期续期（或使用 `SetKeepAlive` + `SetKeepAlivePeriod`）
2. 读写超时设为 30 分钟以上，与 KasmVNC 官方建议一致
3. 在 `done` channel 双向拷贝完成后，确保两端 conn 都显式 Close（当前只 defer 了一端）
4. 添加连接计数器和超时清理，防止 goroutine 泄漏
5. 考虑使用 `golang.org/x/net/websocket` 或 `nhooyr.io/websocket` 库做标准 WebSocket 握手，而不是手动 hijack 重写 HTTP 请求行

**Detection:** 负载测试：开 10 个 VNC 连接，闲置 10 分钟后检查是否仍然存活；用 pprof 检查 goroutine 数量。

**Phase:** KasmVNC 集成阶段。

---

## Moderate Pitfalls

### Pitfall 6: Entry Password 明文存储

**What goes wrong:** 当前 `entry_password` 是明文存储在数据库中（`user.EntryPassword != body.Password` 直接比较）。当用户自助面板上线后，这个密码同时用于 curl bootstrap 和 Web 登录，安全风险显著升级。

**Prevention:**
1. 用户 Web 登录必须使用 bcrypt hash（与管理员一致）
2. Entry password（curl 用）可以保持为一次性 token 或短密码，但应与 Web 登录密码分离
3. 或者统一为一套密码，全部 bcrypt，bootstrap 脚本中的认证接口也走 hash 比较

**Phase:** 用户认证体系阶段。

---

### Pitfall 7: 同一 React 应用中管理员和用户状态互相污染

**What goes wrong:** 当前前端 `localStorage` 中存 `admin_token`，`apiFetch` 硬编码 `API_BASE = "/v1/admin"`。如果用户面板也在同一 SPA 中，需要同时处理两种 token 和两种 API 前缀。管理员登录后切换到用户视角（或反过来）时，localStorage 中的 token 冲突。

**Prevention:**
1. Token 存储使用不同的 key：`admin_token` vs `user_token`
2. `apiFetch` 改为支持 base URL 参数，或创建两个 fetch 实例
3. 路由守卫根据 token 类型判断，而不是只检查 token 是否存在
4. 如果管理员也是用户，需要明确的"角色切换"机制而非两个 token 共存

**Phase:** 用户自助面板前端阶段。

---

### Pitfall 8: ASCII Art 在不同终端中的显示问题

**What goes wrong:** Bootstrap 脚本中加入 ASCII 欢迎艺术字，但不同终端对字符宽度、编码、颜色码的支持差异很大。CJK 字符在非 UTF-8 终端上乱码，ANSI 颜色码在管道传输（`curl | sh`）中可能被解析为乱码。

**Prevention:**
1. ASCII art 只使用 7-bit ASCII 字符，不使用中文
2. 颜色码使用 `tput` 或检测 `$TERM` 而非硬编码 ANSI 转义序列
3. 检测 `[ -t 1 ]`（stdout 是否为 tty）来决定是否输出颜色
4. 宽度控制在 60 列以内，兼容窄终端
5. 在 `curl | bash` 模式下不输出颜色（stdin 不是 tty 时 tput 无效）

**Phase:** Bootstrap 重设计阶段。

---

### Pitfall 9: Docker 容器 IP 获取方式不可靠

**What goes wrong:** 当前 `getContainerIP` 通过 `docker inspect` 取容器 IP，在 `--network=none` 的容器上可能返回空（因为容器没有常规网络接口，IP 是通过 netns 内部的 veth pair 分配的）。当用户面板需要代理 VNC 到容器时，IP 发现机制可能失效。

**Prevention:**
1. 确认当前 WireGuard/sing-box 注入后容器是否有可路由的 IP
2. 如果 IP 来自 netns 内部，考虑通过 host-agent 查询 namespace 内的 IP
3. 或在容器启动时将 VNC 端口通过 `socat` 或 `nsenter` 暴露到宿主机的 Unix socket
4. 建立明确的容器 IP 发现契约，不依赖 Docker inspect 的 NetworkSettings

**Phase:** KasmVNC 集成阶段。

---

### Pitfall 10: 多主机场景下 SSH 端口冲突

**What goes wrong:** 当前 bootstrap 返回 `ssh_port: 2222` 硬编码。当一个用户拥有多台主机时，所有主机都监听 2222 端口，无法通过同一个宿主机 IP 区分。

**Prevention:**
1. 端口分配改为动态：在 host 创建时从端口池分配唯一端口
2. 端口存储在 `hosts` 表中（新增 `ssh_port` 列）
3. Bootstrap 和 entry handler 从数据库读取端口而非硬编码
4. 端口池管理需要考虑回收（主机删除后端口释放）

**Phase:** 数据模型变更阶段，与多账号模型一起设计。

---

### Pitfall 11: 数据库迁移中断导致不一致状态

**What goes wrong:** 添加新表（`claude_accounts`）和修改现有表的迁移如果中途失败，数据库可能处于半迁移状态：新表已建但外键未加，或旧数据未回填。

**Prevention:**
1. 每个迁移文件内使用事务包裹（`BEGIN; ... COMMIT;`）
2. 回填现有数据的迁移与 DDL 变更分开为两个迁移文件
3. 新列先用 `DEFAULT` 值或 `NULL` 允许，数据回填后再加 `NOT NULL` 约束
4. 在测试环境先用生产数据量级验证迁移

**Phase:** 数据模型变更阶段。

---

## Minor Pitfalls

### Pitfall 12: Cookie 路径冲突

**What goes wrong:** 当前 admin token cookie 路径设为 `/v1/admin/`。如果用户端也设置 cookie（路径 `/v1/user/`），浏览器在跨路径请求时可能带错 cookie，或者同域下的 cookie 互相覆盖。

**Prevention:** 用户 token 和管理员 token 使用不同的 cookie name 和 path，且确保 `SameSite` 和 `HttpOnly` 属性正确。

**Phase:** 用户认证体系阶段。

---

### Pitfall 13: KasmVNC 容器内未启动或端口未就绪

**What goes wrong:** 用户点击 VNC 连接时，容器内的 KasmVNC 进程可能还未启动完成（镜像刚创建、服务初始化中），代理连接失败但错误提示不清晰。

**Prevention:**
1. 容器启动流程中加入 KasmVNC ready check（类似现有 SSH ready check）
2. 用户面板显示明确的"桌面正在启动..."状态
3. 代理层返回 503 + Retry-After header

**Phase:** KasmVNC 集成阶段。

---

### Pitfall 14: Short ID 碰撞和可预测性

**What goes wrong:** 如果 `short_id` 生成算法使用简单的自增或短随机串，可能出现碰撞或被枚举。攻击者可以遍历 `/{short_id}` 发现所有用户的 bootstrap 入口。

**Prevention:**
1. 使用密码学安全随机生成器，至少 6 位 base62（62^6 = 568 亿种可能）
2. 数据库 `UNIQUE` 约束已有（当前已实现）
3. 认证端点加入速率限制，防止暴力枚举
4. 失败响应不泄露 short_id 是否存在（统一返回"invalid credentials"）

**Phase:** Bootstrap 重设计阶段。

---

## Phase-Specific Warnings

| Phase Topic | Likely Pitfall | Severity | Mitigation |
|-------------|---------------|----------|------------|
| 用户认证体系 | JWT 签名密钥共享导致越权 (P1) | Critical | 独立密钥 + role claim + 中间件断言 |
| 用户认证体系 | Entry password 明文存储 (P6) | Moderate | bcrypt 或双密码分离 |
| 用户认证体系 | Cookie 路径冲突 (P12) | Minor | 不同 cookie name + path |
| 用户自助面板 API | 前端角色保护不等于后端鉴权 (P2) | Critical | 所有 handler 必须从 JWT 提取 user_id 过滤 |
| 用户自助面板前端 | 管理员/用户状态互相污染 (P7) | Moderate | 分离 token key、fetch 实例、角色切换机制 |
| Bootstrap 重设计 | URL 路由冲突 (P3) | Critical | 保留词表 + 路由优先级测试 + 考虑前缀方案 |
| Bootstrap 重设计 | ASCII art 终端兼容 (P8) | Moderate | 纯 ASCII + tty 检测 + 60 列宽度 |
| Bootstrap 重设计 | Short ID 可枚举 (P14) | Minor | 密码学随机 + 速率限制 |
| 数据模型变更 | 一用户一主机假设崩塌 (P4) | Critical | 中间层表 + 全面审查旧查询 + 数据回填 |
| 数据模型变更 | SSH 端口冲突 (P10) | Moderate | 动态端口分配 |
| 数据模型变更 | 迁移中断导致不一致 (P11) | Moderate | 事务迁移 + 分步迁移 |
| KasmVNC 集成 | WebSocket 超时和 goroutine 泄漏 (P5) | Critical | 超时设置 + keep-alive + 标准 WS 库 |
| KasmVNC 集成 | 容器 IP 发现不可靠 (P9) | Moderate | 明确的 IP 发现契约 |
| KasmVNC 集成 | VNC 服务未就绪 (P13) | Minor | ready check + 用户状态提示 |

## Recommended Phase Ordering (Based on Pitfalls)

1. **数据模型变更** -- 先做，因为 P4 (Critical) 表明几乎所有后续功能都依赖正确的多账号数据结构
2. **用户认证体系** -- 紧随其后，因为 P1 (Critical) 和 P2 (Critical) 是安全基础
3. **Bootstrap 重设计** -- 在认证就位后做，P3 (Critical) 需要独立解决路由冲突
4. **用户自助面板** -- 依赖认证和数据模型两者就位
5. **KasmVNC 集成** -- 最后做，P5 需要较多工程投入但不阻塞其他功能

## "Looks Done But Isn't" Checklist

- [ ] **JWT 中间件:** AdminAuthMiddleware 是否增加了 role/subject 检查 -- 用用户 token 请求 admin API 应返回 403
- [ ] **数据隔离:** 用户 A 的 token 能否查到用户 B 的资源 -- 交叉请求测试
- [ ] **路由优先级:** `/{short_id}` 是否吞掉了 `/healthz` 或 SPA 路由 -- 全路由集成测试
- [ ] **数据迁移:** 现有 user+host 是否自动获得 claude_account 记录 -- 迁移后查数据
- [ ] **VNC 超时:** 闲置 10 分钟后 VNC 连接是否存活 -- 手动测试
- [ ] **SSH 端口:** 两台主机是否分配了不同端口 -- 多主机创建测试
- [ ] **密码安全:** entry_password 是否仍然明文存储 -- grep 代码确认
- [ ] **goroutine 泄漏:** 多个 VNC 连接断开后 goroutine 数量是否回到基线 -- pprof 检查

## Sources

- 项目代码审查：`admin_auth.go`（JWT 仅验签不验角色）、`entry.go`（明文密码比较）、`admin_vnc_proxy.go`（无超时 WebSocket 代理）、`models.go`（一用户一主机数据模型）、`router.go`（路由注册模式）、`api.ts`（硬编码 admin API base）
- [KasmVNC Reverse Proxy Documentation](https://kasmweb.com/kasmvnc/docs/master/how_to/reverse_proxy.html) -- WebSocket 超时建议 1800s+
- [Kasm Troubleshooting - Reverse Proxies](https://www.kasmweb.com/docs/develop/guide/troubleshooting/reverse_proxies.html) -- RDP keep-alive 和连接断开问题
- [Role-based Access Control in Golang with JWT](https://dev.to/bensonmacharia/role-based-access-control-in-golang-with-jwt-go-ijn) -- Go JWT RBAC 实现模式
- [Auth0 Community: Best practice for role-based authorization in React SPA](https://community.auth0.com/t/best-practice-for-role-based-or-permission-based-authorization-in-a-react-spa/194364) -- 前端 RBAC 不能替代后端鉴权

---
*Pitfalls research for: v1.2 用户自助面板与 Bootstrap 重设计*
*Researched: 2026-03-28*
