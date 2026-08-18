package review

import (
	"context"

	"github.com/jackc/pgx/v5/pgconn"
)

// execer is satisfied by both *pgxpool.Pool and pgx.Tx, so callers can
// recompute a rating either standalone or as part of a larger transaction
// (e.g. alongside the review insert/update/delete that triggered it).
type execer interface {
	Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error)
}

// RecomputeBusinessRating recalculates a business's avg_rating/review_count
// from its published reviews. Called after any write to that business's
// reviews.
func RecomputeBusinessRating(ctx context.Context, db execer, businessID any) error {
	_, err := db.Exec(ctx, `
		UPDATE businesses SET
			avg_rating   = COALESCE((SELECT AVG(rating) FROM reviews WHERE business_id = $1 AND status = 0), 0),
			review_count = (SELECT COUNT(*) FROM reviews WHERE business_id = $1 AND status = 0),
			updated_at   = NOW()
		WHERE id = $1`, businessID)
	return err
}

// RecomputeBotRating is RecomputeBusinessRating's counterpart for bots.
func RecomputeBotRating(ctx context.Context, db execer, botID int64) error {
	_, err := db.Exec(ctx, `
		UPDATE bots SET
			avg_rating   = COALESCE((SELECT AVG(rating) FROM reviews WHERE bot_id = $1 AND status = 0), 0),
			review_count = (SELECT COUNT(*) FROM reviews WHERE bot_id = $1 AND status = 0)
		WHERE bot_id = $1`, botID)
	return err
}

// RecomputeProjectRating is RecomputeBusinessRating's counterpart for
// projects.
func RecomputeProjectRating(ctx context.Context, db execer, projectID any) error {
	_, err := db.Exec(ctx, `
		UPDATE projects SET
			avg_rating   = COALESCE((SELECT AVG(rating) FROM reviews WHERE project_id = $1 AND status = 0), 0),
			review_count = (SELECT COUNT(*) FROM reviews WHERE project_id = $1 AND status = 0),
			updated_at   = NOW()
		WHERE id = $1`, projectID)
	return err
}

// RecomputeHelpfulCount recalculates a review's net helpful vote count
// (helpful votes minus unhelpful votes) after a vote is cast or changed.
func RecomputeHelpfulCount(ctx context.Context, db execer, reviewID any) error {
	_, err := db.Exec(ctx, `
		UPDATE reviews SET helpful_count = (
			SELECT COALESCE(SUM(CASE WHEN helpful THEN 1 ELSE -1 END), 0)
			FROM review_votes WHERE review_id = $1
		) WHERE id = $1`, reviewID)
	return err
}
