CREATE TABLE webhooks (
    id                UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    target_type       TEXT NOT NULL,
    target_id         TEXT NOT NULL,
    url               TEXT NOT NULL,
    secret            TEXT NOT NULL,
    events            TEXT[] NOT NULL DEFAULT '{}',
    created_by        UUID NOT NULL REFERENCES users (id),
    enabled           BOOLEAN NOT NULL DEFAULT TRUE,
    failure_count     INTEGER NOT NULL DEFAULT 0,
    last_triggered_at TIMESTAMPTZ,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at        TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_webhooks_target_enabled ON webhooks (target_type, target_id, enabled);
