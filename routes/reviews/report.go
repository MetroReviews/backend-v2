package reviews

import (
	"net/http"

	"github.com/MetroReviews/backend-v2/api"
	"github.com/MetroReviews/backend-v2/helpers"
	"github.com/MetroReviews/backend-v2/state"
	"github.com/MetroReviews/backend-v2/types"
	"github.com/go-chi/chi/v5"
	"github.com/infinitybotlist/eureka/uapi"
	"github.com/jackc/pgx/v5"
)

// postReport files a report against targetType ("review", "business" or
// "bot"), reading the target's ID from the {id} path param.
func postReport(d uapi.RouteData, r *http.Request, targetType string) uapi.HttpResponse {
	targetID := chi.URLParam(r, "id")
	if targetID == "" {
		return uapi.DefaultResponse(http.StatusBadRequest)
	}

	user, resp := api.AuthUser(d.Context, r)
	if resp != nil {
		return *resp
	}

	var payload types.ReportCreate
	if hresp, ok := uapi.MarshalReq(r, &payload); !ok {
		return hresp
	}

	rows, err := state.Pool.Query(d.Context, `
		INSERT INTO reports (target_type, target_id, reporter_id, reason)
		VALUES ($1, $2, $3, $4)
		RETURNING id, target_type, target_id, reporter_id, reason, status, created_at, resolved_by, resolved_at`,
		targetType, targetID, user.ID, payload.Reason,
	)
	if err != nil {
		return helpers.InternalError(err)
	}

	report, err := pgx.CollectOneRow(rows, pgx.RowToStructByName[types.Report])
	if err != nil {
		return helpers.InternalError(err)
	}

	return uapi.HttpResponse{Json: report}
}
