-- Metro's permissions system: named roles (see the perms package for the
-- catalog of permissions a role can grant) that get assigned to users and
-- gate access to the staff panel and its actions.
--
-- A role can optionally link to a Discord server role via discord_role_id.
-- When it does, the bot keeps user_roles in sync with that Discord role's
-- membership (see the roles package's SyncMember/SyncGuild) — assign the
-- Discord role in the server and the panel permissions follow automatically.
-- A role with no linked Discord role is assigned by hand from the panel
-- instead.

CREATE TABLE roles (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name            TEXT NOT NULL UNIQUE,
    discord_role_id BIGINT UNIQUE, -- linked Discord role; NULL for panel-only roles
    permissions     TEXT[] NOT NULL DEFAULT '{}', -- permission slugs this role grants; "*" grants everything
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Who holds which role. For a Discord-linked role this is maintained by
-- the sync rather than edited directly; for a panel-only role it's the
-- only way a user ever gets it.
CREATE TABLE user_roles (
    user_id    UUID NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    role_id    UUID NOT NULL REFERENCES roles (id) ON DELETE CASCADE,
    granted_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (user_id, role_id)
);
CREATE INDEX idx_user_roles_role_id ON user_roles (role_id);
