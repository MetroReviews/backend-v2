package identity

import (
	"context"
	"errors"
	"strings"

	"github.com/MetroReviews/backend-v2/state"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

var ErrEmailTaken = errors.New("email is already registered")

func NormalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

func LookupLocalAccount(ctx context.Context, email string) (userID uuid.UUID, passwordHash string, found bool, err error) {
	err = state.Pool.QueryRow(ctx,
		"SELECT user_id, password_hash FROM local_accounts WHERE email = $1", NormalizeEmail(email),
	).Scan(&userID, &passwordHash)
	if errors.Is(err, pgx.ErrNoRows) {
		return uuid.Nil, "", false, nil
	}
	if err != nil {
		return uuid.Nil, "", false, err
	}
	return userID, passwordHash, true, nil
}

func CreateLocalUser(ctx context.Context, email, passwordHash, username string) (uuid.UUID, error) {
	email = NormalizeEmail(email)

	tx, err := state.Pool.Begin(ctx)
	if err != nil {
		return uuid.Nil, err
	}
	defer tx.Rollback(ctx)

	var userID uuid.UUID
	if err := tx.QueryRow(ctx,
		"INSERT INTO users (username) VALUES ($1) RETURNING id", username,
	).Scan(&userID); err != nil {
		return uuid.Nil, err
	}

	if _, err := tx.Exec(ctx,
		"INSERT INTO local_accounts (user_id, email, password_hash) VALUES ($1, $2, $3)",
		userID, email, passwordHash,
	); err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return uuid.Nil, ErrEmailTaken
		}
		return uuid.Nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return uuid.Nil, err
	}
	return userID, nil
}

func SetLocalAccount(ctx context.Context, userID uuid.UUID, email, passwordHash string) error {
	email = NormalizeEmail(email)

	_, err := state.Pool.Exec(ctx, `
		INSERT INTO local_accounts (user_id, email, password_hash)
		VALUES ($1, $2, $3)
		ON CONFLICT (user_id) DO UPDATE SET
			email = EXCLUDED.email, password_hash = EXCLUDED.password_hash, updated_at = NOW()`,
		userID, email, passwordHash,
	)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return ErrEmailTaken
		}
		return err
	}
	return nil
}
