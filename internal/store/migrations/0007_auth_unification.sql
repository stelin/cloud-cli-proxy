-- 1. users 表增加 role 列（per D-04）
ALTER TABLE users ADD COLUMN IF NOT EXISTS role TEXT NOT NULL DEFAULT 'user';

-- 2. 废弃 entry_password，保留列但标记将来删除（per D-07）
-- 注意：不在此迁移中 DROP COLUMN，避免破坏现有代码。entry.go 改造在 Plan 02 中完成后可安全删除。

-- 3. claude_accounts 表（per D-09, D-10）
CREATE TABLE IF NOT EXISTS claude_accounts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    host_id UUID REFERENCES hosts (id) ON DELETE SET NULL,
    email TEXT NOT NULL,
    display_name TEXT NOT NULL DEFAULT '',
    status TEXT NOT NULL DEFAULT 'active',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_claude_accounts_user_id ON claude_accounts (user_id);
CREATE INDEX IF NOT EXISTS idx_claude_accounts_host_id ON claude_accounts (host_id);
CREATE UNIQUE INDEX IF NOT EXISTS idx_claude_accounts_email ON claude_accounts (email);
