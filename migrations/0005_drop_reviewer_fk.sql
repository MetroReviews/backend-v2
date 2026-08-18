-- bots.reviewer, listings.reviewer and moderation_actions.reviewer were
-- FKed to users(user_id), which assumed every reviewer had logged into the
-- website. Wrong: Discord slash-command reviewers are authorized by
-- holding the Reviewer/Sudo guild role (see bot/commands.isReviewer), not
-- by having a users row, so claiming/approving/denying from Discord
-- violated this FK for anyone who'd never logged into the panel. Drop it —
-- reviewer is just a Discord ID, same as bots.owner/extra_owners, which
-- were never FKed for the same reason.

ALTER TABLE bots DROP CONSTRAINT bots_reviewer_fkey;
ALTER TABLE listings DROP CONSTRAINT listings_reviewer_fkey;
ALTER TABLE moderation_actions DROP CONSTRAINT moderation_actions_reviewer_fkey;
