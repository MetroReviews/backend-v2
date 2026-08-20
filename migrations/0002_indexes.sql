
CREATE INDEX IF NOT EXISTS idx_bot_queue_state_bot_id ON bot_queue (state, bot_id);
CREATE INDEX IF NOT EXISTS idx_bot_queue_list_source ON bot_queue (list_source);

CREATE INDEX IF NOT EXISTS idx_bot_action_bot_id ON bot_action (bot_id);
CREATE INDEX IF NOT EXISTS idx_bot_action_action_time ON bot_action (action_time DESC);
CREATE INDEX IF NOT EXISTS idx_bot_action_list_source ON bot_action (list_source);
