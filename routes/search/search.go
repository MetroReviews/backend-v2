package search

import (
	"net/http"
	"time"

	"github.com/MetroReviews/backend-v2/cache"
	"github.com/MetroReviews/backend-v2/helpers"
	"github.com/MetroReviews/backend-v2/state"
	"github.com/MetroReviews/backend-v2/types"
	"github.com/go-chi/chi/v5"
	docs "github.com/infinitybotlist/eureka/doclib"
	"github.com/infinitybotlist/eureka/uapi"
	"github.com/jackc/pgx/v5"
)

const tagName = "Search"

const searchCacheTTL = 30 * time.Second

type Router struct{}

func (Router) Tag() (string, string) {
	return tagName, "Cross-entity search over businesses and projects."
}

func (Router) Routes(r *chi.Mux) {
	uapi.Route{
		Method:  uapi.GET,
		Pattern: "/search",
		OpId:    "search",
		Docs: func() *docs.Doc {
			return &docs.Doc{
				Summary:     "Search",
				Description: "Ranked full-text search across approved businesses and projects. Paginated with `limit`/`offset` (max limit 100).",
				Resp:        []types.SearchResult{},
				RespName:    "SearchResultArray",
				Params: []docs.Parameter{
					{Name: "q", In: "query", Description: "Search query", Required: true, Schema: docs.IdSchema},
					{Name: "offset", In: "query", Description: "Rows to skip (default 0)", Required: false, Schema: docs.IdSchema},
					{Name: "limit", In: "query", Description: "Max rows to return (default 20, max 100)", Required: false, Schema: docs.IdSchema},
				},
			}
		},
		Handler: search,
	}.Route(r)
}

func search(d uapi.RouteData, r *http.Request) uapi.HttpResponse {
	q := r.URL.Query().Get("q")
	if q == "" {
		return helpers.ErrorResponse(http.StatusBadRequest, "q is required")
	}

	limit, offset := helpers.Pagination(r, 20, 100)

	cacheKey := "search:" + r.URL.RawQuery
	if cached, ok := cache.Get[[]types.SearchResult](d.Context, cacheKey); ok {
		return uapi.HttpResponse{Json: cached}
	}

	rows, err := state.Pool.Query(d.Context, `
		SELECT 'business' AS type, id, slug, name, description, avg_rating, review_count,
		       ts_rank(search_vector, websearch_to_tsquery('english', $1)) AS rank
		FROM businesses
		WHERE status = $2 AND search_vector @@ websearch_to_tsquery('english', $1)
		UNION ALL
		SELECT 'project' AS type, id, NULL::text AS slug, title AS name, description, avg_rating, review_count,
		       ts_rank(search_vector, websearch_to_tsquery('english', $1)) AS rank
		FROM projects
		WHERE status = $2 AND search_vector @@ websearch_to_tsquery('english', $1)
		ORDER BY rank DESC
		LIMIT $3 OFFSET $4`,
		q, types.StateApproved, limit, offset,
	)
	if err != nil {
		return helpers.InternalError(err)
	}

	results, err := pgx.CollectRows(rows, pgx.RowToStructByName[types.SearchResult])
	if err != nil {
		return helpers.InternalError(err)
	}

	_ = cache.Set(d.Context, cacheKey, results, searchCacheTTL)

	return uapi.HttpResponse{Json: results}
}
