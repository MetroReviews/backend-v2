package main

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

func seedInvites(ctx context.Context, tx pgx.Tx) error {
	redeemedReviewID := reviewE2ID
	invites := []struct {
		id               uuid.UUID
		targetEmail      string
		token            string
		status           int
		redeemedReviewID *uuid.UUID
	}{
		{businessAlphaInvite, "regular-diner@example.com", "seed-invite-redeemed-alpha", 1, &redeemedReviewID},
		{businessAlphaInvitePending, "new-diner@example.com", "seed-invite-pending-alpha", 0, nil},
	}

	for _, inv := range invites {
		if _, err := tx.Exec(ctx, `
			INSERT INTO review_invites (id, business_id, target_email, token, created_by, status, redeemed_review_id, expires_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
			ON CONFLICT (id) DO UPDATE SET
				target_email = EXCLUDED.target_email, token = EXCLUDED.token,
				status = EXCLUDED.status, redeemed_review_id = EXCLUDED.redeemed_review_id,
				expires_at = EXCLUDED.expires_at`,
			inv.id, businessAlpha, inv.targetEmail, inv.token, userReviewerID, inv.status, inv.redeemedReviewID,
			time.Now().Add(14*24*time.Hour),
		); err != nil {
			return fmt.Errorf("invite %s: %w", inv.token, err)
		}
	}
	return nil
}
