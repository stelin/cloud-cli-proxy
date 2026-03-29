# Requirements: Cloud CLI Proxy

**Defined:** 2026-03-28
**Core Value:** 给每个用户提供一台开箱即用的 SSH 云主机，并且严格保证其所有出网流量都走受控的指定出口 IP，同时保持"一条命令启动"的体验足够顺滑。

## v1.2 Requirements

Requirements for v1.2 用户自助面板与 Bootstrap 重设计。Each maps to roadmap phases.

### 认证与授权

- [ ] **AUTH-01**: 用户可使用 short_id + 密码登录，获取带 role claim 的 JWT
- [x] **AUTH-02**: 管理员和用户使用同一登录页，登录后根据角色跳转不同面板
- [ ] **AUTH-03**: 用户只能访问自己的主机、出口 IP、Claude 账号等资源，不能看到其他用户的数据

### Bootstrap 体验

- [ ] **BOOT-01**: 用户通过 `curl domain/{short_id}` 进入引导流程，替代现有 `/v1/bootstrap/script` 路径
- [ ] **BOOT-02**: 引导脚本展示产品名称 ASCII 艺术字欢迎界面
- [ ] **BOOT-03**: 容器启动过程通过 SSE 实时推送状态到终端（创建中、配置网络、启动 SSH 等）
- [ ] **BOOT-04**: 启动完成后自动建立 SSH 连接进入容器

### 用户自助面板

- [x] **PANEL-01**: 用户可在自助面板查看自己的主机列表和运行状态
- [x] **PANEL-02**: 用户可在自助面板查看自己主机绑定的出口 IP
- [x] **PANEL-03**: 用户可在自助面板触发主机重建操作
- [ ] **PANEL-04**: 用户可在自助面板通过浏览器直接访问 KasmVNC 远程桌面

### Claude 账号

- [ ] **CLAUDE-01**: 系统支持 claude_accounts 数据模型，一个用户可拥有多个 Claude 账号，每个账号对应一台主机
- [ ] **CLAUDE-02**: 管理员可创建、编辑、删除 Claude 账号并绑定到用户和主机
- [ ] **CLAUDE-03**: 用户可在自助面板查看自己的 Claude 账号信息

## Future Requirements

### 账号交接

- **XFER-01**: 用户可向管理员申请交接账号给其他用户
- **XFER-02**: 管理员可审批或拒绝交接申请

### 增强功能

- **ENH-01**: 用户自选代理节点
- **ENH-02**: 实时流量监控面板

## Out of Scope

| Feature | Reason |
|---------|--------|
| 计费、套餐、余额和自助支付 | 在核心主机生命周期和网络强约束能力验证前不纳入 |
| 多宿主机编排 | v1 限制为单宿主机 |
| 用户自定义任意镜像 | 削弱就绪性、安全性和可支持性 |
| 用户自选代理节点 | 由管理员统一配置 |
| 用户申请交接账号 | 流程未设计清楚，v1.2 暂不做 |

## Traceability

| Requirement | Phase | Status |
|-------------|-------|--------|
| AUTH-01 | Phase 11 | Pending |
| AUTH-02 | Phase 11 | Complete |
| AUTH-03 | Phase 11 | Pending |
| BOOT-01 | Phase 15 | Pending |
| BOOT-02 | Phase 15 | Pending |
| BOOT-03 | Phase 15 | Pending |
| BOOT-04 | Phase 15 | Pending |
| PANEL-01 | Phase 12 | Complete |
| PANEL-02 | Phase 12 | Complete |
| PANEL-03 | Phase 12 | Complete |
| PANEL-04 | Phase 14 | Pending |
| CLAUDE-01 | Phase 11 | Pending |
| CLAUDE-02 | Phase 13 | Pending |
| CLAUDE-03 | Phase 13 | Pending |

**Coverage:**
- v1.2 requirements: 14 total
- Mapped to phases: 14
- Unmapped: 0

---
*Requirements defined: 2026-03-28*
*Last updated: 2026-03-28 after roadmap creation*
