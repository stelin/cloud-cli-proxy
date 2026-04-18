-- Phase 30 Plan 01 · 落地 claude_accounts.persistent_volume_name 列
-- 对齐 30-CONTEXT D-01/D-02/D-10：
--   D-01：v3.0 单 named volume 统一命名 claude-state-{claude_account_id}
--   D-02：列语义 NULL = 尚未分配；禁止空字符串默认值，避免三态
--   D-10：紧随 0013 之后，在空库与自 v2.0 升级库上均可幂等执行
-- 本 migration 仅改 schema，不触碰 HTTP / agent 契约；Phase 33 负责在 volume create 时回写。

ALTER TABLE claude_accounts
    ADD COLUMN IF NOT EXISTS persistent_volume_name TEXT;

-- 回滚路径（由运维在需要时手工执行；migrator 仅 up）：
--   ALTER TABLE claude_accounts DROP COLUMN IF EXISTS persistent_volume_name;
