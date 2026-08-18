package reviews

import (
	"context"
	"errors"
	"net/http"
	"strconv"

	"github.com/MetroReviews/backend-v2/api"
	"github.com/MetroReviews/backend-v2/helpers"
	"github.com/MetroReviews/backend-v2/review"
	"github.com/MetroReviews/backend-v2/state"
	"github.com/MetroReviews/backend-v2/types"
	"github.com/MetroReviews/backend-v2/webhooks"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/infinitybotlist/eureka/uapi"
	"github.com/jackc/pgx/v5"
)

// loadReviewSubject returns a review's author and which subject it belongs
// to, for ownership/permission checks shared by update, delete and response
// handlers.
func loadReviewSubject(ctx context.Context, reviewID uuid.UUID) (authorID uuid.UUID, businessID *uuid.UUID, botID *int64, projectID *uuid.UUID, err error) {
	err = state.Pool.QueryRow(ctx,
		"SELECT author_id, business_id, bot_id, project_id FROM reviews WHERE id = $1", reviewID,
	).Scan(&authorID, &businessID, &botID, &projectID)
	return
}

func updateReview(d uapi.RouteData, r *http.Request) uapi.HttpResponse {
	reviewID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		return uapi.DefaultResponse(http.StatusBadRequest)
	}

	user, resp := api.AuthUser(d.Context, r)
	if resp != nil {
		return *resp
	}

	authorID, businessID, botID, projectID, err := loadReviewSubject(d.Context, reviewID)
	if errors.Is(err, pgx.ErrNoRows) {
		return uapi.DefaultResponse(http.StatusNotFound)
	}
	if err != nil {
		return helpers.InternalError(err)
	}
	if authorID != user.ID {
		return helpers.ErrorResponse(http.StatusForbidden, "You did not write this review")
	}

	var update types.ReviewUpdate
	if hresp, ok := uapi.MarshalReq(r, &update); !ok {
		return hresp
	}
	if update.Rating != nil && (*update.Rating < 1 || *update.Rating > 5) {
		return helpers.ErrorResponse(http.StatusBadRequest, "Rating must be between 1 and 5")
	}

	tx, err := state.Pool.Begin(d.Context)
	if err != nil {
		return helpers.InternalError(err)
	}
	defer tx.Rollback(d.Context) //nolint:errcheck // no-op once committed

	rows, err := tx.Query(d.Context, `
		UPDATE reviews SET
			rating     = COALESCE($1, rating),
			title      = COALESCE($2, title),
			body       = COALESCE($3, body),
			updated_at = NOW()
		WHERE id = $4
		RETURNING `+reviewColumns,
		update.Rating, update.Title, update.Body, reviewID,
	)
	if err != nil {
		return helpers.InternalError(err)
	}
	updated, err := pgx.CollectOneRow(rows, pgx.RowToStructByName[types.Review])
	if err != nil {
		return helpers.InternalError(err)
	}

	if update.Rating != nil {
		if err := recomputeSubjectRating(d.Context, tx, businessID, botID, projectID); err != nil {
			return helpers.InternalError(err)
		}
	}

	if err := tx.Commit(d.Context); err != nil {
		return helpers.InternalError(err)
	}

	targetType, targetID := reviewTarget(businessID, botID, projectID)
	webhooks.Dispatch(targetType, targetID, webhooks.EventReviewUpdated, updated)

	return uapi.HttpResponse{Json: updated}
}

func deleteReview(d uapi.RouteData, r *http.Request) uapi.HttpResponse {
	reviewID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		return uapi.DefaultResponse(http.StatusBadRequest)
	}

	user, resp := api.AuthUser(d.Context, r)
	if resp != nil {
		return *resp
	}

	authorID, businessID, botID, projectID, err := loadReviewSubject(d.Context, reviewID)
	if errors.Is(err, pgx.ErrNoRows) {
		return uapi.DefaultResponse(http.StatusNotFound)
	}
	if err != nil {
		return helpers.InternalError(err)
	}
	if authorID != user.ID && !user.IsStaff {
		return helpers.ErrorResponse(http.StatusForbidden, "You did not write this review")
	}

	tx, err := state.Pool.Begin(d.Context)
	if err != nil {
		return helpers.InternalError(err)
	}
	defer tx.Rollback(d.Context) //nolint:errcheck // no-op once committed

	if _, err := tx.Exec(d.Context, "DELETE FROM reviews WHERE id = $1", reviewID); err != nil {
		return helpers.InternalError(err)
	}
	if err := recomputeSubjectRating(d.Context, tx, businessID, botID, projectID); err != nil {
		return helpers.InternalError(err)
	}

	if err := tx.Commit(d.Context); err != nil {
		return helpers.InternalError(err)
	}

	targetType, targetID := reviewTarget(businessID, botID, projectID)
	webhooks.Dispatch(targetType, targetID, webhooks.EventReviewDeleted, map[string]any{"review_id": reviewID, "author_id": authorID})

	return uapi.HttpResponse{Json: types.ApiError{Message: "Deleted", Error: false}}
}

func recomputeSubjectRating(ctx context.Context, tx pgx.Tx, businessID *uuid.UUID, botID *int64, projectID *uuid.UUID) error {
	switch {
	case businessID != nil:
		return review.RecomputeBusinessRating(ctx, tx, *businessID)
	case botID != nil:
		return review.RecomputeBotRating(ctx, tx, *botID)
	default:
		return review.RecomputeProjectRating(ctx, tx, *projectID)
	}
}

// reviewTarget resolves a review's webhooks.Dispatch target from
// whichever of its three mutually-exclusive subject fields is set —
// shared by every handler that fires a review.* webhook event.
func reviewTarget(businessID *uuid.UUID, botID *int64, projectID *uuid.UUID) (targetType, targetID string) {
	switch {
	case businessID != nil:
		return "business", businessID.String()
	case botID != nil:
		return "bot", strconv.FormatInt(*botID, 10)
	default:
		return "project", projectID.String()
	}
}
