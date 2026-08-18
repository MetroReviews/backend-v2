-- Metro backend-v2 seed data, mirroring cmd/seed/main.go
-- Safe to re-run: every INSERT upserts on the same fixed ids.
--
-- Run with:
--   psql "$POSTGRES_URL" -f scripts/seed.sql
-- or paste directly into a psql session.

BEGIN;

-- nonce/session_token are random strings in Go (crypto.RandString); here we
-- generate equivalent random text inline. They only need to be non-empty and
-- unique-ish for local dev, so exact charset doesn't matter.

-- ============================================================
-- users + discord_accounts
-- A Metro user (its own uuid) with a linked Discord account, mirroring
-- what identity.EnsureDiscordUser + routes/panel/callback.go produce on a
-- real login. userXId/userXDiscord are deliberately different ID spaces.
-- ============================================================
INSERT INTO users (id, username, is_staff) VALUES
    ('00000000-0000-0000-0000-0000000000f1', 'alpha_seed', FALSE),
    ('00000000-0000-0000-0000-0000000000f2', 'beta_seed', FALSE),
    ('00000000-0000-0000-0000-0000000000f3', 'extra_seed', FALSE),
    ('00000000-0000-0000-0000-0000000000f4', 'reviewer_seed', TRUE)
ON CONFLICT (id) DO UPDATE SET username = EXCLUDED.username, is_staff = EXCLUDED.is_staff;

INSERT INTO discord_accounts (discord_id, user_id, nonce, session_token, session_expires_at) VALUES
    (2100000000000000001, '00000000-0000-0000-0000-0000000000f1',
     substr(md5(random()::text) || md5(random()::text), 1, 20),
     substr(md5(random()::text) || md5(random()::text) || md5(random()::text), 1, 43), NOW() + interval '30 days'),
    (2100000000000000002, '00000000-0000-0000-0000-0000000000f2',
     substr(md5(random()::text) || md5(random()::text), 1, 20),
     substr(md5(random()::text) || md5(random()::text) || md5(random()::text), 1, 43), NOW() + interval '30 days'),
    (2100000000000000003, '00000000-0000-0000-0000-0000000000f3',
     substr(md5(random()::text) || md5(random()::text), 1, 20),
     substr(md5(random()::text) || md5(random()::text) || md5(random()::text), 1, 43), NOW() + interval '30 days'),
    (2200000000000000001, '00000000-0000-0000-0000-0000000000f4',
     substr(md5(random()::text) || md5(random()::text), 1, 20),
     substr(md5(random()::text) || md5(random()::text) || md5(random()::text), 1, 43), NOW() + interval '30 days')
ON CONFLICT (discord_id) DO UPDATE SET
    user_id = EXCLUDED.user_id, nonce = EXCLUDED.nonce,
    session_token = EXCLUDED.session_token, session_expires_at = EXCLUDED.session_expires_at;

-- ============================================================
-- categories
-- ============================================================
INSERT INTO categories (id, slug, name, description, icon) VALUES
    ('00000000-0000-0000-0000-0000000000c1', 'restaurants', 'Restaurants',
     'Places to eat, reviewed by diners.', 'https://example.com/icons/restaurants.png'),
    ('00000000-0000-0000-0000-0000000000c2', 'software', 'Software & SaaS',
     'Apps and services, reviewed by their users.', 'https://example.com/icons/software.png')
ON CONFLICT (id) DO UPDATE SET
    slug = EXCLUDED.slug, name = EXCLUDED.name,
    description = EXCLUDED.description, icon = EXCLUDED.icon;

-- ============================================================
-- businesses
-- Same review queue as bots (status 0=Pending 1=UnderReview 2=Approved
-- 3=Denied 4=Suspended). Only approved businesses are publicly visible/
-- reviewable, so taskflow (Pending) demos /queue with a business in it.
-- owner_id/submitted_by/reviewer are all Metro user ids (users.id), not
-- Discord ids.
-- ============================================================
INSERT INTO businesses (id, category_id, slug, name, description, submitted_by, owner_id, status, reviewer) VALUES
    ('00000000-0000-0000-0000-0000000000d1', '00000000-0000-0000-0000-0000000000c1',
     'the-copper-spoon', 'The Copper Spoon', 'A neighborhood bistro used as seed data.',
     '00000000-0000-0000-0000-0000000000f1', '00000000-0000-0000-0000-0000000000f1',
     2, '00000000-0000-0000-0000-0000000000f4'), -- Approved
    ('00000000-0000-0000-0000-0000000000d2', '00000000-0000-0000-0000-0000000000c2',
     'taskflow', 'TaskFlow', 'A project management SaaS used as seed data.',
     '00000000-0000-0000-0000-0000000000f2', NULL,
     0, NULL) -- Pending
ON CONFLICT (id) DO UPDATE SET
    category_id = EXCLUDED.category_id, slug = EXCLUDED.slug, name = EXCLUDED.name,
    description = EXCLUDED.description, submitted_by = EXCLUDED.submitted_by,
    owner_id = EXCLUDED.owner_id, status = EXCLUDED.status, reviewer = EXCLUDED.reviewer;

-- ============================================================
-- projects (portfolio/showcase items posted on a business). Same review
-- queue as businesses/bots (status 0=Pending 1=UnderReview 2=Approved
-- 3=Denied 4=Suspended). Patio Expansion stays Pending, to demo /queue
-- with a project in it. submitted_by/reviewer are Metro user ids.
-- ============================================================
INSERT INTO projects (id, business_id, title, description, submitted_by, status, reviewer) VALUES
    ('00000000-0000-0000-0000-0000000000a1', '00000000-0000-0000-0000-0000000000d1',
     'Kitchen Remodel', 'A full kitchen remodel used as seed data.',
     '00000000-0000-0000-0000-0000000000f1', 2, '00000000-0000-0000-0000-0000000000f4'), -- Approved
    ('00000000-0000-0000-0000-0000000000a2', '00000000-0000-0000-0000-0000000000d1',
     'Patio Expansion', 'An outdoor seating expansion used as seed data.',
     '00000000-0000-0000-0000-0000000000f1', 0, NULL) -- Pending
ON CONFLICT (id) DO UPDATE SET
    business_id = EXCLUDED.business_id, title = EXCLUDED.title,
    description = EXCLUDED.description, submitted_by = EXCLUDED.submitted_by,
    status = EXCLUDED.status, reviewer = EXCLUDED.reviewer;

-- ============================================================
-- bots
-- owner/extra_owners stay raw Discord ids (they describe the Discord bot,
-- not a Metro account); reviewer is a Metro user id, like businesses.reviewer.
-- ============================================================
INSERT INTO bots (
    bot_id, username, banner, description, long_description,
    website, support, donate, library, nsfw, prefix, tags,
    review_note, invite, state, owner, extra_owners,
    reviewer, invite_link
) VALUES
    (1100000000000000001, 'SeedBot Alpha', NULL,
     'A demo moderation bot used as seed data.',
     E'## SeedBot Alpha\n\nHandles auto-moderation, logging and warnings. This entry is seed data for local development.',
     'https://example.com/alpha', 'https://discord.gg/example-alpha', NULL, 'discord.js',
     FALSE, '!', ARRAY['moderation','logging'],
     NULL,
     'https://discord.com/oauth2/authorize?client_id=1100000000000000001&scope=bot',
     0, -- Pending
     2100000000000000001, ARRAY[]::BIGINT[],
     NULL, NULL),

    (1100000000000000002, 'SeedBot Beta', NULL,
     'A demo music bot used as seed data.',
     E'## SeedBot Beta\n\nStreams music from various sources. This entry is seed data for local development.',
     'https://example.com/beta', 'https://discord.gg/example-beta', NULL, 'discord.py',
     FALSE, 'b!', ARRAY['music'],
     NULL,
     'https://discord.com/oauth2/authorize?client_id=1100000000000000002&scope=bot',
     1, -- UnderReview
     2100000000000000002, ARRAY[]::BIGINT[],
     '00000000-0000-0000-0000-0000000000f4', NULL),

    (1100000000000000003, 'SeedBot Gamma', NULL,
     'A demo economy and utility bot used as seed data.',
     E'## SeedBot Gamma\n\nAdds an economy system alongside general utility commands. This entry is seed data for local development.',
     'https://example.com/gamma', 'https://discord.gg/example-gamma', NULL, 'serenity',
     FALSE, '/', ARRAY['utility','economy'],
     NULL,
     'https://discord.com/oauth2/authorize?client_id=1100000000000000003&scope=bot',
     2, -- Approved
     2100000000000000001, ARRAY[2100000000000000003],
     '00000000-0000-0000-0000-0000000000f4', NULL),

    (1100000000000000004, 'SeedBot Delta', NULL,
     'A demo NSFW bot used as seed data.',
     E'## SeedBot Delta\n\nDenied during review for missing a privacy policy. This entry is seed data for local development.',
     'https://example.com/delta', NULL, NULL, 'discord.js',
     TRUE, 'd!', ARRAY['nsfw'],
     'Missing privacy policy',
     'https://discord.com/oauth2/authorize?client_id=1100000000000000004&scope=bot',
     3, -- Denied
     2100000000000000002, ARRAY[]::BIGINT[],
     '00000000-0000-0000-0000-0000000000f4', NULL),

    (1100000000000000005, 'SeedBot Epsilon', NULL,
     'A second demo pending bot used as seed data.',
     E'## SeedBot Epsilon\n\nA freshly submitted bot awaiting review. This entry is seed data for local development.',
     'https://example.com/epsilon', 'https://discord.gg/example-epsilon', NULL, 'discord.js',
     FALSE, 'e.', ARRAY['leveling','economy'],
     NULL,
     'https://discord.com/oauth2/authorize?client_id=1100000000000000005&scope=bot',
     0, -- Pending
     2100000000000000003, ARRAY[]::BIGINT[],
     NULL, NULL)
ON CONFLICT (bot_id) DO UPDATE SET
    username = EXCLUDED.username,
    description = EXCLUDED.description,
    long_description = EXCLUDED.long_description,
    website = EXCLUDED.website,
    support = EXCLUDED.support,
    library = EXCLUDED.library,
    nsfw = EXCLUDED.nsfw,
    prefix = EXCLUDED.prefix,
    tags = EXCLUDED.tags,
    review_note = EXCLUDED.review_note,
    invite = EXCLUDED.invite,
    state = EXCLUDED.state,
    owner = EXCLUDED.owner,
    extra_owners = EXCLUDED.extra_owners,
    reviewer = EXCLUDED.reviewer;

-- ============================================================
-- moderation_actions (the shared bot/business/project review queue's audit
-- log). reviewer is a Metro user id.
-- ============================================================
INSERT INTO moderation_actions (id, target_type, target_id, action, reason, reviewer) VALUES
    ('00000000-0000-0000-0000-0000000000b1', 'bot', '1100000000000000002', 0, -- Claim
     'Claimed for review', '00000000-0000-0000-0000-0000000000f4'),
    ('00000000-0000-0000-0000-0000000000b2', 'bot', '1100000000000000003', 2, -- Approve
     'Meets all business requirements', '00000000-0000-0000-0000-0000000000f4'),
    ('00000000-0000-0000-0000-0000000000b3', 'bot', '1100000000000000004', 3, -- Deny
     'Missing privacy policy', '00000000-0000-0000-0000-0000000000f4'),
    ('00000000-0000-0000-0000-0000000000b4', 'project', '00000000-0000-0000-0000-0000000000a1', 2, -- Approve
     'Great before/after photos, approved', '00000000-0000-0000-0000-0000000000f4')
ON CONFLICT (id) DO UPDATE SET
    target_type = EXCLUDED.target_type,
    target_id = EXCLUDED.target_id,
    action = EXCLUDED.action,
    reason = EXCLUDED.reason,
    reviewer = EXCLUDED.reviewer;

-- ============================================================
-- reviews (only approved subjects are reviewable — taskflow and Patio
-- Expansion are still Pending above, so nothing reviews them yet).
-- author_id is a Metro user id.
-- ============================================================
INSERT INTO reviews (id, business_id, bot_id, project_id, author_id, rating, title, body) VALUES
    ('00000000-0000-0000-0000-0000000000e1', '00000000-0000-0000-0000-0000000000d1', NULL, NULL,
     '00000000-0000-0000-0000-0000000000f2', 5, 'Fantastic', 'Great food and service, seed review.'),
    ('00000000-0000-0000-0000-0000000000e2', '00000000-0000-0000-0000-0000000000d1', NULL, NULL,
     '00000000-0000-0000-0000-0000000000f3', 4, 'Pretty good', 'Enjoyed it, seed review.'),
    ('00000000-0000-0000-0000-0000000000e3', '00000000-0000-0000-0000-0000000000d1', NULL, NULL,
     '00000000-0000-0000-0000-0000000000f1', 3, 'A bit pricey', 'Good but a bit pricey for the portions, seed review.'),
    ('00000000-0000-0000-0000-0000000000e4', NULL, 1100000000000000003, NULL,
     '00000000-0000-0000-0000-0000000000f2', 5, 'Great utility bot', 'Reliable economy commands, seed review.'),
    ('00000000-0000-0000-0000-0000000000e5', NULL, NULL, '00000000-0000-0000-0000-0000000000a1',
     '00000000-0000-0000-0000-0000000000f3', 5, 'Beautiful work', 'The remodel turned out amazing, seed review.')
ON CONFLICT (id) DO UPDATE SET
    rating = EXCLUDED.rating, title = EXCLUDED.title, body = EXCLUDED.body;

-- Roll up ratings the same way the API does after a review write.
UPDATE businesses SET
    avg_rating = COALESCE((SELECT AVG(rating) FROM reviews WHERE business_id = businesses.id AND status = 0), 0),
    review_count = (SELECT COUNT(*) FROM reviews WHERE business_id = businesses.id AND status = 0)
WHERE id IN ('00000000-0000-0000-0000-0000000000d1', '00000000-0000-0000-0000-0000000000d2');

UPDATE bots SET
    avg_rating = COALESCE((SELECT AVG(rating) FROM reviews WHERE bot_id = bots.bot_id AND status = 0), 0),
    review_count = (SELECT COUNT(*) FROM reviews WHERE bot_id = bots.bot_id AND status = 0)
WHERE bot_id = 1100000000000000003;

UPDATE projects SET
    avg_rating = COALESCE((SELECT AVG(rating) FROM reviews WHERE project_id = projects.id AND status = 0), 0),
    review_count = (SELECT COUNT(*) FROM reviews WHERE project_id = projects.id AND status = 0)
WHERE id IN ('00000000-0000-0000-0000-0000000000a1', '00000000-0000-0000-0000-0000000000a2');

COMMIT;

-- Show what ended up seeded.
SELECT slug, name, avg_rating, review_count FROM businesses;
SELECT title, business_id, avg_rating, review_count FROM projects;
