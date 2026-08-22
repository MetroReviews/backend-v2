package businesses

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/MetroReviews/backend-v2/analytics"
	"github.com/MetroReviews/backend-v2/api"
	"github.com/MetroReviews/backend-v2/cache"
	"github.com/MetroReviews/backend-v2/helpers"
	"github.com/MetroReviews/backend-v2/state"
	"github.com/MetroReviews/backend-v2/types"
	"github.com/go-chi/chi/v5"
	"github.com/infinitybotlist/eureka/uapi"
	"github.com/jackc/pgx/v5"
)

const businessDetailTTL = 60 * time.Second
const businessListTTL = 20 * time.Second

func getBusiness(d uapi.RouteData, r *http.Request) uapi.HttpResponse {
	slug := chi.URLParam(r, "slug")
	cacheKey := "biz:detail:" + slug

	if cached, ok := cache.Get[types.Business](d.Context, cacheKey); ok {
		return uapi.HttpResponse{Json: cached}
	}

	// Resolve by slug, falling back to the id — so a business is reachable
	// even if its slug is empty (legacy rows) and the client can link by id.
	rows, err := state.Pool.Query(d.Context,
		"SELECT "+businessColumns+" FROM businesses WHERE slug = $1 OR id::text = $1 LIMIT 1", slug)
	if err != nil {
		return helpers.InternalError(err)
	}

	business, err := pgx.CollectOneRow(rows, pgx.RowToStructByName[types.Business])
	if errors.Is(err, pgx.ErrNoRows) {
		return uapi.DefaultResponse(http.StatusNotFound)
	}
	if err != nil {
		return helpers.InternalError(err)
	}

	if business.Status != types.StateApproved {
		staff, resp := api.AuthStaff(d.Context, r)
		if resp != nil || staff == nil {
			return uapi.DefaultResponse(http.StatusNotFound)
		}
	} else {
		_ = cache.Set(d.Context, cacheKey, business, businessDetailTTL)
		analytics.RecordView(d.Context, business.ID)
	}

	return uapi.HttpResponse{Json: business}
}

func getAllBusinesses(d uapi.RouteData, r *http.Request) uapi.HttpResponse {
	q := r.URL.Query()

	var conditions []string
	var args []any

	arg := func(v any) string {
		args = append(args, v)
		return fmt.Sprintf("$%d", len(args))
	}

	all := q.Get("all") == "true"
	if all {
		if staff, resp := api.AuthStaff(d.Context, r); resp != nil || staff == nil {
			all = false
		}
	}
	if !all {
		conditions = append(conditions, "status = "+arg(types.StateApproved))
	}

	if cat := q.Get("category"); cat != "" {
		conditions = append(conditions, "category_id = (SELECT id FROM categories WHERE slug = "+arg(cat)+")")
	}

	if search := q.Get("q"); search != "" {
		conditions = append(conditions,
			"(name ILIKE "+arg("%"+search+"%")+" OR description ILIKE "+arg("%"+search+"%")+
				" OR search_vector @@ websearch_to_tsquery('english', "+arg(search)+"))")
	}

	if city := q.Get("city"); city != "" {
		conditions = append(conditions, "city ILIKE "+arg(city))
	}
	if country := q.Get("country"); country != "" {
		conditions = append(conditions, "country ILIKE "+arg(country))
	}
	if minRatingStr := q.Get("min_rating"); minRatingStr != "" {
		if minRating, err := strconv.ParseFloat(minRatingStr, 64); err == nil {
			conditions = append(conditions, "avg_rating >= "+arg(minRating))
		}
	}

	var lat, lng *float64
	if v, err := strconv.ParseFloat(q.Get("lat"), 64); err == nil {
		lat = &v
	}
	if v, err := strconv.ParseFloat(q.Get("lng"), 64); err == nil {
		lng = &v
	}
	latArg, lngArg := arg(lat), arg(lng)

	limit, offset := helpers.Pagination(r, 20, 100)

	inner := "SELECT " + businessColumns + ", " + distanceExpr(latArg, lngArg) + " AS distance_km FROM businesses"
	if len(conditions) > 0 {
		inner += " WHERE " + strings.Join(conditions, " AND ")
	}

	secondary := "created_at DESC"
	switch q.Get("sort") {
	case "rating":
		secondary = "avg_rating DESC, review_count DESC"
	case "reviews":
		secondary = "review_count DESC"
	case "distance":
		secondary = "distance_km ASC NULLS LAST"
	}

	query := "SELECT * FROM (" + inner + ") sub"
	if radiusStr := q.Get("radius_km"); radiusStr != "" {
		if radius, err := strconv.ParseFloat(radiusStr, 64); err == nil {
			query += " WHERE distance_km IS NOT NULL AND distance_km <= " + arg(radius)
		}
	}
	query += " ORDER BY featured DESC, " + secondary + " LIMIT " + arg(limit) + " OFFSET " + arg(offset)

	cacheKey := "biz:list:" + r.URL.RawQuery
	if cached, ok := cache.Get[[]types.BusinessSearchResult](d.Context, cacheKey); ok {
		return uapi.HttpResponse{Json: cached}
	}

	rows, err := state.Pool.Query(d.Context, query, args...)
	if err != nil {
		return helpers.InternalError(err)
	}

	list, err := pgx.CollectRows(rows, pgx.RowToStructByName[types.BusinessSearchResult])
	if err != nil {
		return helpers.InternalError(err)
	}

	_ = cache.Set(d.Context, cacheKey, list, businessListTTL)

	return uapi.HttpResponse{Json: list}
}

func distanceExpr(latArg, lngArg string) string {

	return fmt.Sprintf(
		`(CASE WHEN %s::double precision IS NULL OR %s::double precision IS NULL OR latitude IS NULL OR longitude IS NULL THEN NULL ELSE
			6371 * acos(LEAST(1, GREATEST(-1,
				cos(radians(%s::double precision)) * cos(radians(latitude)) * cos(radians(longitude) - radians(%s::double precision))
				+ sin(radians(%s::double precision)) * sin(radians(latitude))
			))) END)`,
		latArg, lngArg, latArg, lngArg, latArg,
	)
}
