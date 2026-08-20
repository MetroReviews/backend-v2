package review

import (
	"context"

	"github.com/jackc/pgx/v5/pgconn"
)

type execer interface {
	Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error)
}

func RecomputeBusinessRating(ctx context.Context, db execer, businessID any) error {
	_, err := db.Exec(ctx, `
		UPDATE businesses SET
			avg_rating   = COALESCE((SELECT AVG(rating) FROM reviews WHERE business_id = $1 AND status = 0), 0),
			review_count = (SELECT COUNT(*) FROM reviews WHERE business_id = $1 AND status = 0),
			updated_at   = NOW()
		WHERE id = $1`, businessID)
	return err
}

func RecomputeProjectRating(ctx context.Context, db execer, projectID any) error {
	_, err := db.Exec(ctx, `
		UPDATE projects SET
			avg_rating   = COALESCE((SELECT AVG(rating) FROM reviews WHERE project_id = $1 AND status = 0), 0),
			review_count = (SELECT COUNT(*) FROM reviews WHERE project_id = $1 AND status = 0),
			updated_at   = NOW()
		WHERE id = $1`, projectID)
	return err
}

func RecomputeHelpfulCount(ctx context.Context, db execer, reviewID any) error {
	_, err := db.Exec(ctx, `
		UPDATE reviews SET helpful_count = (
			SELECT COALESCE(SUM(CASE WHEN helpful THEN 1 ELSE -1 END), 0)
			FROM review_votes WHERE review_id = $1
		) WHERE id = $1`, reviewID)
	return err
}
