-- Webhooks: outbound event notifications for any reviewable subject —
-- currently bots, businesses and projects, but deliberately generic
-- (target_type/target_id as free text, same polymorphic shape
-- moderation_actions/reports already use) so a future subject type needs
-- no schema change here to start firing/receiving events.
--
-- events is the subscription filter: an empty array means "everything",
-- otherwise only the named events (see the webhooks package's Catalog) are
-- delivered. Not CHECK-constrained against a fixed list on purpose — new
-- event names can ship without a migration.
CREATE TABLE webhooks (
    id                UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    target_type       TEXT NOT NULL,
    target_id         TEXT NOT NULL,
    url               TEXT NOT NULL,
    secret            TEXT NOT NULL, -- HMAC-SHA256 signing secret; never returned by a GET, only on creation/rotation
    events            TEXT[] NOT NULL DEFAULT '{}',
    created_by        UUID NOT NULL REFERENCES users (id),
    enabled           BOOLEAN NOT NULL DEFAULT TRUE,
    failure_count     INTEGER NOT NULL DEFAULT 0, -- consecutive delivery failures; auto-disabled past a threshold (see webhooks.recordFailure)
    last_triggered_at TIMESTAMPTZ,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at        TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Every dispatch looks up "enabled webhooks for this exact target" — the
-- one query pattern this table serves.
CREATE INDEX idx_webhooks_target_enabled ON webhooks (target_type, target_id, enabled);
