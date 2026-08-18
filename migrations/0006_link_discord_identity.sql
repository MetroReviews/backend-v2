-- Decouple identity from Discord: a Metro user is now its own row with its
-- own id, and a Discord account links to it via discord_accounts instead
-- of the Discord snowflake being the user's primary key. This also lets a
-- future login method link into the same user without a schema change, and
-- lets bot/listing reviewers resolve to a real user row even when they've
-- only ever interacted through the Discord bot (see the identity package),
-- which is what 0005's FK-drop was really working around.
--
-- Fresh rebuild, same as 0003/0004: this is all dev/seed data, nothing here
-- has ever needed to be preserved across a schema change.

DROP TABLE IF EXISTS moderation_actions;
DROP TABLE IF EXISTS review_votes;
DROP TABLE IF EXISTS reviews;
DROP TABLE IF EXISTS reports;
DROP TABLE IF EXISTS claims;
DROP TABLE IF EXISTS listings;
DROP TABLE IF EXISTS bots;
DROP TABLE IF EXISTS users;

CREATE TABLE users (
    id         UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    username   TEXT,
    avatar     TEXT,
    bio        TEXT,
    is_staff   BOOLEAN NOT NULL DEFAULT FALSE,
    banned     BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- One row per linked external account. Discord is the only provider today,
-- but nothing else in the schema references Discord IDs directly for
-- identity purposes anymore — everything points at users(id) — so adding
-- e.g. a github_accounts table later needs no further changes elsewhere.
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

-- Metro's own Discord bot list. owner/extra_owners stay plain Discord IDs
-- deliberately: they describe who owns the *Discord bot*, which doesn't
-- require that person to have ever created a Metro account. reviewer is a
-- real user (identity.EnsureDiscordUser resolves the claiming staffer's
-- Discord ID to one before it's ever written here).
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
    state            INTEGER NOT NULL DEFAULT 0,
    avg_rating       NUMERIC(3,2) NOT NULL DEFAULT 0,
    review_count     INTEGER NOT NULL DEFAULT 0,
    added_at         TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE listings (
    id            UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    category_id   UUID NOT NULL REFERENCES categories (id),
    slug          TEXT NOT NULL UNIQUE,
    name          TEXT NOT NULL,
    description   TEXT,
    website       TEXT,
    logo          TEXT,
    banner        TEXT,
    address       TEXT,
    city          TEXT,
    country       TEXT,
    metadata      JSONB NOT NULL DEFAULT '{}',
    owner_id      UUID REFERENCES users (id),
    submitted_by  UUID NOT NULL REFERENCES users (id),
    reviewer      UUID REFERENCES users (id),
    status        INTEGER NOT NULL DEFAULT 0,
    avg_rating    NUMERIC(3,2) NOT NULL DEFAULT 0,
    review_count  INTEGER NOT NULL DEFAULT 0,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE reviews (
    id                UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    listing_id        UUID REFERENCES listings (id) ON DELETE CASCADE,
    bot_id            BIGINT REFERENCES bots (bot_id) ON DELETE CASCADE,
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
    CHECK ((listing_id IS NOT NULL) <> (bot_id IS NOT NULL)),
    UNIQUE (listing_id, author_id),
    UNIQUE (bot_id, author_id)
);

CREATE TABLE review_votes (
    review_id UUID NOT NULL REFERENCES reviews (id) ON DELETE CASCADE,
    user_id   UUID NOT NULL REFERENCES users (id),
    helpful   BOOLEAN NOT NULL,
    PRIMARY KEY (review_id, user_id)
);

CREATE TABLE reports (
    id          UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    target_type TEXT NOT NULL CHECK (target_type IN ('review', 'listing', 'bot')),
    target_id   TEXT NOT NULL,
    reporter_id UUID NOT NULL REFERENCES users (id),
    reason      TEXT NOT NULL,
    status      INTEGER NOT NULL DEFAULT 0,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    resolved_by UUID REFERENCES users (id),
    resolved_at TIMESTAMPTZ
);

CREATE TABLE claims (
    id          UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    listing_id  UUID NOT NULL REFERENCES listings (id) ON DELETE CASCADE,
    user_id     UUID NOT NULL REFERENCES users (id),
    note        TEXT,
    status      INTEGER NOT NULL DEFAULT 0,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    resolved_by UUID REFERENCES users (id),
    resolved_at TIMESTAMPTZ
);

CREATE TABLE moderation_actions (
    id          UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    target_type TEXT NOT NULL CHECK (target_type IN ('bot', 'listing')),
    target_id   TEXT NOT NULL,
    action      INTEGER NOT NULL,
    reason      TEXT NOT NULL,
    reviewer    UUID NOT NULL REFERENCES users (id),
    action_time TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_listings_category_id ON listings (category_id);
CREATE INDEX idx_listings_status ON listings (status);
CREATE INDEX idx_bots_state_bot_id ON bots (state, bot_id);
CREATE INDEX idx_moderation_actions_target ON moderation_actions (target_type, target_id);
CREATE INDEX idx_moderation_actions_action_time ON moderation_actions (action_time DESC);
CREATE INDEX idx_reviews_listing_id ON reviews (listing_id);
CREATE INDEX idx_reviews_bot_id ON reviews (bot_id);
CREATE INDEX idx_reviews_author_id ON reviews (author_id);
CREATE INDEX idx_review_votes_user_id ON review_votes (user_id);
CREATE INDEX idx_reports_status ON reports (status);
CREATE INDEX idx_claims_status ON claims (status);
CREATE INDEX idx_claims_listing_id ON claims (listing_id);
