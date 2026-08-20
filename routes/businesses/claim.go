package businesses

import (
	"errors"
	"net/http"

	"github.com/MetroReviews/backend-v2/api"
	"github.com/MetroReviews/backend-v2/helpers"
	"github.com/MetroReviews/backend-v2/state"
	"github.com/MetroReviews/backend-v2/types"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/infinitybotlist/eureka/uapi"
	"github.com/jackc/pgx/v5"
)

func postClaim(d uapi.RouteData, r *http.Request) uapi.HttpResponse {
	businessID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		return uapi.DefaultResponse(http.StatusBadRequest)
	}

	user, resp := api.AuthUser(d.Context, r)
	if resp != nil {
		return *resp
	}

	var payload types.ClaimCreate
	if hresp, ok := uapi.MarshalReq(r, &payload); !ok {
		return hresp
	}

	var exists bool
	if err := state.Pool.QueryRow(d.Context,
		"SELECT EXISTS(SELECT 1 FROM businesses WHERE id = $1)", businessID,
	).Scan(&exists); err != nil {
		return helpers.InternalError(err)
	}
	if !exists {
		return uapi.DefaultResponse(http.StatusNotFound)
	}

	rows, err := state.Pool.Query(d.Context, `
		INSERT INTO claims (business_id, user_id, note)
		VALUES ($1, $2, $3)
		RETURNING id, business_id, user_id, note, status, created_at, resolved_by, resolved_at`,
		businessID, user.ID, payload.Note,
	)
	if err != nil {
		return helpers.InternalError(err)
	}

	claim, err := pgx.CollectOneRow(rows, pgx.RowToStructByName[types.Claim])
	if err != nil {
		return helpers.InternalError(err)
	}

	return uapi.HttpResponse{Json: claim}
}

func resolveClaim(d uapi.RouteData, r *http.Request, outcome types.ClaimStatus) uapi.HttpResponse {
	businessID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		return uapi.DefaultResponse(http.StatusBadRequest)
	}
	claimID, err := uuid.Parse(chi.URLParam(r, "claim_id"))
	if err != nil {
		return uapi.DefaultResponse(http.StatusBadRequest)
	}

	staff, resp := api.AuthStaff(d.Context, r)
	if resp != nil {
		return *resp
	}

	var claimantID uuid.UUID
	var status types.ClaimStatus
	err = state.Pool.QueryRow(d.Context,
		"SELECT user_id, status FROM claims WHERE id = $1 AND business_id = $2", claimID, businessID,
	).Scan(&claimantID, &status)
	if errors.Is(err, pgx.ErrNoRows) {
		return uapi.DefaultResponse(http.StatusNotFound)
	}
	if err != nil {
		return helpers.InternalError(err)
	}
	if status != types.ClaimStatusPending {
		return helpers.ErrorResponse(http.StatusBadRequest, "This claim has already been resolved")
	}

	tx, err := state.Pool.Begin(d.Context)
	if err != nil {
		return helpers.InternalError(err)
	}
	defer tx.Rollback(d.Context)

	if _, err := tx.Exec(d.Context,
		"UPDATE claims SET status = $1, resolved_by = $2, resolved_at = NOW() WHERE id = $3",
		outcome, staff.ID, claimID,
	); err != nil {
		return helpers.InternalError(err)
	}

	if outcome == types.ClaimStatusApproved {
		if _, err := tx.Exec(d.Context,
			"UPDATE businesses SET owner_id = $1, updated_at = NOW() WHERE id = $2", claimantID, businessID,
		); err != nil {
			return helpers.InternalError(err)
		}
	}

	if err := tx.Commit(d.Context); err != nil {
		return helpers.InternalError(err)
	}

	return uapi.HttpResponse{Json: types.ApiError{Message: "Done", Error: false}}
}
