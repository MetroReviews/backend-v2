-- Indexes for query patterns introduced by the reviews platform:
--
--   * listings is browsed by category and looked up by slug on every
--     GET /listings and GET /listings/{slug}.
--   * bots is filtered by `state` on the /queue Discord command and the
--     public bot list, same pattern the old bot_queue index covered.
--   * reviews is looked up per-subject (listing or bot) on every listing/bot
--     detail page, and per-author for a user's review history.
--   * reports/claims are worked from their open/pending queue in the staff
--     panel.

CREATE INDEX IF NOT EXISTS idx_listings_category_id ON listings (category_id);
CREATE INDEX IF NOT EXISTS idx_listings_status ON listings (status);

CREATE INDEX IF NOT EXISTS idx_bots_state_bot_id ON bots (state, bot_id);

CREATE INDEX IF NOT EXISTS idx_moderation_actions_target ON moderation_actions (target_type, target_id);
CREATE INDEX IF NOT EXISTS idx_moderation_actions_action_time ON moderation_actions (action_time DESC);

CREATE INDEX IF NOT EXISTS idx_reviews_listing_id ON reviews (listing_id);
CREATE INDEX IF NOT EXISTS idx_reviews_bot_id ON reviews (bot_id);
CREATE INDEX IF NOT EXISTS idx_reviews_author_id ON reviews (author_id);

CREATE INDEX IF NOT EXISTS idx_review_votes_user_id ON review_votes (user_id);

CREATE INDEX IF NOT EXISTS idx_reports_status ON reports (status);
CREATE INDEX IF NOT EXISTS idx_claims_status ON claims (status);
CREATE INDEX IF NOT EXISTS idx_claims_listing_id ON claims (listing_id);
