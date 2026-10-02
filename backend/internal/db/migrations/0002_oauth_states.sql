-- OAuth state 存储：用于 /auth/login -> /auth/callback 的 CSRF 防护。
-- state 为一次性凭证，回调校验后立即删除；过期行由 CreateState 顺带清理。
CREATE TABLE IF NOT EXISTS oauth_states (
    state       VARCHAR(128) PRIMARY KEY,
    redirect    TEXT,
    expires_at  TIMESTAMPTZ NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_oauth_states_expires ON oauth_states(expires_at);
