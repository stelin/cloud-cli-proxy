-- User expiry field
ALTER TABLE users ADD COLUMN expires_at TIMESTAMPTZ;

-- Extend events table with user_id column
ALTER TABLE events ADD COLUMN user_id UUID REFERENCES users (id) ON DELETE SET NULL;

-- Event query indexes
CREATE INDEX idx_events_created_at ON events (created_at DESC);
CREATE INDEX idx_events_type_created_at ON events (type, created_at DESC);
CREATE INDEX idx_events_user_id_created_at ON events (user_id, created_at DESC);
CREATE INDEX idx_events_host_id_created_at ON events (host_id, created_at DESC);

-- Partial index for expiry scan (only active users with expiry set)
CREATE INDEX idx_users_expires_at_status ON users (expires_at, status)
  WHERE expires_at IS NOT NULL AND status = 'active';

-- Reconciliation scan indexes
CREATE INDEX idx_hosts_status_running ON hosts (status) WHERE status = 'running';
CREATE INDEX idx_tasks_stale ON tasks (status, updated_at) WHERE status IN ('pending', 'running');
