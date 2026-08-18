package reviews

import (
	"context"
	"errors"
	"net/http"

	"github.com/MetroReviews/backend-v2/api"
	"github.com/MetroReviews/backend-v2/helpers"
	"github.com/MetroReviews/backend-v2/state"
	"github.com/MetroReviews/backend-v2/types"
	"github.com/MetroReviews/backend-v2/webhooks"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/infinitybotlist/eureka/uapi"
	"github.com/jackc/pgx/v5"
)

func respondToReview(d uapi.RouteData, r *http.Request) uapi.HttpResponse {
	reviewID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		return uapi.DefaultResponse(http.StatusBadRequest)
	}

	user, resp := api.AuthUser(d.Context, r)
	if resp != nil {
		return *resp
	}

	_, businessID, botID, projectID, err := loadReviewSubject(d.Context, reviewID)
	if errors.Is(err, pgx.ErrNoRows) {
		return uapi.DefaultResponse(http.StatusNotFound)
	}
	if err != nil {
		return helpers.InternalError(err)
	}

	if !user.IsStaff {
		owns, err := ownsSubject(d.Context, user, businessID, botID, projectID)
		if err != nil {
			return helpers.InternalError(err)
		}
		if !owns {
			return helpers.ErrorResponse(http.StatusForbidden, "You do not own the business/bot/project being reviewed")
		}
	}

	var payload types.ReviewResponse
	if hresp, ok := uapi.MarshalReq(r, &payload); !ok {
		return hresp
	}

	rows, err := state.Pool.Query(d.Context, `
		UPDATE reviews SET owner_response = $1, owner_response_at = NOW(), updated_at = NOW()
		WHERE id = $2
		RETURNING `+reviewColumns,
		payload.Response, reviewID,
	)
	if err != nil {
		return helpers.InternalError(err)
	}
	updated, err := pgx.CollectOneRow(rows, pgx.RowToStructByName[types.Review])
	if err != nil {
		return helpers.InternalError(err)
	}

	targetType, targetID := reviewTarget(businessID, botID, projectID)
	webhooks.Dispatch(targetType, targetID, webhooks.EventReviewResponded, updated)

	return uapi.HttpResponse{Json: updated}
}

// ownsSubject checks review-reply permission — the same ownership rule
// webhook registration uses (see the webhooks package's OwnsTarget), just
// resolved from a review's three mutually-exclusive subject columns
// instead of a target_type/target_id pair.
func ownsSubject(ctx context.Context, user *types.User, businessID *uuid.UUID, botID *int64, projectID *uuid.UUID) (bool, error) {
	targetType, targetID := reviewTarget(businessID, botID, projectID)
	return webhooks.OwnsTarget(ctx, user, targetType, targetID)
}
