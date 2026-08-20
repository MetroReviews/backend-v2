
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

UPDATE moderation_actions SET target_type = 'business' WHERE target_type = 'listing';
UPDATE reports SET target_type = 'business' WHERE target_type = 'listing';

ALTER TABLE moderation_actions DROP CONSTRAINT moderation_actions_target_type_check;
ALTER TABLE moderation_actions ADD CONSTRAINT moderation_actions_target_type_check
    CHECK (target_type IN ('bot', 'business'));

ALTER TABLE reports DROP CONSTRAINT reports_target_type_check;
ALTER TABLE reports ADD CONSTRAINT reports_target_type_check
    CHECK (target_type IN ('review', 'business', 'bot'));
