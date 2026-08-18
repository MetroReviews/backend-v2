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
	"github.com/google/uuid"
	"github.com/infinitybotlist/eureka/uapi"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

func postReview(d uapi.RouteData, r *http.Request) uapi.HttpResponse {
	user, resp := api.AuthUser(d.Context, r)
	if resp != nil {
		return *resp
	}

	var payload types.ReviewCreate
	if hresp, ok := uapi.MarshalReq(r, &payload); !ok {
		return hresp
	}

	if subjectCount(payload.BusinessID, payload.BotID, payload.ProjectID) != 1 {
		return helpers.ErrorResponse(http.StatusBadRequest, "Exactly one of business_id, bot_id or project_id is required")
	}
	if payload.Rating < 1 || payload.Rating > 5 {
		return helpers.ErrorResponse(http.StatusBadRequest, "Rating must be between 1 and 5")
	}

	approved, err := isApproved(d.Context, payload.BusinessID, payload.BotID, payload.ProjectID)
	if err != nil {
		return helpers.InternalError(err)
	}
	if !approved {
		return helpers.ErrorResponse(http.StatusBadRequest, "You can only review an approved business, bot or project")
	}

	tx, err := state.Pool.Begin(d.Context)
	if err != nil {
		return helpers.InternalError(err)
	}
	defer tx.Rollback(d.Context) //nolint:errcheck // no-op once committed

	rows, err := tx.Query(d.Context, `
		INSERT INTO reviews (business_id, bot_id, project_id, author_id, rating, title, body)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING `+reviewColumns,
		payload.BusinessID, payload.BotID, payload.ProjectID, user.ID, payload.Rating, payload.Title, payload.Body,
	)
	if err != nil {
		return helpers.InternalError(err)
	}

	newReview, err := pgx.CollectOneRow(rows, pgx.RowToStructByName[types.Review])
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return helpers.ErrorResponse(http.StatusConflict, "You've already reviewed this")
		}
		if errors.As(err, &pgErr) && pgErr.Code == "23503" {
			return helpers.ErrorResponse(http.StatusBadRequest, "That business/bot/project doesn't exist")
		}
		return helpers.InternalError(err)
	}

	if err := recomputeSubjectRating(d.Context, tx, payload.BusinessID, payload.BotID, payload.ProjectID); err != nil {
		return helpers.InternalError(err)
	}

	if err := tx.Commit(d.Context); err != nil {
		return helpers.InternalError(err)
	}

	targetType, targetID := reviewTarget(payload.BusinessID, payload.BotID, payload.ProjectID)
	webhooks.Dispatch(targetType, targetID, webhooks.EventReviewCreated, newReview)

	return uapi.HttpResponse{Json: newReview}
}

// subjectCount reports how many of a review's three mutually-exclusive
// subject fields are set — exactly one is valid, matching the reviews
// table's CHECK constraint.
func subjectCount(businessID *uuid.UUID, botID *int64, projectID *uuid.UUID) int {
	n := 0
	if businessID != nil {
		n++
	}
	if botID != nil {
		n++
	}
	if projectID != nil {
		n++
	}
	return n
}

// isApproved reports whether the given business, bot or project has
// cleared the review queue — only approved subjects can be reviewed.
func isApproved(ctx context.Context, businessID *uuid.UUID, botID *int64, projectID *uuid.UUID) (bool, error) {
	var st types.State
	var err error
	switch {
	case businessID != nil:
		err = state.Pool.QueryRow(ctx, "SELECT status FROM businesses WHERE id = $1", *businessID).Scan(&st)
	case botID != nil:
		err = state.Pool.QueryRow(ctx, "SELECT state FROM bots WHERE bot_id = $1", *botID).Scan(&st)
	default:
		err = state.Pool.QueryRow(ctx, "SELECT status FROM projects WHERE id = $1", *projectID).Scan(&st)
	}
	if errors.Is(err, pgx.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return st == types.StateApproved, nil
}
