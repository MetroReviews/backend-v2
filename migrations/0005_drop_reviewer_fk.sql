
ALTER TABLE bots DROP CONSTRAINT bots_reviewer_fkey;
ALTER TABLE listings DROP CONSTRAINT listings_reviewer_fkey;
ALTER TABLE moderation_actions DROP CONSTRAINT moderation_actions_reviewer_fkey;
