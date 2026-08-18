-- Projects: a portfolio/showcase item a business posts — past work shown
-- on its profile (title, description, image, an optional link and
-- completion date). Posting one goes through the same claim/approve/deny
-- review queue as a new business or bot before it's public (see
-- review/action.go's ApplyProjectAction), so it needs the same shape:
-- a status, whoever submitted it, and the staff member who claimed it.
CREATE TABLE projects (
    id           UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    business_id  UUID NOT NULL REFERENCES businesses (id) ON DELETE CASCADE,
    title        TEXT NOT NULL,
    description  TEXT,
    image        TEXT,
    url          TEXT,
    completed_at TIMESTAMPTZ, -- when the work was finished, if known; unset for an ongoing project
    submitted_by UUID NOT NULL REFERENCES users (id), -- the business's verified owner, or staff posting on their behalf
    reviewer     UUID REFERENCES users (id),
    status       INTEGER NOT NULL DEFAULT 0, -- State.PENDING (types.State, shared with bots/businesses)
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_projects_business_status ON projects (business_id, status);
CREATE INDEX idx_projects_status ON projects (status);

-- moderation_actions' target_type already covers 'bot' and 'business' (see
-- 0009) — projects go through the exact same pipeline, so they need to be
-- a valid target too.
ALTER TABLE moderation_actions DROP CONSTRAINT moderation_actions_target_type_check;
ALTER TABLE moderation_actions ADD CONSTRAINT moderation_actions_target_type_check
    CHECK (target_type IN ('bot', 'business', 'project'));
