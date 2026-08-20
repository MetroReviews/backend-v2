
DROP TABLE IF EXISTS bot_action;
DROP TABLE IF EXISTS bot_queue;
DROP TABLE IF EXISTS bot_list;

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
    owner_id      BIGINT REFERENCES users (user_id),
    submitted_by  BIGINT NOT NULL REFERENCES users (user_id),
    reviewer      BIGINT REFERENCES users (user_id),
    status        INTEGER NOT NULL DEFAULT 0,
    avg_rating    NUMERIC(3,2) NOT NULL DEFAULT 0,
    review_count  INTEGER NOT NULL DEFAULT 0,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

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
    reviewer         BIGINT REFERENCES users (user_id),
    state            INTEGER NOT NULL DEFAULT 0,
    avg_rating       NUMERIC(3,2) NOT NULL DEFAULT 0,
    review_count     INTEGER NOT NULL DEFAULT 0,
    added_at         TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE moderation_actions (
    id          UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    target_type TEXT NOT NULL CHECK (target_type IN ('bot', 'listing')),
    target_id   TEXT NOT NULL,
    action      INTEGER NOT NULL,
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
    status            INTEGER NOT NULL DEFAULT 0,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CHECK ((listing_id IS NOT NULL) <> (bot_id IS NOT NULL)),
    UNIQUE (listing_id, author_id),
    UNIQUE (bot_id, author_id)
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
    target_id   TEXT NOT NULL,
    reporter_id BIGINT NOT NULL REFERENCES users (user_id),
    reason      TEXT NOT NULL,
    status      INTEGER NOT NULL DEFAULT 0,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    resolved_by BIGINT REFERENCES users (user_id),
    resolved_at TIMESTAMPTZ
);

CREATE TABLE claims (
    id          UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    listing_id  UUID NOT NULL REFERENCES listings (id) ON DELETE CASCADE,
    user_id     BIGINT NOT NULL REFERENCES users (user_id),
    note        TEXT,
    status      INTEGER NOT NULL DEFAULT 0,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    resolved_by BIGINT REFERENCES users (user_id),
    resolved_at TIMESTAMPTZ
);
