-- Product rename: "Listing" becomes "Business" everywhere except the bot
-- list, which stays exactly what it was. Same entity, same review queue,
-- just a name that better fits "any reviewable service or business" than
-- the more classified-ad-flavored "listing" did.
--
-- Table/column renames carry every dependent constraint, index and FK
-- along automatically (Postgres tracks them by OID, not by name), so this
-- only needs to touch the two places that don't follow for free: existing
-- 'listing' target_type values (plain text, not an enum) and the CHECK
-- constraints that validate them.
--
-- Explicitly-named indexes are renamed too, for discoverability — but the
-- implicit ones Postgres auto-generated (listings_pkey, listings_slug_key,
-- listings_category_id_fkey, ...) are left with their old names. That's
-- cosmetic staleness only: nothing here or in application code looks a
-- constraint/index up by name, so renaming ~10 more of them for symmetry
-- isn't worth the risk of a typo'd name failing this migration outright.

ALTER TABLE listings RENAME TO businesses;
ALTER TABLE reviews RENAME COLUMN listing_id TO business_id;
ALTER TABLE claims RENAME COLUMN listing_id TO business_id;

ALTER INDEX idx_listings_category_id RENAME TO idx_businesses_category_id;
ALTER INDEX idx_listings_status_created_at RENAME TO idx_businesses_status_created_at;
ALTER INDEX idx_listings_status_rating RENAME TO idx_businesses_status_rating;
ALTER INDEX idx_listings_status_review_count RENAME TO idx_businesses_status_review_count;
ALTER INDEX idx_listings_status_category RENAME TO idx_businesses_status_category;
ALTER INDEX idx_listings_name_trgm RENAME TO idx_businesses_name_trgm;
ALTER INDEX idx_claims_listing_id RENAME TO idx_claims_business_id;
ALTER INDEX idx_reviews_listing_status_created RENAME TO idx_reviews_business_status_created;
-- idx_listings_status and idx_reviews_listing_id were already dropped by
-- 0008 (superseded by the composite indexes above) — nothing to rename.

UPDATE moderation_actions SET target_type = 'business' WHERE target_type = 'listing';
UPDATE reports SET target_type = 'business' WHERE target_type = 'listing';

ALTER TABLE moderation_actions DROP CONSTRAINT moderation_actions_target_type_check;
ALTER TABLE moderation_actions ADD CONSTRAINT moderation_actions_target_type_check
    CHECK (target_type IN ('bot', 'business'));

ALTER TABLE reports DROP CONSTRAINT reports_target_type_check;
ALTER TABLE reports ADD CONSTRAINT reports_target_type_check
    CHECK (target_type IN ('review', 'business', 'bot'));
