package identity

import (
	"context"
	"errors"

	"github.com/MetroReviews/backend-v2/state"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

func Lookup(ctx context.Context, discordID int64) (uuid.UUID, bool, error) {
	var userID uuid.UUID
	err := state.Pool.QueryRow(ctx,
		"SELECT user_id FROM discord_accounts WHERE discord_id = $1", discordID,
	).Scan(&userID)
	if errors.Is(err, pgx.ErrNoRows) {
		return uuid.Nil, false, nil
	}
	if err != nil {
		return uuid.Nil, false, err
	}
	return userID, true, nil
}

func EnsureDiscordUser(ctx context.Context, discordID int64, username string) (uuid.UUID, error) {
	var userID uuid.UUID
	err := state.Pool.QueryRow(ctx,
		"SELECT user_id FROM discord_accounts WHERE discord_id = $1", discordID,
	).Scan(&userID)
	if err == nil {
		return userID, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return uuid.Nil, err
	}

	tx, err := state.Pool.Begin(ctx)
	if err != nil {
		return uuid.Nil, err
	}
	defer tx.Rollback(ctx)

	if err := tx.QueryRow(ctx,
		"INSERT INTO users (username) VALUES ($1) RETURNING id", username,
	).Scan(&userID); err != nil {
		return uuid.Nil, err
	}

	var linked bool
	err = tx.QueryRow(ctx, `
		INSERT INTO discord_accounts (discord_id, user_id) VALUES ($1, $2)
		ON CONFLICT (discord_id) DO NOTHING
		RETURNING TRUE`, discordID, userID,
	).Scan(&linked)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return uuid.Nil, err
	}

	if !linked {

		if err := tx.QueryRow(ctx,
			"SELECT user_id FROM discord_accounts WHERE discord_id = $1", discordID,
		).Scan(&userID); err != nil {
			return uuid.Nil, err
		}
		return userID, tx.Rollback(ctx)
	}

	if err := tx.Commit(ctx); err != nil {
		return uuid.Nil, err
	}
	return userID, nil
}
