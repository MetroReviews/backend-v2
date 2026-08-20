
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS pg_trgm;


CREATE TABLE users (
    id         UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    username   TEXT,
    avatar     TEXT,
    bio        TEXT,
    is_staff   BOOLEAN NOT NULL DEFAULT FALSE,
    banned     BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE discord_accounts (
    discord_id BIGINT PRIMARY KEY,
    user_id    UUID NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    nonce      TEXT,
    linked_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_discord_accounts_user_id ON discord_accounts (user_id);

CREATE TABLE local_accounts (
    user_id       UUID PRIMARY KEY REFERENCES users (id) ON DELETE CASCADE,
    email         TEXT NOT NULL UNIQUE,
    password_hash TEXT NOT NULL,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE sessions (
    token      TEXT PRIMARY KEY,
    user_id    UUID NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    expires_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_sessions_user_id ON sessions (user_id);
CREATE INDEX idx_sessions_expires_at ON sessions (expires_at);


CREATE TABLE roles (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name            TEXT NOT NULL UNIQUE,
    discord_role_id BIGINT UNIQUE,
    permissions     TEXT[] NOT NULL DEFAULT '{}',
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE user_roles (
    user_id    UUID NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    role_id    UUID NOT NULL REFERENCES roles (id) ON DELETE CASCADE,
    granted_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (user_id, role_id)
);
CREATE INDEX idx_user_roles_role_id ON user_roles (role_id);


CREATE TABLE categories (
    id          UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    slug        TEXT NOT NULL UNIQUE,
    name        TEXT NOT NULL,
    description TEXT,
    icon        TEXT
);


CREATE TABLE businesses (
    id             UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    category_id    UUID NOT NULL REFERENCES categories (id),
    slug           TEXT NOT NULL UNIQUE,
    name           TEXT NOT NULL,
    description    TEXT,
    website        TEXT,
    logo           TEXT,
    banner         TEXT,
    address        TEXT,
    city           TEXT,
    country        TEXT,
    metadata       JSONB NOT NULL DEFAULT '{}',
    owner_id       UUID REFERENCES users (id),
    submitted_by   UUID NOT NULL REFERENCES users (id),
    reviewer       UUID REFERENCES users (id),
    status         INTEGER NOT NULL DEFAULT 0,
    avg_rating     NUMERIC(3,2) NOT NULL DEFAULT 0,
    review_count   INTEGER NOT NULL DEFAULT 0,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    latitude       DOUBLE PRECISION,
    longitude      DOUBLE PRECISION,
    gallery        TEXT[] NOT NULL DEFAULT '{}',
    featured       BOOLEAN NOT NULL DEFAULT FALSE,
    featured_until TIMESTAMPTZ,
    view_count     INTEGER NOT NULL DEFAULT 0,
    search_vector  tsvector GENERATED ALWAYS AS (
        setweight(to_tsvector('english', coalesce(name, '')), 'A') ||
        setweight(to_tsvector('english', coalesce(description, '')), 'B')
    ) STORED
);

CREATE TABLE projects (
    id            UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    business_id   UUID NOT NULL REFERENCES businesses (id) ON DELETE CASCADE,
    title         TEXT NOT NULL,
    description   TEXT,
    image         TEXT,
    url           TEXT,
    completed_at  TIMESTAMPTZ,
    submitted_by  UUID NOT NULL REFERENCES users (id),
    reviewer      UUID REFERENCES users (id),
    status        INTEGER NOT NULL DEFAULT 0,
    avg_rating    NUMERIC(3,2) NOT NULL DEFAULT 0,
    review_count  INTEGER NOT NULL DEFAULT 0,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    search_vector tsvector GENERATED ALWAYS AS (
        setweight(to_tsvector('english', coalesce(title, '')), 'A') ||
        setweight(to_tsvector('english', coalesce(description, '')), 'B')
    ) STORED
);

CREATE TABLE moderation_actions (
    id          UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    target_type TEXT NOT NULL CHECK (target_type IN ('business', 'project')),
    target_id   TEXT NOT NULL,
    action      INTEGER NOT NULL,
    reason      TEXT NOT NULL,
    reviewer    UUID NOT NULL REFERENCES users (id),
    action_time TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE reviews (
    id                UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    business_id       UUID REFERENCES businesses (id) ON DELETE CASCADE,
    project_id        UUID REFERENCES projects (id) ON DELETE CASCADE,
    author_id         UUID NOT NULL REFERENCES users (id),
    rating            SMALLINT NOT NULL CHECK (rating BETWEEN 1 AND 5),
    title             TEXT,
    body              TEXT NOT NULL,
    owner_response    TEXT,
    owner_response_at TIMESTAMPTZ,
    helpful_count     INTEGER NOT NULL DEFAULT 0,
    status            INTEGER NOT NULL DEFAULT 0,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    photos            TEXT[] NOT NULL DEFAULT '{}',
    flag_reason       TEXT,
    verified          BOOLEAN NOT NULL DEFAULT FALSE,
    CHECK (
        (CASE WHEN business_id IS NOT NULL THEN 1 ELSE 0 END) +
        (CASE WHEN project_id  IS NOT NULL THEN 1 ELSE 0 END) = 1
    ),
    UNIQUE (business_id, author_id),
    UNIQUE (project_id, author_id)
);

CREATE TABLE review_votes (
    review_id UUID NOT NULL REFERENCES reviews (id) ON DELETE CASCADE,
    user_id   UUID NOT NULL REFERENCES users (id),
    helpful   BOOLEAN NOT NULL,
    PRIMARY KEY (review_id, user_id)
);

CREATE TABLE reports (
    id          UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    target_type TEXT NOT NULL CHECK (target_type IN ('review', 'business', 'project')),
    target_id   TEXT NOT NULL,
    reporter_id UUID NOT NULL REFERENCES users (id),
    reason      TEXT NOT NULL,
    status      INTEGER NOT NULL DEFAULT 0,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    resolved_by UUID REFERENCES users (id),
    resolved_at TIMESTAMPTZ
);

CREATE TABLE review_invites (
    id                 UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    business_id        UUID REFERENCES businesses (id) ON DELETE CASCADE,
    project_id         UUID REFERENCES projects (id) ON DELETE CASCADE,
    target_email       TEXT NOT NULL,
    token              TEXT NOT NULL UNIQUE,
    created_by         UUID NOT NULL REFERENCES users (id),
    status             INTEGER NOT NULL DEFAULT 0,
    redeemed_review_id UUID REFERENCES reviews (id),
    expires_at         TIMESTAMPTZ NOT NULL,
    created_at         TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CHECK ((business_id IS NOT NULL) <> (project_id IS NOT NULL))
);

CREATE TABLE claims (
    id          UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    business_id UUID NOT NULL REFERENCES businesses (id) ON DELETE CASCADE,
    user_id     UUID NOT NULL REFERENCES users (id),
    note        TEXT,
    status      INTEGER NOT NULL DEFAULT 0,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    resolved_by UUID REFERENCES users (id),
    resolved_at TIMESTAMPTZ
);

CREATE TABLE webhooks (
    id                UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    target_type       TEXT NOT NULL,
    target_id         TEXT NOT NULL,
    url               TEXT NOT NULL,
    secret            TEXT NOT NULL,
    events            TEXT[] NOT NULL DEFAULT '{}',
    created_by        UUID NOT NULL REFERENCES users (id),
    enabled           BOOLEAN NOT NULL DEFAULT TRUE,
    failure_count     INTEGER NOT NULL DEFAULT 0,
    last_triggered_at TIMESTAMPTZ,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at        TIMESTAMPTZ NOT NULL DEFAULT NOW()
);


CREATE INDEX idx_businesses_category_id ON businesses (category_id);
CREATE INDEX idx_businesses_status_created_at ON businesses (status, created_at DESC);
CREATE INDEX idx_businesses_status_rating ON businesses (status, avg_rating DESC, review_count DESC);
CREATE INDEX idx_businesses_status_review_count ON businesses (status, review_count DESC);
CREATE INDEX idx_businesses_status_category ON businesses (status, category_id);
CREATE INDEX idx_businesses_name_trgm ON businesses USING GIN (name gin_trgm_ops);
CREATE INDEX idx_businesses_search_vector ON businesses USING GIN (search_vector);
CREATE INDEX idx_businesses_featured ON businesses (featured) WHERE featured;

CREATE INDEX idx_projects_business_status ON projects (business_id, status);
CREATE INDEX idx_projects_status ON projects (status);
CREATE INDEX idx_projects_search_vector ON projects USING GIN (search_vector);

CREATE INDEX idx_moderation_actions_target ON moderation_actions (target_type, target_id);
CREATE INDEX idx_moderation_actions_action_time ON moderation_actions (action_time DESC);

CREATE INDEX idx_reviews_business_status_created ON reviews (business_id, status, created_at DESC);
CREATE INDEX idx_reviews_project_status_created ON reviews (project_id, status, created_at DESC);
CREATE INDEX idx_reviews_author_id ON reviews (author_id);

CREATE INDEX idx_review_votes_user_id ON review_votes (user_id);

CREATE INDEX idx_reports_status ON reports (status);

CREATE INDEX idx_claims_status ON claims (status);
CREATE INDEX idx_claims_business_id ON claims (business_id);

CREATE INDEX idx_review_invites_token ON review_invites (token);
CREATE INDEX idx_review_invites_business_id ON review_invites (business_id);
CREATE INDEX idx_review_invites_project_id ON review_invites (project_id);

CREATE INDEX idx_webhooks_target_enabled ON webhooks (target_type, target_id, enabled);
