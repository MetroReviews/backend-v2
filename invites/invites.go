package invites

import (
	"context"
	"errors"
	"time"

	"github.com/MetroReviews/backend-v2/mail"
	"github.com/MetroReviews/backend-v2/state"
	"github.com/MetroReviews/backend-v2/types"
	"github.com/google/uuid"
	"github.com/infinitybotlist/eureka/crypto"
	"github.com/jackc/pgx/v5"
)

const inviteColumns = `id, business_id, project_id, target_email, token, created_by, status, redeemed_review_id, expires_at, created_at`

const TTL = 14 * 24 * time.Hour

var (
	ErrNotFound        = errors.New("invite not found")
	ErrAlreadyRedeemed = errors.New("invite already redeemed")
	ErrExpired         = errors.New("invite expired")

	ErrSubjectMismatch = errors.New("invite does not match the business/project being reviewed")
)

func scan(row pgx.Row) (*types.ReviewInvite, error) {
	var inv types.ReviewInvite
	if err := row.Scan(
		&inv.ID, &inv.BusinessID, &inv.ProjectID, &inv.TargetEmail, &inv.Token,
		&inv.CreatedBy, &inv.Status, &inv.RedeemedReviewID, &inv.ExpiresAt, &inv.CreatedAt,
	); err != nil {
		return nil, err
	}
	return &inv, nil
}

func Create(ctx context.Context, businessID, projectID *uuid.UUID, targetEmail string, createdBy uuid.UUID) (*types.ReviewInvite, error) {
	token := crypto.RandString(32)

	inv, err := scan(state.Pool.QueryRow(ctx, `
		INSERT INTO review_invites (business_id, project_id, target_email, token, created_by, expires_at)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING `+inviteColumns,
		businessID, projectID, targetEmail, token, createdBy, time.Now().Add(TTL),
	))
	if err != nil {
		return nil, err
	}

	_ = mail.Send(targetEmail, "You're invited to leave a review",
		"You've been asked to share your experience. Use this code to leave a review: "+token)

	return inv, nil
}

func Lookup(ctx context.Context, token string) (*types.ReviewInvite, error) {
	inv, err := scan(state.Pool.QueryRow(ctx, "SELECT "+inviteColumns+" FROM review_invites WHERE token = $1", token))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	return inv, err
}

func Redeem(ctx context.Context, tx pgx.Tx, token string, businessID, projectID *uuid.UUID, reviewID uuid.UUID) error {
	inv, err := scan(tx.QueryRow(ctx, "SELECT "+inviteColumns+" FROM review_invites WHERE token = $1 FOR UPDATE", token))
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	}
	if err != nil {
		return err
	}
	if inv.Status == types.InviteStatusRedeemed {
		return ErrAlreadyRedeemed
	}
	if time.Now().After(inv.ExpiresAt) {
		return ErrExpired
	}
	if !sameSubject(inv.BusinessID, businessID) || !sameSubject(inv.ProjectID, projectID) {
		return ErrSubjectMismatch
	}

	_, err = tx.Exec(ctx, "UPDATE review_invites SET status = $1, redeemed_review_id = $2 WHERE id = $3",
		types.InviteStatusRedeemed, reviewID, inv.ID)
	return err
}

func sameSubject(a, b *uuid.UUID) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}
	return *a == *b
}
