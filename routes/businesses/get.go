package businesses

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/MetroReviews/backend-v2/api"
	"github.com/MetroReviews/backend-v2/helpers"
	"github.com/MetroReviews/backend-v2/state"
	"github.com/MetroReviews/backend-v2/types"
	"github.com/go-chi/chi/v5"
	"github.com/infinitybotlist/eureka/uapi"
	"github.com/jackc/pgx/v5"
)

func getBusiness(d uapi.RouteData, r *http.Request) uapi.HttpResponse {
	slug := chi.URLParam(r, "slug")

	rows, err := state.Pool.Query(d.Context, "SELECT "+businessColumns+" FROM businesses WHERE slug = $1", slug)
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
	}

	return uapi.HttpResponse{Json: business}
}

func getAllBusinesses(d uapi.RouteData, r *http.Request) uapi.HttpResponse {
	q := r.URL.Query()

	// Column names below are all literal strings chosen by this code, never
	// derived from request input, so building the WHERE/ORDER clauses with
	// fmt is safe; user input only ever flows in as bind parameters.
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
		conditions = append(conditions, "name ILIKE "+arg("%"+search+"%"))
	}

	query := "SELECT " + businessColumns + " FROM businesses"
	if len(conditions) > 0 {
		query += " WHERE " + strings.Join(conditions, " AND ")
	}

	switch q.Get("sort") {
	case "rating":
		query += " ORDER BY avg_rating DESC, review_count DESC"
	case "reviews":
		query += " ORDER BY review_count DESC"
	default:
		query += " ORDER BY created_at DESC"
	}

	rows, err := state.Pool.Query(d.Context, query, args...)
	if err != nil {
		return helpers.InternalError(err)
	}

	list, err := pgx.CollectRows(rows, pgx.RowToStructByName[types.Business])
	if err != nil {
		return helpers.InternalError(err)
	}

	return uapi.HttpResponse{Json: list}
}
