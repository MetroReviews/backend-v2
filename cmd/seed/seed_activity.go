package main

import (
	"context"
	"fmt"
	"strconv"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

func seedActions(ctx context.Context, tx pgx.Tx) error {
	actions := []struct {
		id         uuid.UUID
		targetType string
		targetID   string
		action     int
		reason     string
	}{
		{uuid.MustParse("00000000-0000-0000-0000-0000000000b1"), "bot", strconv.FormatInt(1100000000000000002, 10), 0 /* Claim */, "Claimed for review"},
		{uuid.MustParse("00000000-0000-0000-0000-0000000000b2"), "bot", strconv.FormatInt(1100000000000000003, 10), 2 /* Approve */, "Meets all business requirements"},
		{uuid.MustParse("00000000-0000-0000-0000-0000000000b3"), "bot", strconv.FormatInt(1100000000000000004, 10), 3 /* Deny */, "Missing privacy policy"},
		{uuid.MustParse("00000000-0000-0000-0000-0000000000b4"), "project", projectKitchen.String(), 2 /* Approve */, "Great before/after photos, approved"},
	}

	for _, a := range actions {
		if _, err := tx.Exec(ctx, `
			INSERT INTO moderation_actions (id, target_type, target_id, action, reason, reviewer)
			VALUES ($1, $2, $3, $4, $5, $6)
			ON CONFLICT (id) DO UPDATE SET
				target_type = EXCLUDED.target_type,
				target_id = EXCLUDED.target_id,
				action = EXCLUDED.action,
				reason = EXCLUDED.reason,
				reviewer = EXCLUDED.reviewer`,
			a.id, a.targetType, a.targetID, a.action, a.reason, userReviewerID,
		); err != nil {
			return fmt.Errorf("action %s: %w", a.id, err)
		}
	}
	return nil
}

func seedReviews(ctx context.Context, tx pgx.Tx) error {
	type review struct {
		id         uuid.UUID
		businessID *uuid.UUID
		botID      *int64
		projectID  *uuid.UUID
		authorID   uuid.UUID
		rating     int16
		title      string
		body       string
	}

	// businessBeta and projectPatio are still Pending (seedBusinesses/
	// seedProjects), so nothing reviews them yet — only approved
	// businesses/bots/projects are reviewable (see routes/reviews/post.go).
	reviews := []review{
		{uuid.MustParse("00000000-0000-0000-0000-0000000000e1"), &businessAlpha, nil, nil, userBetaID, 5, "Fantastic", "Great food and service, seed review."},
		{uuid.MustParse("00000000-0000-0000-0000-0000000000e2"), &businessAlpha, nil, nil, userExtraID, 4, "Pretty good", "Enjoyed it, seed review."},
		{uuid.MustParse("00000000-0000-0000-0000-0000000000e3"), &businessAlpha, nil, nil, userAlphaID, 3, "A bit pricey", "Good but a bit pricey for the portions, seed review."},
		{uuid.MustParse("00000000-0000-0000-0000-0000000000e4"), nil, int64Ptr(1100000000000000003), nil, userBetaID, 5, "Great utility bot", "Reliable economy commands, seed review."},
		{uuid.MustParse("00000000-0000-0000-0000-0000000000e5"), nil, nil, &projectKitchen, userExtraID, 5, "Beautiful work", "The remodel turned out amazing, seed review."},
	}

	for _, r := range reviews {
		if _, err := tx.Exec(ctx, `
			INSERT INTO reviews (id, business_id, bot_id, project_id, author_id, rating, title, body)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
			ON CONFLICT (id) DO UPDATE SET
				rating = EXCLUDED.rating, title = EXCLUDED.title, body = EXCLUDED.body`,
			r.id, r.businessID, r.botID, r.projectID, r.authorID, r.rating, r.title, r.body,
		); err != nil {
			return fmt.Errorf("review %s: %w", r.id, err)
		}
	}

	// Roll up ratings the same way the API does after a review write.
	for _, id := range []uuid.UUID{businessAlpha, businessBeta} {
		if _, err := tx.Exec(ctx, `
			UPDATE businesses SET
				avg_rating = COALESCE((SELECT AVG(rating) FROM reviews WHERE business_id = $1 AND status = 0), 0),
				review_count = (SELECT COUNT(*) FROM reviews WHERE business_id = $1 AND status = 0)
			WHERE id = $1`, id); err != nil {
			return err
		}
	}
	if _, err := tx.Exec(ctx, `
		UPDATE bots SET
			avg_rating = COALESCE((SELECT AVG(rating) FROM reviews WHERE bot_id = 1100000000000000003 AND status = 0), 0),
			review_count = (SELECT COUNT(*) FROM reviews WHERE bot_id = 1100000000000000003 AND status = 0)
		WHERE bot_id = 1100000000000000003`); err != nil {
		return err
	}
	for _, id := range []uuid.UUID{projectKitchen, projectPatio} {
		if _, err := tx.Exec(ctx, `
			UPDATE projects SET
				avg_rating = COALESCE((SELECT AVG(rating) FROM reviews WHERE project_id = $1 AND status = 0), 0),
				review_count = (SELECT COUNT(*) FROM reviews WHERE project_id = $1 AND status = 0)
			WHERE id = $1`, id); err != nil {
			return err
		}
	}

	return nil
}
