-- Projects become reviewable, same as businesses and bots: rating rollups
-- on the project itself, and a third subject column on reviews.
ALTER TABLE projects
    ADD COLUMN avg_rating   NUMERIC(3,2) NOT NULL DEFAULT 0,
    ADD COLUMN review_count INTEGER NOT NULL DEFAULT 0;

ALTER TABLE reviews ADD COLUMN project_id UUID REFERENCES projects (id) ON DELETE CASCADE;

-- The original two-way "exactly one of business_id/bot_id" CHECK (added as
-- an unnamed table constraint back in 0006, so it carries the default
-- name Postgres gave it: reviews_check) doesn't generalize to <> once
-- there are three columns — switch to counting how many are set.
ALTER TABLE reviews DROP CONSTRAINT reviews_check;
ALTER TABLE reviews ADD CONSTRAINT reviews_check CHECK (
    (CASE WHEN business_id IS NOT NULL THEN 1 ELSE 0 END) +
    (CASE WHEN bot_id      IS NOT NULL THEN 1 ELSE 0 END) +
    (CASE WHEN project_id  IS NOT NULL THEN 1 ELSE 0 END) = 1
);

ALTER TABLE reviews ADD CONSTRAINT reviews_project_id_author_id_key UNIQUE (project_id, author_id);

CREATE INDEX idx_reviews_project_status_created ON reviews (project_id, status, created_at DESC);

-- moderation_actions/reports already accept 'project' as a target_type
-- (see 0010) — reviews against a project reuse the existing 'review'
-- report target_type, no change needed there.
