
CREATE TABLE review_invites (
    id                 UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    business_id        UUID REFERENCES businesses(id) ON DELETE CASCADE,
    project_id         UUID REFERENCES projects(id) ON DELETE CASCADE,
    target_email       TEXT NOT NULL,
    token              TEXT NOT NULL UNIQUE,
    created_by         UUID NOT NULL REFERENCES users(id),
    status             INTEGER NOT NULL DEFAULT 0,
    redeemed_review_id UUID REFERENCES reviews(id),
    expires_at         TIMESTAMPTZ NOT NULL,
    created_at         TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CHECK ((business_id IS NOT NULL) <> (project_id IS NOT NULL))
);
CREATE INDEX idx_review_invites_token ON review_invites (token);
CREATE INDEX idx_review_invites_business_id ON review_invites (business_id);
CREATE INDEX idx_review_invites_project_id ON review_invites (project_id);

ALTER TABLE reviews ADD COLUMN verified BOOLEAN NOT NULL DEFAULT FALSE;
