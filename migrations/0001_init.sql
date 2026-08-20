
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

CREATE TABLE IF NOT EXISTS bot_list (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    state           INTEGER NOT NULL DEFAULT 0,
    name            TEXT NOT NULL,
    description     TEXT,
    icon            TEXT,
    claim_bot_api   TEXT NOT NULL,
    unclaim_bot_api TEXT NOT NULL,
    approve_bot_api TEXT NOT NULL,
    deny_bot_api    TEXT NOT NULL,
    domain          TEXT NOT NULL,
    secret_key      TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS bot_queue (
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
    added_at         TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    state            INTEGER NOT NULL DEFAULT 0,
    list_source      UUID NOT NULL REFERENCES bot_list (id) ON DELETE CASCADE ON UPDATE CASCADE,
    owner            BIGINT NOT NULL,
    extra_owners     BIGINT[] NOT NULL DEFAULT '{}',
    reviewer         BIGINT,
    invite_link      TEXT,
    cross_add        BOOLEAN NOT NULL DEFAULT TRUE
);

CREATE TABLE IF NOT EXISTS bot_action (
    id          UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    bot_id      BIGINT NOT NULL,
    action      INTEGER NOT NULL,
    reason      TEXT NOT NULL,
    reviewer    TEXT NOT NULL,
    action_time TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    list_source UUID NOT NULL REFERENCES bot_list (id) ON DELETE CASCADE ON UPDATE CASCADE
);

CREATE TABLE IF NOT EXISTS users (
    user_id BIGINT PRIMARY KEY,
    nonce   TEXT NOT NULL
);
