
ALTER TABLE reviews DROP CONSTRAINT reviews_check;
ALTER TABLE reviews DROP COLUMN bot_id;
DELETE FROM reviews WHERE business_id IS NULL AND project_id IS NULL;
ALTER TABLE reviews ADD CONSTRAINT reviews_check CHECK (
    (CASE WHEN business_id IS NOT NULL THEN 1 ELSE 0 END) +
    (CASE WHEN project_id  IS NOT NULL THEN 1 ELSE 0 END) = 1
);

ALTER TABLE moderation_actions DROP CONSTRAINT moderation_actions_target_type_check;
DELETE FROM moderation_actions WHERE target_type = 'bot';
ALTER TABLE moderation_actions ADD CONSTRAINT moderation_actions_target_type_check
    CHECK (target_type IN ('business', 'project'));

ALTER TABLE reports DROP CONSTRAINT reports_target_type_check;
DELETE FROM reports WHERE target_type = 'bot';
ALTER TABLE reports ADD CONSTRAINT reports_target_type_check
    CHECK (target_type IN ('review', 'business', 'project'));

DROP TABLE IF EXISTS bots;
