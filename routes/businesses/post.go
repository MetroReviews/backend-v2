package businesses

import (
	"errors"
	"net/http"

	"github.com/MetroReviews/backend-v2/api"
	"github.com/MetroReviews/backend-v2/helpers"
	"github.com/MetroReviews/backend-v2/state"
	"github.com/MetroReviews/backend-v2/types"
	"github.com/infinitybotlist/eureka/uapi"
	"github.com/jackc/pgx/v5"
)

func postBusiness(d uapi.RouteData, r *http.Request) uapi.HttpResponse {
	user, resp := api.AuthUser(d.Context, r)
	if resp != nil {
		return *resp
	}

	var payload types.BusinessCreate
	if hresp, ok := uapi.MarshalReq(r, &payload); !ok {
		return hresp
	}

	metadata := payload.Metadata
	if metadata == nil {
		metadata = map[string]any{}
	}

	var categoryExists bool
	if err := state.Pool.QueryRow(d.Context,
		"SELECT EXISTS(SELECT 1 FROM categories WHERE id = $1)", payload.CategoryID,
	).Scan(&categoryExists); err != nil {
		return helpers.InternalError(err)
	}
	if !categoryExists {
		return helpers.ErrorResponse(http.StatusBadRequest, "Unknown category_id")
	}

	// ON CONFLICT DO NOTHING RETURNING makes the slug uniqueness check
	// atomic, same pattern used by POST /bots for bot_id.
	rows, err := state.Pool.Query(d.Context, `
		INSERT INTO businesses (
			category_id, slug, name, description, website, logo, banner,
			address, city, country, metadata, submitted_by
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		ON CONFLICT (slug) DO NOTHING
		RETURNING `+businessColumns,
		payload.CategoryID, payload.Slug, payload.Name, payload.Description, payload.Website, payload.Logo, payload.Banner,
		payload.Address, payload.City, payload.Country, metadata, user.ID,
	)
	if err != nil {
		return helpers.InternalError(err)
	}

	business, err := pgx.CollectOneRow(rows, pgx.RowToStructByName[types.Business])
	if errors.Is(err, pgx.ErrNoRows) {
		return helpers.ErrorResponse(http.StatusConflict, "That slug is already taken")
	}
	if err != nil {
		return helpers.InternalError(err)
	}

	return uapi.HttpResponse{Json: business}
}
