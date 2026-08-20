ALTER TABLE projects
    ADD COLUMN avg_rating   NUMERIC(3,2) NOT NULL DEFAULT 0,
    ADD COLUMN review_count INTEGER NOT NULL DEFAULT 0;

ALTER TABLE reviews ADD COLUMN project_id UUID REFERENCES projects (id) ON DELETE CASCADE;

ALTER TABLE reviews DROP CONSTRAINT reviews_check;
ALTER TABLE reviews ADD CONSTRAINT reviews_check CHECK (
    (CASE WHEN business_id IS NOT NULL THEN 1 ELSE 0 END) +
    (CASE WHEN bot_id      IS NOT NULL THEN 1 ELSE 0 END) +
    (CASE WHEN project_id  IS NOT NULL THEN 1 ELSE 0 END) = 1
);

ALTER TABLE reviews ADD CONSTRAINT reviews_project_id_author_id_key UNIQUE (project_id, author_id);

CREATE INDEX idx_reviews_project_status_created ON reviews (project_id, status, created_at DESC);

