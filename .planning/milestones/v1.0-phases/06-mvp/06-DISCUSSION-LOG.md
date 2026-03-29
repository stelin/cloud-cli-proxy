# Phase 6: 加固与 MVP 就绪 - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-03-27
**Phase:** 06-mvp
**Areas discussed:** 冒烟测试策略, 部署文档范围, 体验打磨方向, 上线前检查清单
**Mode:** --auto (all decisions auto-selected using recommended defaults)

---

## 冒烟测试策略

| Option | Description | Selected |
|--------|-------------|----------|
| Go integration + shell scripts | 集成测试需真实 Docker daemon（build tag），快速层用 httptest mock | ✓ |
| 纯 shell 端到端脚本 | 只通过 shell 脚本验证 API，无 Go 测试层 | |
| 第三方测试框架 (testify/ginkgo) | 引入外部测试框架 | |

**User's choice:** [auto] Go integration + shell scripts (recommended default)
**Notes:** 沿用项目现有 Go 标准测试模式，不引入额外依赖。集成测试用 build tag 隔离，CI 可按需开关。

---

## 部署文档范围

| Option | Description | Selected |
|--------|-------------|----------|
| Markdown + 自动化脚本并行 | 手册提供理解，脚本提供可执行路径 | ✓ |
| 纯 Markdown 手册 | 只有文字说明，无可执行脚本 | |
| 纯脚本 (一键部署) | 只有脚本，无文档解释 | |

**User's choice:** [auto] Markdown + 自动化脚本并行 (recommended default)
**Notes:** 目标读者为有 Linux 运维经验的技术人员。覆盖首次部署、日常运维和常见排障，灾难恢复简要附录。

---

## 体验打磨方向

| Option | Description | Selected |
|--------|-------------|----------|
| 终端 + 后台 + 运维三线并行 | 同时完善 bootstrap 错误提示、后台 UI 交互和运维日志/健康检查 | ✓ |
| 仅终端侧 | 只打磨 bootstrap 体验 | |
| 仅后台侧 | 只补齐后台 UI 细节 | |

**User's choice:** [auto] 终端 + 后台 + 运维三线并行 (recommended default)
**Notes:** Phase 6 作为 MVP 最终打磨阶段，三个维度缺一不可。

---

## 上线前检查清单

| Option | Description | Selected |
|--------|-------------|----------|
| 安全 + 稳定性 + 可运维性全覆盖 | 审查 API 权限、优雅关闭、连接池、日志、健康检查、备份 | ✓ |
| 仅安全审查 | 只覆盖权限和敏感信息 | |
| 仅稳定性 | 只覆盖资源管理和优雅关闭 | |

**User's choice:** [auto] 安全 + 稳定性 + 可运维性全覆盖 (recommended default)
**Notes:** MVP 交付门槛需要全面通过，不能只覆盖单一维度。

---

## Claude's Discretion

- 测试 fixture 组织、文档章节编排、日志字段命名、健康检查超时参数、前端组件交互细节

## Deferred Ideas

None — discussion stayed within phase scope.
