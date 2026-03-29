# Phase 11: 认证基础设施与数据迁移 - Discussion Log (Assumptions Mode)

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions captured in CONTEXT.md — this log preserves the analysis.

**Date:** 2026-03-29
**Phase:** 11-认证基础设施与数据迁移
**Mode:** assumptions (--auto)
**Areas analyzed:** JWT 统一认证, 用户表角色模型, 密码体系统一, claude_accounts 数据模型

## Assumptions Presented

### JWT 统一认证方案
| Assumption | Confidence | Evidence |
|------------|-----------|----------|
| 合并管理员硬编码凭证和用户 bootstrap 认证为统一登录端点 | Confident | admin_auth.go:90-91 硬编码 hmac.Equal, bootstrap_auth.go:81 bcrypt |
| 登录凭证使用 short_id + 密码 | Confident | AUTH-01 明确要求, 0005 迁移已有 short_id 字段 |

### 用户表角色模型
| Assumption | Confidence | Evidence |
|------------|-----------|----------|
| users 表添加 role 列，管理员迁移为数据库记录 | Likely | 现有管理员完全绕过 DB (app.go:109-115), AUTH-02 要求共用登录页 |

### 密码体系统一
| Assumption | Confidence | Evidence |
|------------|-----------|----------|
| 统一 bcrypt + password_hash，废弃 entry_password 明文 | Likely | entry.go:104 明文比对, 0005 迁移的 entry_password 字段 |

### claude_accounts 数据模型
| Assumption | Confidence | Evidence |
|------------|-----------|----------|
| 独立 claude_accounts 表，外键关联 users 和 hosts | Confident | 无现有 claude 相关代码, CLAUDE-01 要求, 现有模式为 UUID PK + FK |

## Corrections Made

No corrections — all assumptions confirmed (--auto mode).

## Auto-Resolved

- 用户表角色模型 (Likely): auto-selected `role TEXT NOT NULL DEFAULT 'user'` 列
- 密码体系统一 (Likely): auto-selected 统一 bcrypt，废弃 entry_password

## External Research

- JWT role claim: standard custom claims pattern in golang-jwt/jwt/v5 — resolved without external research
- claude_accounts 字段: deferred to Phase 13 for business requirement clarification
