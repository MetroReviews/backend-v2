package main

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/infinitybotlist/eureka/crypto"
	"github.com/jackc/pgx/v5"
	"golang.org/x/crypto/bcrypt"
)

const seedPassword = "password123"

func seedUsers(ctx context.Context, tx pgx.Tx) error {
	passwordHash, err := bcrypt.GenerateFromPassword([]byte(seedPassword), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("hash seed password: %w", err)
	}

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
			INSERT INTO discord_accounts (discord_id, user_id, nonce)
			VALUES ($1, $2, $3)
			ON CONFLICT (discord_id) DO UPDATE SET nonce = EXCLUDED.nonce`,
			u.discordID, u.id, crypto.RandString(20),
		); err != nil {
			return fmt.Errorf("discord account for %s: %w", u.username, err)
		}
	}

	if _, err := tx.Exec(ctx, `
		INSERT INTO local_accounts (user_id, email, password_hash)
		VALUES ($1, $2, $3)
		ON CONFLICT (user_id) DO UPDATE SET
			email = EXCLUDED.email, password_hash = EXCLUDED.password_hash, updated_at = NOW()`,
		userAlphaID, "alpha_seed@example.com", string(passwordHash),
	); err != nil {
		return fmt.Errorf("local account link for alpha_seed: %w", err)
	}

	if _, err := tx.Exec(ctx, `
		INSERT INTO users (id, username, is_staff)
		VALUES ($1, $2, FALSE)
		ON CONFLICT (id) DO UPDATE SET username = EXCLUDED.username`,
		userLocalID, "local_seed",
	); err != nil {
		return fmt.Errorf("user local_seed: %w", err)
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO local_accounts (user_id, email, password_hash)
		VALUES ($1, $2, $3)
		ON CONFLICT (user_id) DO UPDATE SET
			email = EXCLUDED.email, password_hash = EXCLUDED.password_hash, updated_at = NOW()`,
		userLocalID, "local_seed@example.com", string(passwordHash),
	); err != nil {
		return fmt.Errorf("local account for local_seed: %w", err)
	}

	return nil
}
