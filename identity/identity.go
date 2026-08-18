// Package identity resolves external accounts (currently just Discord) to
// a Metro user, creating the user on first contact. A Metro user is not a
// Discord account — it's an internal row that a Discord account (and, one
// day, some other login method) links to — so anything that only has a
// Discord ID to go on (the OAuth callback, the Discord bot's review
// commands) goes through here rather than treating the Discord ID as the
// identity itself.
package identity

import (
	"context"
	"errors"

	"github.com/MetroReviews/backend-v2/state"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// Lookup returns the Metro user linked to discordID without creating one,
// reporting false if that Discord account has never been seen before.
// Prefer this over EnsureDiscordUser wherever a miss should be a no-op
// rather than minting a throwaway account (e.g. syncing role membership
// for guild members who've never logged in or been reviewed).
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

// EnsureDiscordUser returns the Metro user linked to discordID, creating
// both the user and the link on first contact. username seeds the new
// user's display name; it's ignored if the account already exists (profile
// syncing on every login happens separately, see routes/panel/callback.go).
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
	defer tx.Rollback(ctx) //nolint:errcheck // no-op once committed

	if err := tx.QueryRow(ctx,
		"INSERT INTO users (username) VALUES ($1) RETURNING id", username,
	).Scan(&userID); err != nil {
		return uuid.Nil, err
	}

	// ON CONFLICT DO NOTHING + a RETURNING check, rather than a plain
	// INSERT, because two concurrent first-contacts from the same Discord
	// ID (e.g. two staff clicking Claim near-simultaneously) would otherwise
	// both pass the SELECT above before either INSERT lands.
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
		// Lost the race: use the winner's user and roll back the throwaway
		// user row we just inserted instead of committing an orphan.
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
