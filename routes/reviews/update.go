package reviews

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/MetroReviews/backend-v2/api"
	"github.com/MetroReviews/backend-v2/fraud"
	"github.com/MetroReviews/backend-v2/helpers"
	"github.com/MetroReviews/backend-v2/moderation"
	"github.com/MetroReviews/backend-v2/review"
	"github.com/MetroReviews/backend-v2/state"
	"github.com/MetroReviews/backend-v2/types"
	"github.com/MetroReviews/backend-v2/webhooks"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/infinitybotlist/eureka/uapi"
	"github.com/jackc/pgx/v5"
)

func loadReviewSubject(ctx context.Context, reviewID uuid.UUID) (authorID uuid.UUID, businessID *uuid.UUID, projectID *uuid.UUID, err error) {
	err = state.Pool.QueryRow(ctx,
		"SELECT author_id, business_id, project_id FROM reviews WHERE id = $1", reviewID,
	).Scan(&authorID, &businessID, &projectID)
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

	authorID, businessID, projectID, err := loadReviewSubject(d.Context, reviewID)
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

	// Re-screen edited text with the same checks review creation runs (minus the LLM
	// authenticity classifier — see below). Only escalates to Flagged when the new text trips
	// something; an edit that doesn't touch title/body never changes status, and a flag is
	// never lifted here (staff clear that manually).
	var flagged bool
	var flagReason *string
	var newEmbedding []float32
	var refreshEmbedding bool
	if update.Title != nil || update.Body != nil {
		newTitle := ""
		if update.Title != nil {
			newTitle = *update.Title
		}
		newBody := ""
		if update.Body != nil {
			newBody = *update.Body
		}

		if modFlagged, reason := moderation.Check(d.Context, newTitle, newBody); modFlagged {
			flagged = true
			flagReason = &reason
		}

		// Re-embed regardless of the moderation verdict, so stored embeddings stay current for
		// future semantic-duplicate comparisons against this review.
		newEmbedding = fraud.Embed(d.Context, strings.TrimSpace(newTitle+"\n"+newBody))
		refreshEmbedding = true

		if !flagged {
			if semDup, err := fraud.IsSemanticDuplicate(d.Context, businessID, projectID, newEmbedding); err != nil {
				return helpers.InternalError(err)
			} else if semDup {
				flagged = true
				reason := "near-duplicate of another recent review on this listing (semantic match)"
				flagReason = &reason
			}
		}
		// The LLM authenticity classifier (fraud.ClassifyAuthenticity) is creation-only: it
		// needs a star rating for context, and an edit doesn't always carry one — re-running it
		// here would mean an extra DB round-trip to look up the unchanged rating for what's
		// usually a minor text tweak. The cheaper checks above still apply to every edit.
	}

	tx, err := state.Pool.Begin(d.Context)
	if err != nil {
		return helpers.InternalError(err)
	}
	defer tx.Rollback(d.Context)

	rows, err := tx.Query(d.Context, `
		UPDATE reviews SET
			rating      = COALESCE($1, rating),
			title       = COALESCE($2, title),
			body        = COALESCE($3, body),
			status      = CASE WHEN $5 THEN $6 ELSE status END,
			flag_reason = CASE WHEN $5 THEN $7 ELSE flag_reason END,
			embedding   = CASE WHEN $8 THEN $9 ELSE embedding END,
			updated_at  = NOW()
		WHERE id = $4
		RETURNING `+reviewColumns,
		update.Rating, update.Title, update.Body, reviewID, flagged, types.ReviewStatusFlagged, flagReason, refreshEmbedding, newEmbedding,
	)
	if err != nil {
		return helpers.InternalError(err)
	}
	updated, err := pgx.CollectOneRow(rows, pgx.RowToStructByName[types.Review])
	if err != nil {
		return helpers.InternalError(err)
	}

	if update.Rating != nil {
		if err := recomputeSubjectRating(d.Context, tx, businessID, projectID); err != nil {
			return helpers.InternalError(err)
		}
	}

	if err := tx.Commit(d.Context); err != nil {
		return helpers.InternalError(err)
	}
	if businessID != nil {
		review.InvalidateBusinessCache(d.Context, *businessID)
	}

	targetType, targetID := reviewTarget(businessID, projectID)
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

	authorID, businessID, projectID, err := loadReviewSubject(d.Context, reviewID)
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
	defer tx.Rollback(d.Context)

	if _, err := tx.Exec(d.Context, "DELETE FROM reviews WHERE id = $1", reviewID); err != nil {
		return helpers.InternalError(err)
	}
	if err := recomputeSubjectRating(d.Context, tx, businessID, projectID); err != nil {
		return helpers.InternalError(err)
	}

	if err := tx.Commit(d.Context); err != nil {
		return helpers.InternalError(err)
	}
	if businessID != nil {
		review.InvalidateBusinessCache(d.Context, *businessID)
	}

	targetType, targetID := reviewTarget(businessID, projectID)
	webhooks.Dispatch(targetType, targetID, webhooks.EventReviewDeleted, map[string]any{"review_id": reviewID, "author_id": authorID})

	return uapi.HttpResponse{Json: types.ApiError{Message: "Deleted", Error: false}}
}

func recomputeSubjectRating(ctx context.Context, tx pgx.Tx, businessID *uuid.UUID, projectID *uuid.UUID) error {
	switch {
	case businessID != nil:
		return review.RecomputeBusinessRating(ctx, tx, *businessID)
	default:
		return review.RecomputeProjectRating(ctx, tx, *projectID)
	}
}

func reviewTarget(businessID *uuid.UUID, projectID *uuid.UUID) (targetType, targetID string) {
	switch {
	case businessID != nil:
		return "business", businessID.String()
	default:
		return "project", projectID.String()
	}
}
