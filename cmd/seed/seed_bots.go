package main

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// seedBots seeds bots owner/extra_owners as raw Discord snowflakes — bot
// ownership describes the Discord bot, not a Metro account (see
// migrations/0006) — while reviewer is a Metro user id, same as businesses.
func seedBots(ctx context.Context, tx pgx.Tx) error {
	type bot struct {
		botID           int64
		username        string
		description     string
		longDescription string
		website         string
		support         string
		library         string
		nsfw            bool
		prefix          string
		tags            []string
		reviewNote      *string
		invite          string
		state           int
		owner           int64
		extraOwners     []int64
		reviewer        *uuid.UUID
	}

	bots := []bot{
		{
			botID:           1100000000000000001,
			username:        "SeedBot Alpha",
			description:     "A demo moderation bot used as seed data.",
			longDescription: "## SeedBot Alpha\n\nHandles auto-moderation, logging and warnings. This entry is seed data for local development.",
			website:         "https://example.com/alpha",
			support:         "https://discord.gg/example-alpha",
			library:         "discord.js",
			prefix:          "!",
			tags:            []string{"moderation", "logging"},
			invite:          "https://discord.com/oauth2/authorize?client_id=1100000000000000001&scope=bot",
			state:           0, // Pending
			owner:           userAlphaDiscord,
		},
		{
			botID:           1100000000000000002,
			username:        "SeedBot Beta",
			description:     "A demo music bot used as seed data.",
			longDescription: "## SeedBot Beta\n\nStreams music from various sources. This entry is seed data for local development.",
			website:         "https://example.com/beta",
			support:         "https://discord.gg/example-beta",
			library:         "discord.py",
			prefix:          "b!",
			tags:            []string{"music"},
			invite:          "https://discord.com/oauth2/authorize?client_id=1100000000000000002&scope=bot",
			state:           1, // UnderReview
			owner:           userBetaDiscord,
			reviewer:        &userReviewerID,
		},
		{
			botID:           1100000000000000003,
			username:        "SeedBot Gamma",
			description:     "A demo economy and utility bot used as seed data.",
			longDescription: "## SeedBot Gamma\n\nAdds an economy system alongside general utility commands. This entry is seed data for local development.",
			website:         "https://example.com/gamma",
			support:         "https://discord.gg/example-gamma",
			library:         "serenity",
			prefix:          "/",
			tags:            []string{"utility", "economy"},
			invite:          "https://discord.com/oauth2/authorize?client_id=1100000000000000003&scope=bot",
			state:           2, // Approved
			owner:           userAlphaDiscord,
			extraOwners:     []int64{userExtraDiscord},
			reviewer:        &userReviewerID,
		},
		{
			botID:           1100000000000000004,
			username:        "SeedBot Delta",
			description:     "A demo NSFW bot used as seed data.",
			longDescription: "## SeedBot Delta\n\nDenied during review for missing a privacy policy. This entry is seed data for local development.",
			website:         "https://example.com/delta",
			library:         "discord.js",
			nsfw:            true,
			prefix:          "d!",
			tags:            []string{"nsfw"},
			reviewNote:      strPtr("Missing privacy policy"),
			invite:          "https://discord.com/oauth2/authorize?client_id=1100000000000000004&scope=bot",
			state:           3, // Denied
			owner:           userBetaDiscord,
			reviewer:        &userReviewerID,
		},
		{
			botID:           1100000000000000005,
			username:        "SeedBot Epsilon",
			description:     "A second demo pending bot used as seed data.",
			longDescription: "## SeedBot Epsilon\n\nA freshly submitted bot awaiting review. This entry is seed data for local development.",
			website:         "https://example.com/epsilon",
			support:         "https://discord.gg/example-epsilon",
			library:         "discord.js",
			prefix:          "e.",
			tags:            []string{"leveling", "economy"},
			invite:          "https://discord.com/oauth2/authorize?client_id=1100000000000000005&scope=bot",
			state:           0, // Pending
			owner:           userExtraDiscord,
		},
	}

	for _, b := range bots {
		extraOwners := b.extraOwners
		if extraOwners == nil {
			extraOwners = []int64{}
		}
		if _, err := tx.Exec(ctx, `
			INSERT INTO bots (
				bot_id, username, banner, description, long_description,
				website, support, donate, library, nsfw, prefix, tags,
				review_note, invite, state, owner, extra_owners,
				reviewer, invite_link
			) VALUES (
				$1, $2, NULL, $3, $4,
				$5, $6, NULL, $7, $8, $9, $10,
				$11, $12, $13, $14, $15,
				$16, NULL
			)
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
				reviewer = EXCLUDED.reviewer`,
			b.botID, b.username, b.description, b.longDescription,
			b.website, b.support, b.library, b.nsfw, b.prefix, b.tags,
			b.reviewNote, b.invite, b.state, b.owner, extraOwners,
			b.reviewer,
		); err != nil {
			return fmt.Errorf("bot %d: %w", b.botID, err)
		}
	}
	return nil
}
