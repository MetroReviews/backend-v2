
CREATE INDEX IF NOT EXISTS idx_listings_status_created_at ON listings (status, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_listings_status_rating ON listings (status, avg_rating DESC, review_count DESC);
CREATE INDEX IF NOT EXISTS idx_listings_status_review_count ON listings (status, review_count DESC);
CREATE INDEX IF NOT EXISTS idx_listings_status_category ON listings (status, category_id);

CREATE EXTENSION IF NOT EXISTS pg_trgm;
CREATE INDEX IF NOT EXISTS idx_listings_name_trgm ON listings USING GIN (name gin_trgm_ops);

DROP INDEX IF EXISTS idx_listings_status;

CREATE INDEX IF NOT EXISTS idx_reviews_listing_status_created ON reviews (listing_id, status, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_reviews_bot_status_created ON reviews (bot_id, status, created_at DESC);

DROP INDEX IF EXISTS idx_reviews_listing_id;
DROP INDEX IF EXISTS idx_reviews_bot_id;

