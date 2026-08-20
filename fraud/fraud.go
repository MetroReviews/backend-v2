package fraud

import (
	"context"
	"time"

	"github.com/MetroReviews/backend-v2/state"
	"github.com/google/uuid"
)

const similarityThreshold = 0.6

const minAccountAge = 10 * time.Minute

func IsDuplicate(ctx context.Context, authorID uuid.UUID, body string) (bool, error) {
	var exists bool
	err := state.Pool.QueryRow(ctx, `
		SELECT EXISTS(
			SELECT 1 FROM reviews
			WHERE author_id = $1
			  AND created_at >= NOW() - INTERVAL '30 days'
			  AND similarity(body, $2) >= $3
		)`,
		authorID, body, similarityThreshold,
	).Scan(&exists)
	return exists, err
}

func AccountTooNew(createdAt time.Time) bool {
	return time.Since(createdAt) < minAccountAge
}
