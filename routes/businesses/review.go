package businesses

import (
	"errors"
	"net/http"

	"github.com/MetroReviews/backend-v2/api"
	"github.com/MetroReviews/backend-v2/helpers"
	"github.com/MetroReviews/backend-v2/rpc"
	"github.com/MetroReviews/backend-v2/types"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/infinitybotlist/eureka/uapi"
)

func reviewAction(d uapi.RouteData, r *http.Request, action types.Action) uapi.HttpResponse {
	businessID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		return uapi.DefaultResponse(http.StatusBadRequest)
	}

	user, resp := api.AuthUser(d.Context, r)
	if resp != nil {
		return *resp
	}

	var reason types.Reason
	if hresp, ok := uapi.MarshalReq(r, &reason); !ok {
		return hresp
	}

	actor := rpc.Actor{UserID: user.ID, DiscordID: user.DiscordID, Source: "api"}
	res, err := rpc.ReviewBusiness(d.Context, actor, businessID, action, reason.Reason)
	var forbidden *rpc.ForbiddenError
	if errors.As(err, &forbidden) {
		return helpers.ErrorResponse(http.StatusForbidden, "Missing required permission: "+forbidden.Permission)
	}
	if err != nil {
		return helpers.InternalError(err)
	}
	if !res.OK {
		return helpers.ErrorResponse(http.StatusBadRequest, res.Message)
	}

	return uapi.HttpResponse{Json: types.ApiError{Message: res.Message, Error: false}}
}
