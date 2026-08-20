package reviews

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/MetroReviews/backend-v2/api"
	"github.com/MetroReviews/backend-v2/fraud"
	"github.com/MetroReviews/backend-v2/helpers"
	"github.com/MetroReviews/backend-v2/invites"
	"github.com/MetroReviews/backend-v2/moderation"
	"github.com/MetroReviews/backend-v2/review"
	"github.com/MetroReviews/backend-v2/state"
	"github.com/MetroReviews/backend-v2/types"
	"github.com/MetroReviews/backend-v2/webhooks"
	"github.com/google/uuid"
	"github.com/infinitybotlist/eureka/uapi"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

const maxReviewPhotos = 6

func postReview(d uapi.RouteData, r *http.Request) uapi.HttpResponse {
	user, resp := api.AuthUser(d.Context, r)
	if resp != nil {
		return *resp
	}
	if resp := helpers.RateLimit(r, "review-create", 10, time.Hour); resp != nil {
		return *resp
	}

	var payload types.ReviewCreate
	if hresp, ok := uapi.MarshalReq(r, &payload); !ok {
		return hresp
	}

	if subjectCount(payload.BusinessID, payload.ProjectID) != 1 {
		return helpers.ErrorResponse(http.StatusBadRequest, "Exactly one of business_id or project_id is required")
	}
	if payload.Rating < 1 || payload.Rating > 5 {
		return helpers.ErrorResponse(http.StatusBadRequest, "Rating must be between 1 and 5")
	}

	photos, ok := helpers.ValidateImageURLs(payload.Photos, maxReviewPhotos)
	if !ok {
		return helpers.ErrorResponse(http.StatusBadRequest, "photos must be https:// URLs, at most 6")
	}

	approved, err := isApproved(d.Context, payload.BusinessID, payload.ProjectID)
	if err != nil {
		return helpers.InternalError(err)
	}
	if !approved {
		return helpers.ErrorResponse(http.StatusBadRequest, "You can only review an approved business or project")
	}

	title := ""
	if payload.Title != nil {
		title = *payload.Title
	}

	status := types.ReviewStatusPublished
	var flagReason *string
	if modFlagged, modReason := moderation.Check(d.Context, title, payload.Body); modFlagged {
		status = types.ReviewStatusFlagged
		flagReason = &modReason
	} else if fraud.AccountTooNew(user.CreatedAt) {
		status = types.ReviewStatusFlagged
		reason := "posted by a newly created account"
		flagReason = &reason
	} else if dup, err := fraud.IsDuplicate(d.Context, user.ID, payload.Body); err != nil {
		return helpers.InternalError(err)
	} else if dup {
		status = types.ReviewStatusFlagged
		reason := "near-duplicate of another recent review by the same author"
		flagReason = &reason
	}

	// Best-effort, computed regardless of the outcome above so it's on hand for *future*
	// semantic-duplicate comparisons even when this review wasn't itself flagged by anything.
	// nil whenever OpenAI isn't configured or the call fails.
	embedding := fraud.Embed(d.Context, strings.TrimSpace(title+"\n"+payload.Body))

	// The remaining two checks are the most expensive (an embeddings call, then a full chat
	// completion) — only spend them if nothing free has already flagged this review.
	if status == types.ReviewStatusPublished {
		if semDup, err := fraud.IsSemanticDuplicate(d.Context, payload.BusinessID, payload.ProjectID, embedding); err != nil {
			return helpers.InternalError(err)
		} else if semDup {
			status = types.ReviewStatusFlagged
			reason := "near-duplicate of another recent review on this listing (semantic match)"
			flagReason = &reason
		} else if suspicious, reason := fraud.ClassifyAuthenticity(d.Context, payload.Rating, title, payload.Body); suspicious {
			status = types.ReviewStatusFlagged
			flagReason = &reason
		}
	}

	tx, err := state.Pool.Begin(d.Context)
	if err != nil {
		return helpers.InternalError(err)
	}
	defer tx.Rollback(d.Context)

	rows, err := tx.Query(d.Context, `
		INSERT INTO reviews (business_id, project_id, author_id, rating, title, body, photos, status, flag_reason, embedding)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		RETURNING `+reviewColumns,
		payload.BusinessID, payload.ProjectID, user.ID, payload.Rating, payload.Title, payload.Body, photos, status, flagReason, embedding,
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
			return helpers.ErrorResponse(http.StatusBadRequest, "That business/project doesn't exist")
		}
		return helpers.InternalError(err)
	}

	if payload.InviteToken != nil {
		if err := invites.Redeem(d.Context, tx, *payload.InviteToken, payload.BusinessID, payload.ProjectID, newReview.ID); err != nil {
			switch {
			case errors.Is(err, invites.ErrNotFound), errors.Is(err, invites.ErrSubjectMismatch):
				return helpers.ErrorResponse(http.StatusBadRequest, "Invalid invite token")
			case errors.Is(err, invites.ErrAlreadyRedeemed):
				return helpers.ErrorResponse(http.StatusBadRequest, "That invite has already been used")
			case errors.Is(err, invites.ErrExpired):
				return helpers.ErrorResponse(http.StatusBadRequest, "That invite has expired")
			default:
				return helpers.InternalError(err)
			}
		}
		if _, err := tx.Exec(d.Context, "UPDATE reviews SET verified = TRUE WHERE id = $1", newReview.ID); err != nil {
			return helpers.InternalError(err)
		}
		newReview.Verified = true
	}

	if err := recomputeSubjectRating(d.Context, tx, payload.BusinessID, payload.ProjectID); err != nil {
		return helpers.InternalError(err)
	}

	if err := tx.Commit(d.Context); err != nil {
		return helpers.InternalError(err)
	}
	if payload.BusinessID != nil {
		review.InvalidateBusinessCache(d.Context, *payload.BusinessID)
	}

	targetType, targetID := reviewTarget(payload.BusinessID, payload.ProjectID)
	webhooks.Dispatch(targetType, targetID, webhooks.EventReviewCreated, newReview)

	return uapi.HttpResponse{Json: newReview}
}

func subjectCount(businessID *uuid.UUID, projectID *uuid.UUID) int {
	n := 0
	if businessID != nil {
		n++
	}
	if projectID != nil {
		n++
	}
	return n
}

func isApproved(ctx context.Context, businessID *uuid.UUID, projectID *uuid.UUID) (bool, error) {
	var st types.State
	var err error
	switch {
	case businessID != nil:
		err = state.Pool.QueryRow(ctx, "SELECT status FROM businesses WHERE id = $1", *businessID).Scan(&st)
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
