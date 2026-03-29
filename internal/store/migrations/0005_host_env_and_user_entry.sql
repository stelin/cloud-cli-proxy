-- Host environment: timezone + hostname for realistic container appearance
ALTER TABLE hosts ADD COLUMN IF NOT EXISTS timezone TEXT NOT NULL DEFAULT 'America/Los_Angeles';
ALTER TABLE hosts ADD COLUMN IF NOT EXISTS hostname TEXT NOT NULL DEFAULT '';

-- User entry: short_id for curl entry + password for authentication
ALTER TABLE users ADD COLUMN IF NOT EXISTS short_id TEXT UNIQUE;
ALTER TABLE users ADD COLUMN IF NOT EXISTS entry_password TEXT NOT NULL DEFAULT '';
