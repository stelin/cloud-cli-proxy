CREATE TABLE IF NOT EXISTS ssh_keys (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id     UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    purpose     VARCHAR(20) NOT NULL CHECK (purpose IN ('inbound', 'outbound')),
    label       VARCHAR(100) NOT NULL DEFAULT '',
    public_key  TEXT NOT NULL,
    private_key TEXT NOT NULL DEFAULT '',
    key_type    VARCHAR(20) NOT NULL DEFAULT 'ed25519',
    fingerprint VARCHAR(100) NOT NULL DEFAULT '',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_ssh_keys_user_id ON ssh_keys(user_id);
CREATE INDEX IF NOT EXISTS idx_ssh_keys_user_purpose ON ssh_keys(user_id, purpose);

-- Migrate existing keys from users table
INSERT INTO ssh_keys (user_id, purpose, label, public_key, private_key, key_type)
SELECT id, 'outbound', 'default', ssh_public_key, COALESCE(ssh_private_key, ''), COALESCE(ssh_key_type, 'ed25519')
FROM users
WHERE ssh_public_key IS NOT NULL AND ssh_public_key != '';
