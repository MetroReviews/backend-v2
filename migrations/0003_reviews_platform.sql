-- Pivot: Metro becomes a Yelp/Trustpilot-style review site for any listing,
-- plus its own standalone Discord bot list (no longer a federation hub).
--
-- Drops the list-federation tables (bot_list, bot_action) and the old
-- bot_queue in favor of a self-contained `bots` table, and adds the generic
-- review platform: categories, listings, reviews, review_votes, reports,
-- claims. No data is preserved from the old tables (fresh start, agreed
-- with product direction change).

DROP TABLE IF EXISTS bot_action;
DROP TABLE IF EXISTS bot_queue;
DROP TABLE IF EXISTS bot_list;

-- users: was (user_id, nonce) only, used purely as an OAuth nonce store.
-- Becomes a real account: profile fields, staff/ban flags, and a long-lived
-- session token (see api.AuthUser) alongside the existing short-lived nonce
-- (still used by the staff panel ticket flow in routes/panel).
ALTER TABLE users
    ADD COLUMN IF NOT EXISTS username           TEXT,
    ADD COLUMN IF NOT EXISTS avatar             TEXT,
    ADD COLUMN IF NOT EXISTS bio                TEXT,
    ADD COLUMN IF NOT EXISTS is_staff            BOOLEAN NOT NULL DEFAULT FALSE,
    ADD COLUMN IF NOT EXISTS banned             BOOLEAN NOT NULL DEFAULT FALSE,
    ADD COLUMN IF NOT EXISTS session_token      TEXT,
    ADD COLUMN IF NOT EXISTS session_expires_at TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS created_at         TIMESTAMPTZ NOT NULL DEFAULT NOW();

CREATE TABLE categories (
    id          UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    slug        TEXT NOT NULL UNIQUE,
    name        TEXT NOT NULL,
    description TEXT,
    icon        TEXT
);

-- Enum values (stored as integers):
--   State (shared by bots.state and listings.status — the review queue):
--     PENDING=0, UNDER_REVIEW=1, APPROVED=2, DENIED=3, SUSPENDED=4
--   ClaimStatus (ownership claims on a listing, unrelated to the review
--     queue above): PENDING=0, APPROVED=1, DENIED=2
--   ReviewStatus:  PUBLISHED=0, FLAGGED=1, REMOVED=2
--   ReportStatus:  OPEN=0, RESOLVED=1, DISMISSED=2

CREATE TABLE listings (
    id            UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    category_id   UUID NOT NULL REFERENCES categories (id),
    slug          TEXT NOT NULL UNIQUE,
    name          TEXT NOT NULL,
    description   TEXT,
    website       TEXT,
    logo          TEXT,
    banner        TEXT,
    address       TEXT, -- nullable: not every listing is a physical place
    city          TEXT,
    country       TEXT,
    metadata      JSONB NOT NULL DEFAULT '{}', -- category-specific extra fields
    owner_id      BIGINT REFERENCES users (user_id), -- verified owner, set once an ownership claim is approved
    submitted_by  BIGINT NOT NULL REFERENCES users (user_id),
    reviewer      BIGINT REFERENCES users (user_id), -- staff member who claimed it for review
    status        INTEGER NOT NULL DEFAULT 0, -- State.PENDING — goes through the same review queue as bots
    avg_rating    NUMERIC(3,2) NOT NULL DEFAULT 0,
    review_count  INTEGER NOT NULL DEFAULT 0,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Metro's own Discord bot list. Same shape as the old bot_queue minus the
-- list-federation columns (list_source, cross_add), plus rating rollups.
-- owner/extra_owners are not FKed to users: a bot's owner is just a Discord
-- ID supplied at submission time and need never have logged in to Metro.
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
    reviewer         BIGINT REFERENCES users (user_id), -- staff member who claimed/actioned it
    state            INTEGER NOT NULL DEFAULT 0, -- State.PENDING (types.State, unchanged)
    avg_rating       NUMERIC(3,2) NOT NULL DEFAULT 0,
    review_count     INTEGER NOT NULL DEFAULT 0,
    added_at         TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Review queue action audit log, shared by bots and listings (the same
-- claim/unclaim/approve/deny pipeline covers both — see the review
-- package). Was bot_action, generalized past just bots and minus
-- list_source (there's no longer any list to attribute the action to).
CREATE TABLE moderation_actions (
    id          UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    target_type TEXT NOT NULL CHECK (target_type IN ('bot', 'listing')),
    target_id   TEXT NOT NULL, -- a bot_id or a listing UUID, stored as text since the target type varies
    action      INTEGER NOT NULL, -- Action (types.Action, unchanged)
    reason      TEXT NOT NULL,
    reviewer    BIGINT NOT NULL REFERENCES users (user_id),
    action_time TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE reviews (
    id                UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    listing_id        UUID REFERENCES listings (id) ON DELETE CASCADE,
    bot_id            BIGINT REFERENCES bots (bot_id) ON DELETE CASCADE,
    author_id         BIGINT NOT NULL REFERENCES users (user_id),
    rating            SMALLINT NOT NULL CHECK (rating BETWEEN 1 AND 5),
    title             TEXT,
    body              TEXT NOT NULL,
    owner_response    TEXT,
    owner_response_at TIMESTAMPTZ,
    helpful_count     INTEGER NOT NULL DEFAULT 0,
    status            INTEGER NOT NULL DEFAULT 0, -- ReviewStatus.PUBLISHED
    created_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CHECK ((listing_id IS NOT NULL) <> (bot_id IS NOT NULL)), -- exactly one subject
    UNIQUE (listing_id, author_id), -- one review per user per listing
    UNIQUE (bot_id, author_id)      -- one review per user per bot
);

CREATE TABLE review_votes (
    review_id UUID NOT NULL REFERENCES reviews (id) ON DELETE CASCADE,
    user_id   BIGINT NOT NULL REFERENCES users (user_id),
    helpful   BOOLEAN NOT NULL,
    PRIMARY KEY (review_id, user_id)
);

CREATE TABLE reports (
    id          UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    target_type TEXT NOT NULL CHECK (target_type IN ('review', 'listing', 'bot')),
    target_id   TEXT NOT NULL, -- a UUID or a bot_id, stored as text since the target type varies
    reporter_id BIGINT NOT NULL REFERENCES users (user_id),
    reason      TEXT NOT NULL,
    status      INTEGER NOT NULL DEFAULT 0, -- ReportStatus.OPEN
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    resolved_by BIGINT REFERENCES users (user_id),
    resolved_at TIMESTAMPTZ
);

-- Ownership claim requests on a listing.
CREATE TABLE claims (
    id          UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    listing_id  UUID NOT NULL REFERENCES listings (id) ON DELETE CASCADE,
    user_id     BIGINT NOT NULL REFERENCES users (user_id),
    note        TEXT,
    status      INTEGER NOT NULL DEFAULT 0, -- ClaimStatus.PENDING
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    resolved_by BIGINT REFERENCES users (user_id),
    resolved_at TIMESTAMPTZ
);
