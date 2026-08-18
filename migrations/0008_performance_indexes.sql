-- Indexes matching the exact query shapes the handlers issue today (see
-- routes/listings/get.go and routes/reviews/get.go), replacing the earlier
-- single-column indexes that only partially matched them.
--
--   * getAllListings filters on `status` (almost always present), optionally
--     `category_id` and/or a `name ILIKE '%q%'` search, then sorts by one of
--     created_at DESC / (avg_rating DESC, review_count DESC) / review_count
--     DESC. None of that was covered by an index that also satisfied the
--     ORDER BY, so every browse paid a filesort — and the ILIKE search,
--     having a leading wildcard, could never use a plain btree index at all.
--   * getListingReviews/getBotReviews filter (subject_id, status) and sort
--     by created_at DESC; the old single-column subject_id indexes couldn't
--     help with the status filter or the sort.

CREATE INDEX IF NOT EXISTS idx_listings_status_created_at ON listings (status, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_listings_status_rating ON listings (status, avg_rating DESC, review_count DESC);
CREATE INDEX IF NOT EXISTS idx_listings_status_review_count ON listings (status, review_count DESC);
CREATE INDEX IF NOT EXISTS idx_listings_status_category ON listings (status, category_id);

-- name search: ILIKE '%q%' has a leading wildcard, so no btree index (single-
-- or multi-column) can ever help it — only a trigram index can.
CREATE EXTENSION IF NOT EXISTS pg_trgm;
CREATE INDEX IF NOT EXISTS idx_listings_name_trgm ON listings USING GIN (name gin_trgm_ops);

-- Superseded by the composites above (status is their leading column, so
-- they serve every query the standalone index did, plus more).
DROP INDEX IF EXISTS idx_listings_status;
-- Not superseded: idx_listings_category_id still serves the staff `all=true`
-- + category case, which has no status filter at all.

CREATE INDEX IF NOT EXISTS idx_reviews_listing_status_created ON reviews (listing_id, status, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_reviews_bot_status_created ON reviews (bot_id, status, created_at DESC);

DROP INDEX IF EXISTS idx_reviews_listing_id;
DROP INDEX IF EXISTS idx_reviews_bot_id;

-- Left alone, on purpose:
--   * bots: idx_bots_state_bot_id (state, bot_id) already matches
--     getAllBots's exact `WHERE state = X ORDER BY bot_id ASC`.
--   * claims/reports: existing indexes already match every query issued
--     against them today.
--   * roles.permissions: only ever a handful of hand-configured rows, so a
--     GIN index would cost more to maintain than a sequential scan ever
--     costs to run.
