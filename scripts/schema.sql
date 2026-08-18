-- Metro backend-v2 schema — consolidated snapshot of migrations/0001..0012.
-- Postgres. This is a convenience snapshot for spinning up a fresh database
-- in one shot (e.g. local dev, CI); it is not itself a migration and is not
-- read by the app. The migrations/ directory remains the source of truth —
-- if you change the schema, add a new migrations/NNNN_*.sql file and mirror
-- the result here rather than editing history.
--
-- Run with:
--   psql "$POSTGRES_URL" -f scripts/schema.sql

CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS pg_trgm;

-- ============================================================
-- Identity
-- A Metro user is its own row with its own id; external accounts (Discord
-- today, potentially others later) link to it via a per-provider table
-- instead of the provider's id being the user's primary key.
-- ============================================================

CREATE TABLE users (
    id         UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    username   TEXT,
    avatar     TEXT,
    bio        TEXT,
    is_staff   BOOLEAN NOT NULL DEFAULT FALSE,
    banned     BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- One row per linked external account. Discord is the only provider today;
-- everything else in the schema references users(id), never a Discord ID,
-- for identity purposes, so a future provider table needs no other changes.
CREATE TABLE discord_accounts (
    discord_id         BIGINT PRIMARY KEY,
    user_id            UUID NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    nonce              TEXT,          -- short-lived OAuth ticket nonce (routes/panel)
    session_token      TEXT,          -- long-lived API bearer session (api.AuthUser)
    session_expires_at TIMESTAMPTZ,
    linked_at          TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_discord_accounts_user_id ON discord_accounts (user_id);
CREATE INDEX idx_discord_accounts_session_token ON discord_accounts (session_token);

-- ============================================================
-- Permissions
-- Named roles assigned to users, gating access to the staff panel and its
-- actions. A role can optionally link to a Discord server role via
-- discord_role_id; when it does, the bot keeps user_roles in sync with that
-- Discord role's membership. A role with no linked Discord role is assigned
-- by hand from the panel instead.
-- ============================================================

CREATE TABLE roles (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name            TEXT NOT NULL UNIQUE,
    discord_role_id BIGINT UNIQUE, -- linked Discord role; NULL for panel-only roles
    permissions     TEXT[] NOT NULL DEFAULT '{}', -- permission slugs this role grants; "*" grants everything
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Who holds which role. For a Discord-linked role this is maintained by the
-- sync rather than edited directly; for a panel-only role it's the only way
-- a user ever gets it.
CREATE TABLE user_roles (
    user_id    UUID NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    role_id    UUID NOT NULL REFERENCES roles (id) ON DELETE CASCADE,
    granted_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (user_id, role_id)
);
CREATE INDEX idx_user_roles_role_id ON user_roles (role_id);

-- ============================================================
-- Review-platform catalog
-- ============================================================

CREATE TABLE categories (
    id          UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    slug        TEXT NOT NULL UNIQUE,
    name        TEXT NOT NULL,
    description TEXT,
    icon        TEXT
);

-- Enum values (stored as integers, matching the Go types package):
--   State (shared by bots.state and businesses.status — the review queue):
--     PENDING=0, UNDER_REVIEW=1, APPROVED=2, DENIED=3, SUSPENDED=4
--   ClaimStatus (ownership claims on a business, unrelated to the review
--     queue above): PENDING=0, APPROVED=1, DENIED=2
--   ReviewStatus:  PUBLISHED=0, FLAGGED=1, REMOVED=2
--   ReportStatus:  OPEN=0, RESOLVED=1, DISMISSED=2
--   Action (moderation_actions.action): CLAIM=0, UNCLAIM=1, APPROVE=2, DENY=3

-- Any reviewable service or business (formerly "listing" — see 0009 for the
-- rename). Goes through the same claim/approve/deny review queue as a bot.
CREATE TABLE businesses (
    id            UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    category_id   UUID NOT NULL REFERENCES categories (id),
    slug          TEXT NOT NULL UNIQUE,
    name          TEXT NOT NULL,
    description   TEXT,
    website       TEXT,
    logo          TEXT,
    banner        TEXT,
    address       TEXT, -- nullable: not every business is a physical place
    city          TEXT,
    country       TEXT,
    metadata      JSONB NOT NULL DEFAULT '{}', -- category-specific extra fields
    owner_id      UUID REFERENCES users (id), -- verified owner, set once an ownership claim is approved
    submitted_by  UUID NOT NULL REFERENCES users (id),
    reviewer      UUID REFERENCES users (id), -- staff member who claimed it for review
    status        INTEGER NOT NULL DEFAULT 0, -- State.PENDING
    avg_rating    NUMERIC(3,2) NOT NULL DEFAULT 0,
    review_count  INTEGER NOT NULL DEFAULT 0,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Metro's own Discord bot list. owner/extra_owners stay plain Discord IDs
-- deliberately: they describe who owns the *Discord bot*, which doesn't
-- require that person to have ever created a Metro account. reviewer is a
-- real user (identity.EnsureDiscordUser resolves the claiming staffer's
-- Discord ID to one before it's ever written here) — not FKed, since a
-- Discord slash-command reviewer is authorized by holding the
-- Reviewer/Sudo guild role, not by having necessarily logged into the panel.
CREATE TABLE bots (
    bot_id           BIGINT PRIMARY KEY,
    username         TEXT NOT NULL,
    banner           TEXT,
    description      TEXT NOT NULL,
    long_description TEXT NOT NULL,
    website          TEXT,
    support          TEXT,
    donate           TEXT,
    library          TEXT,
    nsfw             BOOLEAN NOT NULL DEFAULT FALSE,
    prefix           TEXT,
    tags             TEXT[] NOT NULL DEFAULT '{}',
    review_note      TEXT,
    invite           TEXT,
    invite_link      TEXT,
    owner            BIGINT NOT NULL,
    extra_owners     BIGINT[] NOT NULL DEFAULT '{}',
    reviewer         UUID REFERENCES users (id),
    state            INTEGER NOT NULL DEFAULT 0, -- State.PENDING
    avg_rating       NUMERIC(3,2) NOT NULL DEFAULT 0,
    review_count     INTEGER NOT NULL DEFAULT 0,
    added_at         TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Portfolio/showcase item a business posts — past work shown on its
-- profile. Posting one goes through the same claim/approve/deny review
-- queue as a new business or bot, and (0011) is itself reviewable.
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
    status       INTEGER NOT NULL DEFAULT 0, -- State.PENDING
    avg_rating   NUMERIC(3,2) NOT NULL DEFAULT 0,
    review_count INTEGER NOT NULL DEFAULT 0,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Review queue action audit log, shared by bots, businesses and projects
-- (the same claim/unclaim/approve/deny pipeline covers all three — see the
-- review package).
CREATE TABLE moderation_actions (
    id          UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    target_type TEXT NOT NULL CHECK (target_type IN ('bot', 'business', 'project')),
    target_id   TEXT NOT NULL, -- a bot_id or a business/project UUID, stored as text since the target type varies
    action      INTEGER NOT NULL, -- Action
    reason      TEXT NOT NULL,
    reviewer    UUID NOT NULL REFERENCES users (id),
    action_time TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE reviews (
    id                UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    business_id       UUID REFERENCES businesses (id) ON DELETE CASCADE,
    bot_id            BIGINT REFERENCES bots (bot_id) ON DELETE CASCADE,
    project_id        UUID REFERENCES projects (id) ON DELETE CASCADE,
    author_id         UUID NOT NULL REFERENCES users (id),
    rating            SMALLINT NOT NULL CHECK (rating BETWEEN 1 AND 5),
    title             TEXT,
    body              TEXT NOT NULL,
    owner_response    TEXT,
    owner_response_at TIMESTAMPTZ,
    helpful_count     INTEGER NOT NULL DEFAULT 0,
    status            INTEGER NOT NULL DEFAULT 0, -- ReviewStatus.PUBLISHED
    created_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    -- exactly one subject
    CHECK (
        (CASE WHEN business_id IS NOT NULL THEN 1 ELSE 0 END) +
        (CASE WHEN bot_id      IS NOT NULL THEN 1 ELSE 0 END) +
        (CASE WHEN project_id  IS NOT NULL THEN 1 ELSE 0 END) = 1
    ),
    UNIQUE (business_id, author_id), -- one review per user per business
    UNIQUE (bot_id, author_id),      -- one review per user per bot
    UNIQUE (project_id, author_id)   -- one review per user per project
);

CREATE TABLE review_votes (
    review_id UUID NOT NULL REFERENCES reviews (id) ON DELETE CASCADE,
    user_id   UUID NOT NULL REFERENCES users (id),
    helpful   BOOLEAN NOT NULL,
    PRIMARY KEY (review_id, user_id)
);

CREATE TABLE reports (
    id          UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    target_type TEXT NOT NULL CHECK (target_type IN ('review', 'business', 'bot')),
    target_id   TEXT NOT NULL, -- a UUID or a bot_id, stored as text since the target type varies
    reporter_id UUID NOT NULL REFERENCES users (id),
    reason      TEXT NOT NULL,
    status      INTEGER NOT NULL DEFAULT 0, -- ReportStatus.OPEN
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    resolved_by UUID REFERENCES users (id),
    resolved_at TIMESTAMPTZ
);

-- Ownership claim requests on a business.
CREATE TABLE claims (
    id          UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    business_id UUID NOT NULL REFERENCES businesses (id) ON DELETE CASCADE,
    user_id     UUID NOT NULL REFERENCES users (id),
    note        TEXT,
    status      INTEGER NOT NULL DEFAULT 0, -- ClaimStatus.PENDING
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    resolved_by UUID REFERENCES users (id),
    resolved_at TIMESTAMPTZ
);

-- Outbound event notifications for any reviewable subject — currently bots,
-- businesses and projects, but deliberately generic (target_type/target_id
-- as free text, same polymorphic shape moderation_actions/reports use) so a
-- future subject type needs no schema change here to start firing/receiving
-- events.
--
-- events is the subscription filter: an empty array means "everything",
-- otherwise only the named events (see the webhooks package's Catalog) are
-- delivered. Not CHECK-constrained against a fixed list on purpose — new
-- event names can ship without a migration.
CREATE TABLE webhooks (
    id                UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    target_type       TEXT NOT NULL,
    target_id         TEXT NOT NULL,
    url               TEXT NOT NULL,
    secret            TEXT NOT NULL, -- HMAC-SHA256 signing secret; never returned by a GET, only on creation/rotation
    events            TEXT[] NOT NULL DEFAULT '{}',
    created_by        UUID NOT NULL REFERENCES users (id),
    enabled           BOOLEAN NOT NULL DEFAULT TRUE,
    failure_count     INTEGER NOT NULL DEFAULT 0, -- consecutive delivery failures; auto-disabled past a threshold (see webhooks.recordFailure)
    last_triggered_at TIMESTAMPTZ,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at        TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- ============================================================
-- Indexes
-- ============================================================

CREATE INDEX idx_businesses_category_id ON businesses (category_id);
CREATE INDEX idx_businesses_status_created_at ON businesses (status, created_at DESC);
CREATE INDEX idx_businesses_status_rating ON businesses (status, avg_rating DESC, review_count DESC);
CREATE INDEX idx_businesses_status_review_count ON businesses (status, review_count DESC);
CREATE INDEX idx_businesses_status_category ON businesses (status, category_id);
-- name search: ILIKE '%q%' has a leading wildcard, so no btree index
-- (single- or multi-column) can ever help it — only a trigram index can.
CREATE INDEX idx_businesses_name_trgm ON businesses USING GIN (name gin_trgm_ops);

CREATE INDEX idx_bots_state_bot_id ON bots (state, bot_id);

CREATE INDEX idx_projects_business_status ON projects (business_id, status);
CREATE INDEX idx_projects_status ON projects (status);

CREATE INDEX idx_moderation_actions_target ON moderation_actions (target_type, target_id);
CREATE INDEX idx_moderation_actions_action_time ON moderation_actions (action_time DESC);

CREATE INDEX idx_reviews_business_status_created ON reviews (business_id, status, created_at DESC);
CREATE INDEX idx_reviews_bot_status_created ON reviews (bot_id, status, created_at DESC);
CREATE INDEX idx_reviews_project_status_created ON reviews (project_id, status, created_at DESC);
CREATE INDEX idx_reviews_author_id ON reviews (author_id);

CREATE INDEX idx_review_votes_user_id ON review_votes (user_id);

CREATE INDEX idx_reports_status ON reports (status);

CREATE INDEX idx_claims_status ON claims (status);
CREATE INDEX idx_claims_business_id ON claims (business_id);

-- Every dispatch looks up "enabled webhooks for this exact target" — the
-- one query pattern this table serves.
CREATE INDEX idx_webhooks_target_enabled ON webhooks (target_type, target_id, enabled);
