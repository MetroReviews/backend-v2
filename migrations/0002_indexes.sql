-- Indexes for query patterns that were previously doing sequential scans:
--
--   * bot_queue is filtered by `state` and sorted by (state, bot_id) on
--     every /queue view (Discord command) and every bot listing.
--   * bot_action is filtered by bot_id (per-bot history) and sorted by
--     action_time DESC on GET /actions.
--   * list_source is a foreign key into bot_list on both bot_queue and
--     bot_action; Postgres does not index foreign keys automatically, so
--     both the ON DELETE CASCADE and any per-list lookups were unindexed.

CREATE INDEX IF NOT EXISTS idx_bot_queue_state_bot_id ON bot_queue (state, bot_id);
CREATE INDEX IF NOT EXISTS idx_bot_queue_list_source ON bot_queue (list_source);

CREATE INDEX IF NOT EXISTS idx_bot_action_bot_id ON bot_action (bot_id);
CREATE INDEX IF NOT EXISTS idx_bot_action_action_time ON bot_action (action_time DESC);
CREATE INDEX IF NOT EXISTS idx_bot_action_list_source ON bot_action (list_source);
