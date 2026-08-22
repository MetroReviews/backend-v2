package businesses

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/MetroReviews/backend-v2/api"
	"github.com/MetroReviews/backend-v2/helpers"
	"github.com/MetroReviews/backend-v2/state"
	"github.com/MetroReviews/backend-v2/types"
	"github.com/infinitybotlist/eureka/uapi"
	"github.com/jackc/pgx/v5"
)

// uniqueBusinessSlug returns a slug based on `base` that no business currently
// uses, appending -2, -3, … on collision. The slug column has a unique index,
// so a race can still 409 on insert; this just avoids it in the common case.
func uniqueBusinessSlug(ctx context.Context, base string) (string, error) {
	if base == "" {
		base = "business"
	}
	candidate := base
	for n := 2; n <= 1000; n++ {
		var exists bool
		if err := state.Pool.QueryRow(ctx,
			"SELECT EXISTS(SELECT 1 FROM businesses WHERE slug = $1)", candidate,
		).Scan(&exists); err != nil {
			return "", err
		}
		if !exists {
			return candidate, nil
		}
		candidate = fmt.Sprintf("%s-%d", base, n)
	}
	return candidate, nil
}

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

	gallery, ok := validateGallery(payload.Gallery)
	if !ok {
		return helpers.ErrorResponse(http.StatusBadRequest, "gallery URLs must be https:// and at most 12 images")
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

	// The slug is server-generated so every business is reachable by URL.
	// A client-supplied slug is treated as a preference; the name is the
	// fallback. Either way it's normalized and de-duplicated.
	base := helpers.Slugify(payload.Slug)
	if base == "" {
		base = helpers.Slugify(payload.Name)
	}
	slug, err := uniqueBusinessSlug(d.Context, base)
	if err != nil {
		return helpers.InternalError(err)
	}

	rows, err := state.Pool.Query(d.Context, `
		INSERT INTO businesses (
			category_id, slug, name, description, website, logo, banner,
			address, city, country, metadata, submitted_by, latitude, longitude, gallery
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15)
		ON CONFLICT (slug) DO NOTHING
		RETURNING `+businessColumns,
		payload.CategoryID, slug, payload.Name, payload.Description, payload.Website, payload.Logo, payload.Banner,
		payload.Address, payload.City, payload.Country, metadata, user.ID, payload.Latitude, payload.Longitude, gallery,
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
