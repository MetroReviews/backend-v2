
BEGIN;

CREATE EXTENSION IF NOT EXISTS pgcrypto;

INSERT INTO users (id, username, is_staff) VALUES
    ('00000000-0000-0000-0000-0000000000f1', 'alpha_seed', FALSE),
    ('00000000-0000-0000-0000-0000000000f2', 'beta_seed', FALSE),
    ('00000000-0000-0000-0000-0000000000f3', 'extra_seed', FALSE),
    ('00000000-0000-0000-0000-0000000000f4', 'reviewer_seed', TRUE),
    ('00000000-0000-0000-0000-0000000000f5', 'local_seed', FALSE)
ON CONFLICT (id) DO UPDATE SET username = EXCLUDED.username, is_staff = EXCLUDED.is_staff;

INSERT INTO discord_accounts (discord_id, user_id, nonce) VALUES
    (2100000000000000001, '00000000-0000-0000-0000-0000000000f1',
     substr(md5(random()::text) || md5(random()::text), 1, 20)),
    (2100000000000000002, '00000000-0000-0000-0000-0000000000f2',
     substr(md5(random()::text) || md5(random()::text), 1, 20)),
    (2100000000000000003, '00000000-0000-0000-0000-0000000000f3',
     substr(md5(random()::text) || md5(random()::text), 1, 20)),
    (2200000000000000001, '00000000-0000-0000-0000-0000000000f4',
     substr(md5(random()::text) || md5(random()::text), 1, 20))
ON CONFLICT (discord_id) DO UPDATE SET
    user_id = EXCLUDED.user_id, nonce = EXCLUDED.nonce;

INSERT INTO local_accounts (user_id, email, password_hash) VALUES
    ('00000000-0000-0000-0000-0000000000f1', 'alpha_seed@example.com', crypt('password123', gen_salt('bf'))),
    ('00000000-0000-0000-0000-0000000000f5', 'local_seed@example.com', crypt('password123', gen_salt('bf')))
ON CONFLICT (user_id) DO UPDATE SET
    email = EXCLUDED.email, password_hash = EXCLUDED.password_hash, updated_at = NOW();

INSERT INTO categories (id, slug, name, description, icon) VALUES
    ('00000000-0000-0000-0000-0000000000c1', 'restaurants', 'Restaurants',
     'Places to eat, reviewed by diners.', 'https://example.com/icons/restaurants.png'),
    ('00000000-0000-0000-0000-0000000000c2', 'software', 'Software & SaaS',
     'Apps and services, reviewed by their users.', 'https://example.com/icons/software.png')
ON CONFLICT (id) DO UPDATE SET
    slug = EXCLUDED.slug, name = EXCLUDED.name,
    description = EXCLUDED.description, icon = EXCLUDED.icon;

INSERT INTO businesses (
    id, category_id, slug, name, description, submitted_by, owner_id, status, reviewer,
    latitude, longitude, gallery, featured, view_count
) VALUES
    ('00000000-0000-0000-0000-0000000000d1', '00000000-0000-0000-0000-0000000000c1',
     'the-copper-spoon', 'The Copper Spoon', 'A neighborhood bistro used as seed data.',
     '00000000-0000-0000-0000-0000000000f1', '00000000-0000-0000-0000-0000000000f1',
     2, '00000000-0000-0000-0000-0000000000f4',
     40.7308, -73.9973, ARRAY['https://example.com/gallery/copper-spoon-1.jpg'], TRUE, 42),
    ('00000000-0000-0000-0000-0000000000d2', '00000000-0000-0000-0000-0000000000c2',
     'taskflow', 'TaskFlow', 'A project management SaaS used as seed data.',
     '00000000-0000-0000-0000-0000000000f2', NULL,
     0, NULL,
     NULL, NULL, '{}', FALSE, 0)
ON CONFLICT (id) DO UPDATE SET
    category_id = EXCLUDED.category_id, slug = EXCLUDED.slug, name = EXCLUDED.name,
    description = EXCLUDED.description, submitted_by = EXCLUDED.submitted_by,
    owner_id = EXCLUDED.owner_id, status = EXCLUDED.status, reviewer = EXCLUDED.reviewer,
    latitude = EXCLUDED.latitude, longitude = EXCLUDED.longitude, gallery = EXCLUDED.gallery,
    featured = EXCLUDED.featured, view_count = EXCLUDED.view_count;

INSERT INTO projects (id, business_id, title, description, submitted_by, status, reviewer) VALUES
    ('00000000-0000-0000-0000-0000000000a1', '00000000-0000-0000-0000-0000000000d1',
     'Kitchen Remodel', 'A full kitchen remodel used as seed data.',
     '00000000-0000-0000-0000-0000000000f1', 2, '00000000-0000-0000-0000-0000000000f4'),
    ('00000000-0000-0000-0000-0000000000a2', '00000000-0000-0000-0000-0000000000d1',
     'Patio Expansion', 'An outdoor seating expansion used as seed data.',
     '00000000-0000-0000-0000-0000000000f1', 0, NULL)
ON CONFLICT (id) DO UPDATE SET
    business_id = EXCLUDED.business_id, title = EXCLUDED.title,
    description = EXCLUDED.description, submitted_by = EXCLUDED.submitted_by,
    status = EXCLUDED.status, reviewer = EXCLUDED.reviewer;

INSERT INTO moderation_actions (id, target_type, target_id, action, reason, reviewer) VALUES
    ('00000000-0000-0000-0000-0000000000b4', 'project', '00000000-0000-0000-0000-0000000000a1', 2,
     'Great before/after photos, approved', '00000000-0000-0000-0000-0000000000f4')
ON CONFLICT (id) DO UPDATE SET
    target_type = EXCLUDED.target_type,
    target_id = EXCLUDED.target_id,
    action = EXCLUDED.action,
    reason = EXCLUDED.reason,
    reviewer = EXCLUDED.reviewer;

INSERT INTO reviews (id, business_id, project_id, author_id, rating, title, body, photos, verified) VALUES
    ('00000000-0000-0000-0000-0000000000e1', '00000000-0000-0000-0000-0000000000d1', NULL,
     '00000000-0000-0000-0000-0000000000f2', 5, 'Fantastic', 'Great food and service, seed review.', '{}', FALSE),
    ('00000000-0000-0000-0000-0000000000e2', '00000000-0000-0000-0000-0000000000d1', NULL,
     '00000000-0000-0000-0000-0000000000f3', 4, 'Pretty good', 'Enjoyed it, seed review.',
     ARRAY['https://example.com/reviews/e2-photo-1.jpg'], TRUE),
    ('00000000-0000-0000-0000-0000000000e3', '00000000-0000-0000-0000-0000000000d1', NULL,
     '00000000-0000-0000-0000-0000000000f1', 3, 'A bit pricey', 'Good but a bit pricey for the portions, seed review.', '{}', FALSE),
    ('00000000-0000-0000-0000-0000000000e5', NULL, '00000000-0000-0000-0000-0000000000a1',
     '00000000-0000-0000-0000-0000000000f3', 5, 'Beautiful work', 'The remodel turned out amazing, seed review.', '{}', FALSE)
ON CONFLICT (id) DO UPDATE SET
    rating = EXCLUDED.rating, title = EXCLUDED.title, body = EXCLUDED.body,
    photos = EXCLUDED.photos, verified = EXCLUDED.verified;

INSERT INTO review_invites (id, business_id, target_email, token, created_by, status, redeemed_review_id, expires_at) VALUES
    ('00000000-0000-0000-0000-0000000000d3', '00000000-0000-0000-0000-0000000000d1',
     'regular-diner@example.com', 'seed-invite-redeemed-alpha', '00000000-0000-0000-0000-0000000000f4',
     1, '00000000-0000-0000-0000-0000000000e2', NOW() + INTERVAL '14 days'),
    ('00000000-0000-0000-0000-0000000000d4', '00000000-0000-0000-0000-0000000000d1',
     'new-diner@example.com', 'seed-invite-pending-alpha', '00000000-0000-0000-0000-0000000000f4',
     0, NULL, NOW() + INTERVAL '14 days')
ON CONFLICT (id) DO UPDATE SET
    target_email = EXCLUDED.target_email, token = EXCLUDED.token,
    status = EXCLUDED.status, redeemed_review_id = EXCLUDED.redeemed_review_id,
    expires_at = EXCLUDED.expires_at;

UPDATE businesses SET
    avg_rating = COALESCE((SELECT AVG(rating) FROM reviews WHERE business_id = businesses.id AND status = 0), 0),
    review_count = (SELECT COUNT(*) FROM reviews WHERE business_id = businesses.id AND status = 0)
WHERE id IN ('00000000-0000-0000-0000-0000000000d1', '00000000-0000-0000-0000-0000000000d2');

UPDATE projects SET
    avg_rating = COALESCE((SELECT AVG(rating) FROM reviews WHERE project_id = projects.id AND status = 0), 0),
    review_count = (SELECT COUNT(*) FROM reviews WHERE project_id = projects.id AND status = 0)
WHERE id IN ('00000000-0000-0000-0000-0000000000a1', '00000000-0000-0000-0000-0000000000a2');

COMMIT;

SELECT slug, name, avg_rating, review_count FROM businesses;
SELECT title, business_id, avg_rating, review_count FROM projects;
