
CREATE TABLE local_accounts (
    user_id       UUID PRIMARY KEY REFERENCES users (id) ON DELETE CASCADE,
    email         TEXT NOT NULL UNIQUE,
    password_hash TEXT NOT NULL,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE sessions (
    token      TEXT PRIMARY KEY,
    user_id    UUID NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    expires_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_sessions_user_id ON sessions (user_id);
CREATE INDEX idx_sessions_expires_at ON sessions (expires_at);

INSERT INTO sessions (token, user_id, expires_at)
SELECT session_token, user_id, session_expires_at
FROM discord_accounts
WHERE session_token IS NOT NULL AND session_expires_at IS NOT NULL;

ALTER TABLE discord_accounts DROP COLUMN session_token;
ALTER TABLE discord_accounts DROP COLUMN session_expires_at;
