package reviews

import (
	"errors"
	"net/http"

	"github.com/MetroReviews/backend-v2/api"
	"github.com/MetroReviews/backend-v2/helpers"
	"github.com/MetroReviews/backend-v2/review"
	"github.com/MetroReviews/backend-v2/state"
	"github.com/MetroReviews/backend-v2/types"
	"github.com/MetroReviews/backend-v2/webhooks"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/infinitybotlist/eureka/uapi"
	"github.com/jackc/pgx/v5/pgconn"
)

func voteReview(d uapi.RouteData, r *http.Request) uapi.HttpResponse {
	reviewID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		return uapi.DefaultResponse(http.StatusBadRequest)
	}

	user, resp := api.AuthUser(d.Context, r)
	if resp != nil {
		return *resp
	}

	var payload types.ReviewVote
	if hresp, ok := uapi.MarshalReq(r, &payload); !ok {
		return hresp
	}

	tx, err := state.Pool.Begin(d.Context)
	if err != nil {
		return helpers.InternalError(err)
	}
	defer tx.Rollback(d.Context) //nolint:errcheck // no-op once committed

	if _, err := tx.Exec(d.Context, `
		INSERT INTO review_votes (review_id, user_id, helpful)
		VALUES ($1, $2, $3)
		ON CONFLICT (review_id, user_id) DO UPDATE SET helpful = EXCLUDED.helpful`,
		reviewID, user.ID, payload.Helpful,
	); err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23503" {
			return uapi.DefaultResponse(http.StatusNotFound)
		}
		return helpers.InternalError(err)
	}

	if err := review.RecomputeHelpfulCount(d.Context, tx, reviewID); err != nil {
		return helpers.InternalError(err)
	}

	if err := tx.Commit(d.Context); err != nil {
		return helpers.InternalError(err)
	}

	// Best-effort: the vote itself already succeeded above, so a failure
	// resolving who to notify shouldn't turn into an error response for
	// something that already happened.
	if _, businessID, botID, projectID, err := loadReviewSubject(d.Context, reviewID); err == nil {
		targetType, targetID := reviewTarget(businessID, botID, projectID)
		webhooks.Dispatch(targetType, targetID, webhooks.EventReviewVoted, map[string]any{
			"review_id": reviewID, "user_id": user.ID, "helpful": payload.Helpful,
		})
	}

	return uapi.HttpResponse{Json: types.ApiError{Message: "Vote recorded", Error: false}}
}
