# Roadmap: Cloud CLI Proxy

## Milestones

- ✅ **v1.0 MVP** — Phases 1-6 (shipped 2026-03-28) — [Archive](milestones/v1.0-ROADMAP.md)
- ✅ **v1.1 支持代理协议出网** — Phases 7-10 (shipped 2026-03-28) — [Archive](milestones/v1.1-ROADMAP.md)
- 🚧 **v1.2 用户自助面板与 Bootstrap 重设计** — Phases 11-16 (partially shipped, remaining deferred)
- 🚧 **v1.3 claude-shell 本地透明代理** — Phases 17-23 (in progress)

## Phases

<details>
<summary>✅ v1.0 MVP (Phases 1-6) — SHIPPED 2026-03-28</summary>

- [x] Phase 1: 基础控制面与主机代理 (3/3 plans) — completed 2026-03-26
- [x] Phase 2: 隧道出网强制层 (3/3 plans) — completed 2026-03-27
- [x] Phase 3: 启动入口与 SSH 接入 (3/3 plans) — completed 2026-03-27
- [x] Phase 4: 后台管理界面 (3/3 plans) — completed 2026-03-27
- [x] Phase 5: 到期、审计与清理 (3/3 plans) — completed 2026-03-27
- [x] Phase 6: 加固与 MVP 就绪 (4/4 plans) — completed 2026-03-28

</details>

<details>
<summary>✅ v1.1 支持代理协议出网 (Phases 7-10) — SHIPPED 2026-03-28</summary>

- [x] Phase 7: 数据层与类型化 (3/3 plans) — completed 2026-03-28
- [x] Phase 8: SingBoxProvider 与受管镜像 (3/3 plans) — completed 2026-03-28
- [x] Phase 9: 前端适配与代理测试 (3/3 plans) — completed 2026-03-28
- [x] Phase 10: 技术债务清理 (2/2 plans) — completed 2026-03-28

</details>

### 🚧 v1.2 用户自助面板与 Bootstrap 重设计

**Milestone Goal:** 建立用户认证与自助面板体系，让用户可以独立查看和管理自己的资源；同时重设计 Bootstrap 入口，提升首次接入体验。

- [x] **Phase 11: 认证基础设施与数据迁移** — 用户登录认证体系与 Claude 账号数据模型 (completed 2026-03-29)
- [x] **Phase 12: 用户自助 API 与前端路由** — 用户自助面板骨架与角色路由 (completed 2026-03-29)
- [ ] **Phase 13: 账号管理与用户资源视图** — 账号 CRUD、有效期、售后换号、用户资源汇总
- [ ] **Phase 14: KasmVNC 用户面** — 用户通过面板直接访问远程桌面
- [ ] **Phase 15: Bootstrap 重设计** — 短 URL 入口与实时状态推送
- [ ] **Phase 16: 级联禁用与到期治理** — 用户/账号/主机到期联动与自动关机

## Phase Details

### Phase 11: 认证基础设施与数据迁移
**Goal**: 用户可以使用自己的凭证登录系统，系统能区分管理员和普通用户角色，Claude 账号数据模型就绪
**Depends on**: Phase 10 (v1.1 基线)
**Requirements**: AUTH-01, AUTH-02, AUTH-03, CLAUDE-01
**Success Criteria** (what must be TRUE):
  1. 用户使用 short_id + 密码登录后获取 JWT，JWT 中包含 role claim 区分管理员和普通用户
  2. 管理员和用户共用同一登录页面，登录后根据角色自动跳转到对应的面板
  3. 用户 API 请求只能访问自己的资源，尝试访问他人资源返回 403
  4. claude_accounts 表已创建，支持一个用户拥有多个 Claude 账号且每个账号关联一台主机
**Plans**: TBD

### Phase 12: 用户自助 API 与前端路由
**Goal**: 用户可以在自助面板中查看自己的主机状态、出口 IP 并执行主机重建
**Depends on**: Phase 11
**Requirements**: PANEL-01, PANEL-02, PANEL-03
**Success Criteria** (what must be TRUE):
  1. 用户登录后看到自己的主机列表，包含运行状态和基本信息
  2. 用户可以查看每台主机绑定的出口 IP 信息
  3. 用户可以对自己的主机触发重建操作，重建过程有状态反馈
  4. 用户面板与管理员面板共存于同一 React 应用，通过角色路由守卫隔离
**Plans**: 2 plans
Plans:
- [x] 12-01-PLAN.md — 用户自助 API 后端（UserHostsHandler + 查询 + 路由注册）
- [x] 12-02-PLAN.md — 用户自助面板前端（API 客户端 + 主机列表页 + 详情页 + 重建）
**UI hint**: yes

### Phase 13: 账号管理与用户资源视图
**Goal**: 建立完整的账号生命周期管理体系，每个账号与主机一一绑定，支持备注、有效期、售后换号；用户管理页能直观展示用户拥有的所有资源
**Depends on**: Phase 12
**Requirements**: ACCT-01, ACCT-02, ACCT-03, ACCT-04, ACCT-05
**Success Criteria** (what must be TRUE):
  1. 管理后台新增"账号管理"页面，支持 CRUD，每个账号可设置备注、有效期（默认 1 个月）、关联用户和主机
  2. 账号与主机是一一对应关系，一个账号只能绑定一个主机
  3. 管理后台有"售后换号"按钮，可直接替换用户的账号，旧账号进入售后记录
  4. 用户管理详情页展示该用户拥有的所有账号和主机信息
  5. 主机列表、账号列表中，即将过期（3 天内）的条目红色标记并排名靠前
  6. 每个主机、账号、出口 IP 都支持设置有效期（到 xx 时间过期）
  7. 用户在自助面板可以看到管理员分配给自己的账号信息
  8. 账号的创建、删除、换号操作记录在事件日志中
**Plans**: TBD
**UI hint**: yes

### Phase 14: KasmVNC 用户面
**Goal**: 用户可以在自助面板中直接通过浏览器访问容器内的远程桌面
**Depends on**: Phase 12
**Requirements**: PANEL-04
**Success Criteria** (what must be TRUE):
  1. 用户在主机详情页可以看到 KasmVNC 访问入口并点击打开远程桌面
  2. KasmVNC 连接通过权限校验，用户只能访问自己的主机桌面
  3. WebSocket 连接稳定，Nginx 正确处理 WebSocket 升级头和超时设置
**Plans**: TBD
**UI hint**: yes

### Phase 15: Bootstrap 重设计
**Goal**: 用户通过更短的 URL 和更流畅的终端交互完成首次接入，全程有实时状态反馈
**Depends on**: Phase 11
**Requirements**: BOOT-01, BOOT-02, BOOT-03, BOOT-04
**Success Criteria** (what must be TRUE):
  1. 用户通过 `curl domain/{short_id}` 进入引导流程，不再需要记忆长路径
  2. 引导脚本启动后展示产品名称 ASCII 艺术字欢迎界面
  3. 容器启动过程中终端实时显示各阶段状态（创建中、配置网络、启动 SSH 等），由 SSE 驱动
  4. 启动完成后自动建立 SSH 连接，用户无需手动输入 SSH 命令
**Plans**: TBD

### Phase 16: 级联禁用与到期治理
**Goal**: 用户、账号、主机三个维度的到期和禁用实现联动——任一上游实体被禁用或过期时，自动级联关闭其下游资源，保证不留"孤儿"运行容器
**Depends on**: Phase 13
**Requirements**: CASCADE-01, CASCADE-02, CASCADE-03
**Success Criteria** (what must be TRUE):
  1. 禁用用户时，该用户关联的所有主机立即自动关机（stop），所有账号状态变为 disabled
  2. 用户过期时，行为与禁用一致——主机关机、账号停用
  3. 主机到期（3 天内）时，到期扫描器自动关闭主机；到期后主机无法再启动，直到管理员续期
  4. 账号到期时，账号状态自动变为 expired，前端展示过期标记
  5. 用户续费（重新激活或延长有效期）后，可以重新启动主机和账号，数据不丢失
  6. 到期扫描器支持主机、账号、出口 IP 三个维度的过期检测，每分钟一次
  7. 所有级联操作都记录在事件日志中，包含操作原因（用户禁用、到期等）
**Plans**: TBD

### 🚧 v1.3 claude-shell 本地透明代理

**Milestone Goal:** 交付单一 Go 二进制 `claude` 命令，透明启动 Docker 容器运行 Claude Code，容器内全流量走代理出口，设备指纹完全伪装，用户和 Claude Code 均无感知。

- [ ] **Phase 17: 镜像与 Entrypoint 基线** — 容器镜像和启动编排就绪，Claude Code 通过官方安装脚本运行
- [ ] **Phase 18: 网络隔离与分流** — sing-box tun + nftables 全流量代理，公网走出口，私网回连宿主机
- [ ] **Phase 19: CLI 骨架与 Docker 编排** — Go 二进制 `claude` 命令的基础框架、配置和 Docker 编排
- [ ] **Phase 20: TTY 透传与交互体验** — 终端尺寸、信号、退出码完全透传，交互体验无差异
- [ ] **Phase 21: 指纹伪造与反检测** — 系统级设备指纹伪装和容器标记清除
- [ ] **Phase 22: 验证与自检** — verify 子命令一键检测出口 IP、DNS、指纹和容器标记
- [ ] **Phase 23: 混淆构建与交付** — garble 混淆产出可直接替换的单一二进制

## Phase Details — v1.3

### Phase 17: 镜像与 Entrypoint 基线
**Goal**: 容器镜像和 entrypoint 就绪，Claude Code 通过官方安装脚本正确安装并启动
**Depends on**: Nothing (v1.3 首个阶段，与 v1.2 无代码依赖)
**Requirements**: INFRA-01, INFRA-02, INFRA-03
**Success Criteria** (what must be TRUE):
  1. docker build 产出可用镜像，镜像包含 sing-box 二进制和基础开发工具
  2. Claude Code 通过官方 curl 安装脚本（Bun standalone）安装，容器内 `claude` 命令可执行
  3. entrypoint 按"网络配置 → 指纹伪造 → 反检测 → Claude Code"顺序编排，各步骤失败时输出明确错误
  4. DISABLE_AUTOUPDATER=1 生效，Claude Code 不会在运行时触发自动更新
**Plans**: TBD

### Phase 18: 网络隔离与分流
**Goal**: 容器内所有出站流量强制走代理出口，DNS 不泄漏，本地流量正确回连宿主机
**Depends on**: Phase 17
**Requirements**: NET-01, NET-02, NET-03, NET-04, NET-05
**Success Criteria** (what must be TRUE):
  1. sing-box tun 接管容器内所有出站流量，外网请求走配置的代理出口 IP
  2. nftables 默认拒绝策略生效，绕过 tun 的直连外网请求被丢弃
  3. 外网域名的 DNS 查询通过代理通道解析，不通过宿主机或容器默认 DNS 泄漏
  4. 本地地址（127.0.0.1、10.0.0.0/8、172.16.0.0/12、192.168.0.0/16）可通过 host-gateway 回连宿主机
  5. 支持 SOCKS5、HTTP、VMess、Shadowsocks、Trojan 五种代理协议出站
**Plans**: TBD

### Phase 19: CLI 骨架与 Docker 编排
**Goal**: 用户可以在终端执行 `claude` 命令，由 Go 二进制完成配置加载、Docker 检测和容器启动的基础闭环
**Depends on**: Phase 18
**Requirements**: CLI-01, CLI-03, CLI-05, CLI-06, BUILD-02
**Success Criteria** (what must be TRUE):
  1. `claude` 命令无子命令时透传所有参数给容器内 Claude Code，基本输入输出可用
  2. `claude init` 在 ~/.claude-shell/ 生成包含代理、指纹、网络选项的 config.yaml 配置模板
  3. Docker 不可用时给出明确中文错误提示；镜像不存在时自动拉取并显示进度
  4. claude-shell/ 子目录拥有独立 go.mod，与 cloud-cli-proxy 主项目零依赖
**Plans**: TBD

### Phase 20: TTY 透传与交互体验
**Goal**: 容器内 Claude Code 的终端交互与直接运行原生 `claude` 无差异——尺寸、信号、退出码完全透传
**Depends on**: Phase 19
**Requirements**: CLI-02
**Success Criteria** (what must be TRUE):
  1. docker run 以交互模式启动，bind mount 当前目录到 /workspace 作为工作目录
  2. 终端窗口 resize 时 SIGWINCH 正确传递到容器内进程，Claude Code 界面跟随调整
  3. Ctrl+C / Ctrl+\ 等信号正确转发到容器，容器退出码透传给宿主机 CLI 进程
  4. 容器退出时自动清理（--rm），不留孤儿容器或残留网络资源
**Plans**: TBD

### Phase 21: 指纹伪造与反检测
**Goal**: 容器的设备指纹完全伪装，常规容器检测手段无法识别出 Docker 环境
**Depends on**: Phase 20
**Requirements**: SPOOF-01, SPOOF-02, SPOOF-03, SPOOF-04
**Success Criteria** (what must be TRUE):
  1. /etc/machine-id 包含基于配置派生的稳定伪造值，重启容器后保持一致
  2. 容器 hostname 通过 Docker --hostname 设为配置的伪造主机名
  3. 容器内 cat /proc/cpuinfo 和 cat /proc/meminfo 显示伪造的硬件信息（通过 docker run -v 注入）
  4. /.dockerenv 文件不存在、/proc/1/cgroup 无 docker 关键字、container 环境变量已清除
**Plans**: TBD

### Phase 22: 验证与自检
**Goal**: 用户可以一键验证容器环境的网络出口、DNS 路径、设备指纹和容器标记是否符合预期
**Depends on**: Phase 21
**Requirements**: CLI-04
**Success Criteria** (what must be TRUE):
  1. `claude verify` 在容器内运行检测脚本，输出出口 IP 是否匹配配置的代理出口
  2. verify 检测 DNS 查询是否走代理通道，报告是否存在泄漏
  3. verify 检测 machine-id、hostname、/proc/* 文件是否为伪造值
  4. verify 检测容器标记（/.dockerenv、cgroup、环境变量）是否已清除
  5. 所有检测项以清晰的 ✓/✗ 状态逐项输出，便于排查
**Plans**: TBD

### Phase 23: 混淆构建与交付
**Goal**: 交付经 garble 混淆的单一 Go 二进制，可直接放入 PATH 替代原生 `claude` 命令
**Depends on**: Phase 22
**Requirements**: BUILD-01
**Success Criteria** (what must be TRUE):
  1. garble build 成功产出单一可执行二进制文件，体积合理
  2. 混淆后二进制的所有功能（启动、init、verify）与未混淆版本行为一致
  3. 二进制可直接放入 PATH 替代原生 claude 命令使用，用户无感知差异
**Plans**: TBD

## Progress

**Execution Order:**
Phases execute in numeric order: 17 → 18 → 19 → 20 → 21 → 22 → 23

| Phase | Milestone | Plans Complete | Status | Completed |
|-------|-----------|----------------|--------|-----------|
| 1. 基础控制面与主机代理 | v1.0 | 3/3 | Complete | 2026-03-26 |
| 2. 隧道出网强制层 | v1.0 | 3/3 | Complete | 2026-03-27 |
| 3. 启动入口与 SSH 接入 | v1.0 | 3/3 | Complete | 2026-03-27 |
| 4. 后台管理界面 | v1.0 | 3/3 | Complete | 2026-03-27 |
| 5. 到期、审计与清理 | v1.0 | 3/3 | Complete | 2026-03-27 |
| 6. 加固与 MVP 就绪 | v1.0 | 4/4 | Complete | 2026-03-28 |
| 7. 数据层与类型化 | v1.1 | 3/3 | Complete | 2026-03-28 |
| 8. SingBoxProvider 与受管镜像 | v1.1 | 3/3 | Complete | 2026-03-28 |
| 9. 前端适配与代理测试 | v1.1 | 3/3 | Complete | 2026-03-28 |
| 10. 技术债务清理 | v1.1 | 2/2 | Complete | 2026-03-28 |
| 11. 认证基础设施与数据迁移 | v1.2 | 1/3 | Complete | 2026-03-29 |
| 12. 用户自助 API 与前端路由 | v1.2 | 2/2 | Complete | 2026-03-29 |
| 13. 账号管理与用户资源视图 | v1.2 | 0/0 | Not started | - |
| 14. KasmVNC 用户面 | v1.2 | 0/0 | Not started | - |
| 15. Bootstrap 重设计 | v1.2 | 0/0 | Not started | - |
| 16. 级联禁用与到期治理 | v1.2 | 0/0 | Not started | - |
| 17. 镜像与 Entrypoint 基线 | v1.3 | 0/0 | Not started | - |
| 18. 网络隔离与分流 | v1.3 | 0/0 | Not started | - |
| 19. CLI 骨架与 Docker 编排 | v1.3 | 0/0 | Not started | - |
| 20. TTY 透传与交互体验 | v1.3 | 0/0 | Not started | - |
| 21. 指纹伪造与反检测 | v1.3 | 0/0 | Not started | - |
| 22. 验证与自检 | v1.3 | 0/0 | Not started | - |
| 23. 混淆构建与交付 | v1.3 | 0/0 | Not started | - |

---
*Last updated: 2026-04-09 — v1.3 roadmap created*
