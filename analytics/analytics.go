package analytics

import (
	"context"
	"time"

	"github.com/MetroReviews/backend-v2/state"
	"github.com/MetroReviews/backend-v2/types"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"go.uber.org/zap"
)

const viewKeyPrefix = "analytics:views:"

func RecordView(ctx context.Context, businessID uuid.UUID) {
	if state.Redis == nil {
		return
	}
	if err := state.Redis.Incr(ctx, viewKeyPrefix+businessID.String()).Err(); err != nil {
		state.Logger.Warn("analytics: failed to buffer view", zap.Error(err))
	}
}

func StartFlusher(ctx context.Context) {
	if state.Redis == nil {
		return
	}

	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			flush(ctx)
		}
	}
}

func flush(ctx context.Context) {
	iter := state.Redis.Scan(ctx, 0, viewKeyPrefix+"*", 0).Iterator()
	for iter.Next(ctx) {
		key := iter.Val()

		count, err := state.Redis.GetDel(ctx, key).Int64()
		if err != nil || count == 0 {
			continue
		}

		id, err := uuid.Parse(key[len(viewKeyPrefix):])
		if err != nil {
			continue
		}

		if _, err := state.Pool.Exec(ctx,
			"UPDATE businesses SET view_count = view_count + $1 WHERE id = $2", count, id,
		); err != nil {
			state.Logger.Warn("analytics: failed to flush view count", zap.Error(err))
		}
	}
	if err := iter.Err(); err != nil {
		state.Logger.Warn("analytics: scan failed", zap.Error(err))
	}
}

type TrendPoint struct {
	WeekStart time.Time `db:"week_start" json:"week_start" description:"Start of the week (Monday, UTC)"`
	Count     int64     `db:"count" json:"count" description:"Reviews posted that week"`
}

func ReviewTrend(ctx context.Context, businessID uuid.UUID, weeks int) ([]TrendPoint, error) {
	rows, err := state.Pool.Query(ctx, `
		SELECT date_trunc('week', created_at) AS week_start, COUNT(*) AS count
		FROM reviews
		WHERE business_id = $1 AND created_at >= NOW() - make_interval(weeks => $2)
		GROUP BY week_start
		ORDER BY week_start ASC`,
		businessID, weeks,
	)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, pgx.RowToStructByName[TrendPoint])
}

type RatingBucket struct {
	Rating int16 `db:"rating" json:"rating" description:"Star rating, 1 to 5"`
	Count  int64 `db:"count" json:"count" description:"Published reviews at this rating"`
}

func RatingDistribution(ctx context.Context, businessID uuid.UUID) ([]RatingBucket, error) {
	rows, err := state.Pool.Query(ctx, `
		SELECT rating, COUNT(*) AS count
		FROM reviews
		WHERE business_id = $1 AND status = $2
		GROUP BY rating
		ORDER BY rating ASC`,
		businessID, types.ReviewStatusPublished,
	)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, pgx.RowToStructByName[RatingBucket])
}
