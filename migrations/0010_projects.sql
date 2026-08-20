CREATE TABLE projects (
    id           UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    business_id  UUID NOT NULL REFERENCES businesses (id) ON DELETE CASCADE,
    title        TEXT NOT NULL,
    description  TEXT,
    image        TEXT,
    url          TEXT,
    completed_at TIMESTAMPTZ,
    submitted_by UUID NOT NULL REFERENCES users (id),
    reviewer     UUID REFERENCES users (id),
    status       INTEGER NOT NULL DEFAULT 0,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_projects_business_status ON projects (business_id, status);
CREATE INDEX idx_projects_status ON projects (status);

ALTER TABLE moderation_actions DROP CONSTRAINT moderation_actions_target_type_check;
ALTER TABLE moderation_actions ADD CONSTRAINT moderation_actions_target_type_check
    CHECK (target_type IN ('bot', 'business', 'project'));
