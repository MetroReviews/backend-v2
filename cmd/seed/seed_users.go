package main

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/infinitybotlist/eureka/crypto"
	"github.com/jackc/pgx/v5"
)

// seedUsers seeds a Metro user plus its linked Discord account for each
// fixture identity, mirroring what identity.EnsureDiscordUser +
// routes/panel/callback.go would produce on a real login.
func seedUsers(ctx context.Context, tx pgx.Tx) error {
	users := []struct {
		id        uuid.UUID
		discordID int64
		username  string
		isStaff   bool
	}{
		{userAlphaID, userAlphaDiscord, "alpha_seed", false},
		{userBetaID, userBetaDiscord, "beta_seed", false},
		{userExtraID, userExtraDiscord, "extra_seed", false},
		{userReviewerID, userReviewerDiscord, "reviewer_seed", true},
	}
	for _, u := range users {
		if _, err := tx.Exec(ctx, `
			INSERT INTO users (id, username, is_staff)
			VALUES ($1, $2, $3)
			ON CONFLICT (id) DO UPDATE SET username = EXCLUDED.username, is_staff = EXCLUDED.is_staff`,
			u.id, u.username, u.isStaff,
		); err != nil {
			return fmt.Errorf("user %s: %w", u.username, err)
		}
		if _, err := tx.Exec(ctx, `
			INSERT INTO discord_accounts (discord_id, user_id, nonce, session_token, session_expires_at)
			VALUES ($1, $2, $3, $4, NOW() + interval '30 days')
			ON CONFLICT (discord_id) DO UPDATE SET
				nonce = EXCLUDED.nonce, session_token = EXCLUDED.session_token,
				session_expires_at = EXCLUDED.session_expires_at`,
			u.discordID, u.id, crypto.RandString(20), crypto.RandString(43),
		); err != nil {
			return fmt.Errorf("discord account for %s: %w", u.username, err)
		}
	}
	return nil
}
