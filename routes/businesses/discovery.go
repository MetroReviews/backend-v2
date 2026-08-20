package businesses

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/MetroReviews/backend-v2/analytics"
	"github.com/MetroReviews/backend-v2/api"
	"github.com/MetroReviews/backend-v2/cache"
	"github.com/MetroReviews/backend-v2/helpers"
	"github.com/MetroReviews/backend-v2/invites"
	"github.com/MetroReviews/backend-v2/perms"
	"github.com/MetroReviews/backend-v2/state"
	"github.com/MetroReviews/backend-v2/types"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/infinitybotlist/eureka/uapi"
	"github.com/jackc/pgx/v5"
)

const similarCacheTTL = 5 * time.Minute
const widgetCacheTTL = 5 * time.Minute

func requireOwnerOrStaff(d uapi.RouteData, businessID uuid.UUID, user *types.User) *uapi.HttpResponse {
	if user.IsStaff {
		return nil
	}
	var ownerID *uuid.UUID
	if err := state.Pool.QueryRow(d.Context,
		"SELECT owner_id FROM businesses WHERE id = $1", businessID,
	).Scan(&ownerID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			resp := uapi.DefaultResponse(http.StatusNotFound)
			return &resp
		}
		resp := helpers.InternalError(err)
		return &resp
	}
	if ownerID == nil || *ownerID != user.ID {
		resp := helpers.ErrorResponse(http.StatusForbidden, "You do not own this business")
		return &resp
	}
	return nil
}

func getSimilarBusinesses(d uapi.RouteData, r *http.Request) uapi.HttpResponse {
	businessID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		return uapi.DefaultResponse(http.StatusBadRequest)
	}

	cacheKey := "biz:similar:" + businessID.String()
	if cached, ok := cache.Get[[]types.Business](d.Context, cacheKey); ok {
		return uapi.HttpResponse{Json: cached}
	}

	rows, err := state.Pool.Query(d.Context, `
		SELECT `+businessColumns+`
		FROM businesses
		WHERE status = $1 AND id != $2
		  AND category_id = (SELECT category_id FROM businesses WHERE id = $2)
		ORDER BY featured DESC, avg_rating DESC, review_count DESC
		LIMIT 6`,
		types.StateApproved, businessID,
	)
	if err != nil {
		return helpers.InternalError(err)
	}
	list, err := pgx.CollectRows(rows, pgx.RowToStructByName[types.Business])
	if err != nil {
		return helpers.InternalError(err)
	}

	_ = cache.Set(d.Context, cacheKey, list, similarCacheTTL)
	return uapi.HttpResponse{Json: list}
}

func getRecommendedBusinesses(d uapi.RouteData, r *http.Request) uapi.HttpResponse {
	user, resp := api.AuthUser(d.Context, r)
	if resp != nil {
		return *resp
	}

	rows, err := state.Pool.Query(d.Context, `
		SELECT `+businessColumns+`
		FROM businesses b
		WHERE b.status = $1
		  AND b.category_id IN (
		      SELECT DISTINCT bb.category_id FROM reviews rv
		      JOIN businesses bb ON bb.id = rv.business_id
		      WHERE rv.author_id = $2
		  )
		  AND b.id NOT IN (
		      SELECT business_id FROM reviews WHERE author_id = $2 AND business_id IS NOT NULL
		  )
		ORDER BY b.featured DESC, b.avg_rating DESC, b.review_count DESC
		LIMIT 10`,
		types.StateApproved, user.ID,
	)
	if err != nil {
		return helpers.InternalError(err)
	}
	list, err := pgx.CollectRows(rows, pgx.RowToStructByName[types.Business])
	if err != nil {
		return helpers.InternalError(err)
	}

	if len(list) == 0 {
		rows, err := state.Pool.Query(d.Context, `
			SELECT `+businessColumns+`
			FROM businesses WHERE status = $1
			ORDER BY featured DESC, avg_rating DESC, review_count DESC
			LIMIT 10`,
			types.StateApproved,
		)
		if err != nil {
			return helpers.InternalError(err)
		}
		list, err = pgx.CollectRows(rows, pgx.RowToStructByName[types.Business])
		if err != nil {
			return helpers.InternalError(err)
		}
	}

	return uapi.HttpResponse{Json: list}
}

func getMyBusinesses(d uapi.RouteData, r *http.Request) uapi.HttpResponse {
	user, resp := api.AuthUser(d.Context, r)
	if resp != nil {
		return *resp
	}

	limit, offset := helpers.Pagination(r, 20, 100)

	rows, err := state.Pool.Query(d.Context, `
		SELECT `+businessColumns+`
		FROM businesses
		WHERE owner_id = $1 OR submitted_by = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3`,
		user.ID, limit, offset,
	)
	if err != nil {
		return helpers.InternalError(err)
	}
	list, err := pgx.CollectRows(rows, pgx.RowToStructByName[types.Business])
	if err != nil {
		return helpers.InternalError(err)
	}

	return uapi.HttpResponse{Json: list}
}

func getMyClaims(d uapi.RouteData, r *http.Request) uapi.HttpResponse {
	user, resp := api.AuthUser(d.Context, r)
	if resp != nil {
		return *resp
	}

	limit, offset := helpers.Pagination(r, 20, 100)

	rows, err := state.Pool.Query(d.Context, `
		SELECT id, business_id, user_id, note, status, created_at, resolved_by, resolved_at
		FROM claims
		WHERE user_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3`,
		user.ID, limit, offset,
	)
	if err != nil {
		return helpers.InternalError(err)
	}
	list, err := pgx.CollectRows(rows, pgx.RowToStructByName[types.Claim])
	if err != nil {
		return helpers.InternalError(err)
	}

	return uapi.HttpResponse{Json: list}
}

type businessAnalyticsResponse struct {
	ViewCount          int                      `json:"view_count" description:"Total profile views, flushed periodically from Redis — see the analytics package"`
	ReviewTrend        []analytics.TrendPoint   `json:"review_trend" description:"Reviews posted per week, last 12 weeks"`
	RatingDistribution []analytics.RatingBucket `json:"rating_distribution" description:"Published review count by star rating"`
}

func getBusinessAnalytics(d uapi.RouteData, r *http.Request) uapi.HttpResponse {
	businessID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		return uapi.DefaultResponse(http.StatusBadRequest)
	}

	user, resp := api.AuthUser(d.Context, r)
	if resp != nil {
		return *resp
	}
	if errResp := requireOwnerOrStaff(d, businessID, user); errResp != nil {
		return *errResp
	}

	var viewCount int
	if err := state.Pool.QueryRow(d.Context, "SELECT view_count FROM businesses WHERE id = $1", businessID).Scan(&viewCount); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return uapi.DefaultResponse(http.StatusNotFound)
		}
		return helpers.InternalError(err)
	}

	trend, err := analytics.ReviewTrend(d.Context, businessID, 12)
	if err != nil {
		return helpers.InternalError(err)
	}
	dist, err := analytics.RatingDistribution(d.Context, businessID)
	if err != nil {
		return helpers.InternalError(err)
	}

	return uapi.HttpResponse{Json: businessAnalyticsResponse{
		ViewCount:          viewCount,
		ReviewTrend:        trend,
		RatingDistribution: dist,
	}}
}

type widgetReview struct {
	Rating    int16     `db:"rating" json:"rating"`
	Title     *string   `db:"title" json:"title"`
	Body      string    `db:"body" json:"body"`
	CreatedAt time.Time `db:"created_at" json:"created_at"`
}

type widgetResponse struct {
	Name          string         `json:"name"`
	Slug          string         `json:"slug"`
	AvgRating     float64        `json:"avg_rating"`
	ReviewCount   int            `json:"review_count"`
	Featured      bool           `json:"featured"`
	RecentReviews []widgetReview `json:"recent_reviews"`
}

func getBusinessWidget(d uapi.RouteData, r *http.Request) uapi.HttpResponse {
	businessID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		return uapi.DefaultResponse(http.StatusBadRequest)
	}

	cacheKey := "biz:widget:" + businessID.String()
	if cached, ok := cache.Get[widgetResponse](d.Context, cacheKey); ok {
		return uapi.HttpResponse{Json: cached}
	}

	var w widgetResponse
	var status types.State
	err = state.Pool.QueryRow(d.Context,
		"SELECT name, slug, avg_rating, review_count, featured, status FROM businesses WHERE id = $1", businessID,
	).Scan(&w.Name, &w.Slug, &w.AvgRating, &w.ReviewCount, &w.Featured, &status)
	if errors.Is(err, pgx.ErrNoRows) {
		return uapi.DefaultResponse(http.StatusNotFound)
	}
	if err != nil {
		return helpers.InternalError(err)
	}
	if status != types.StateApproved {
		return uapi.DefaultResponse(http.StatusNotFound)
	}

	rows, err := state.Pool.Query(d.Context, `
		SELECT rating, title, body, created_at FROM reviews
		WHERE business_id = $1 AND status = $2
		ORDER BY created_at DESC LIMIT 3`,
		businessID, types.ReviewStatusPublished,
	)
	if err != nil {
		return helpers.InternalError(err)
	}
	w.RecentReviews, err = pgx.CollectRows(rows, pgx.RowToStructByName[widgetReview])
	if err != nil {
		return helpers.InternalError(err)
	}

	_ = cache.Set(d.Context, cacheKey, w, widgetCacheTTL)
	return uapi.HttpResponse{Json: w}
}

func featureBusiness(d uapi.RouteData, r *http.Request) uapi.HttpResponse {
	businessID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		return uapi.DefaultResponse(http.StatusBadRequest)
	}

	if _, resp := api.AuthPermission(d.Context, r, perms.BusinessesFeature); resp != nil {
		return *resp
	}

	var payload types.BusinessFeature
	if hresp, ok := uapi.MarshalReq(r, &payload); !ok {
		return hresp
	}

	var slug string
	err = state.Pool.QueryRow(d.Context,
		"UPDATE businesses SET featured = $1, featured_until = $2, updated_at = NOW() WHERE id = $3 RETURNING slug",
		payload.Featured, payload.Until, businessID,
	).Scan(&slug)
	if errors.Is(err, pgx.ErrNoRows) {
		return uapi.DefaultResponse(http.StatusNotFound)
	}
	if err != nil {
		return helpers.InternalError(err)
	}

	_ = cache.Del(d.Context, "biz:detail:"+slug)

	return uapi.HttpResponse{Json: types.ApiError{Message: "Updated", Error: false}}
}

func postBusinessInvite(d uapi.RouteData, r *http.Request) uapi.HttpResponse {
	businessID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		return uapi.DefaultResponse(http.StatusBadRequest)
	}

	user, resp := api.AuthUser(d.Context, r)
	if resp != nil {
		return *resp
	}
	if errResp := requireOwnerOrStaff(d, businessID, user); errResp != nil {
		return *errResp
	}
	if resp := helpers.RateLimit(r, "invite-create", 30, time.Hour); resp != nil {
		return *resp
	}

	var payload types.ReviewInviteCreate
	if hresp, ok := uapi.MarshalReq(r, &payload); !ok {
		return hresp
	}

	if !strings.Contains(payload.TargetEmail, "@") {
		return helpers.ErrorResponse(http.StatusBadRequest, "A valid email is required")
	}

	inv, err := invites.Create(d.Context, &businessID, nil, payload.TargetEmail, user.ID)
	if err != nil {
		return helpers.InternalError(err)
	}

	return uapi.HttpResponse{Json: inv}
}

func getInvite(d uapi.RouteData, r *http.Request) uapi.HttpResponse {

	if resp := helpers.RateLimit(r, "invite-lookup", 60, time.Hour); resp != nil {
		return *resp
	}

	inv, err := invites.Lookup(d.Context, chi.URLParam(r, "token"))
	if errors.Is(err, invites.ErrNotFound) {
		return uapi.DefaultResponse(http.StatusNotFound)
	}
	if err != nil {
		return helpers.InternalError(err)
	}
	return uapi.HttpResponse{Json: inv}
}
