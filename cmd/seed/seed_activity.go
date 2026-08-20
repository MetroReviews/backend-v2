package main

import (
	"context"
	"fmt"

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
		{uuid.MustParse("00000000-0000-0000-0000-0000000000b4"), "project", projectKitchen.String(), 2, "Great before/after photos, approved"},
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

var reviewE2ID = uuid.MustParse("00000000-0000-0000-0000-0000000000e2")

func seedReviews(ctx context.Context, tx pgx.Tx) error {
	type review struct {
		id         uuid.UUID
		businessID *uuid.UUID
		projectID  *uuid.UUID
		authorID   uuid.UUID
		rating     int16
		title      string
		body       string
		photos     []string
		verified   bool
	}

	reviews := []review{
		{uuid.MustParse("00000000-0000-0000-0000-0000000000e1"), &businessAlpha, nil, userBetaID, 5, "Fantastic", "Great food and service, seed review.", []string{}, false},
		{reviewE2ID, &businessAlpha, nil, userExtraID, 4, "Pretty good", "Enjoyed it, seed review.",
			[]string{"https://example.com/reviews/e2-photo-1.jpg"}, true},
		{uuid.MustParse("00000000-0000-0000-0000-0000000000e3"), &businessAlpha, nil, userAlphaID, 3, "A bit pricey", "Good but a bit pricey for the portions, seed review.", []string{}, false},
		{uuid.MustParse("00000000-0000-0000-0000-0000000000e5"), nil, &projectKitchen, userExtraID, 5, "Beautiful work", "The remodel turned out amazing, seed review.", []string{}, false},
	}

	for _, r := range reviews {
		if _, err := tx.Exec(ctx, `
			INSERT INTO reviews (id, business_id, project_id, author_id, rating, title, body, photos, verified)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
			ON CONFLICT (id) DO UPDATE SET
				rating = EXCLUDED.rating, title = EXCLUDED.title, body = EXCLUDED.body,
				photos = EXCLUDED.photos, verified = EXCLUDED.verified`,
			r.id, r.businessID, r.projectID, r.authorID, r.rating, r.title, r.body, r.photos, r.verified,
		); err != nil {
			return fmt.Errorf("review %s: %w", r.id, err)
		}
	}

	for _, id := range []uuid.UUID{businessAlpha, businessBeta} {
		if _, err := tx.Exec(ctx, `
			UPDATE businesses SET
				avg_rating = COALESCE((SELECT AVG(rating) FROM reviews WHERE business_id = $1 AND status = 0), 0),
				review_count = (SELECT COUNT(*) FROM reviews WHERE business_id = $1 AND status = 0)
			WHERE id = $1`, id); err != nil {
			return err
		}
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
