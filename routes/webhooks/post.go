package webhooks

import (
	"net/http"
	"strings"

	"github.com/MetroReviews/backend-v2/helpers"
	"github.com/MetroReviews/backend-v2/types"
	svc "github.com/MetroReviews/backend-v2/webhooks"
	"github.com/infinitybotlist/eureka/uapi"
)

func createWebhook(d uapi.RouteData, r *http.Request) uapi.HttpResponse {
	var payload types.WebhookCreate
	if hresp, ok := uapi.MarshalReq(r, &payload); !ok {
		return hresp
	}

	user, resp := authorizeTarget(d, r, payload.TargetType, payload.TargetID)
	if resp != nil {
		return *resp
	}

	if bad := invalidEvents(payload.Events); len(bad) > 0 {
		return helpers.ErrorResponse(http.StatusBadRequest, "Unknown event(s): "+strings.Join(bad, ", "))
	}

	hook, err := svc.Create(d.Context, payload.TargetType, payload.TargetID, payload.URL, payload.Events, user.ID)
	if err != nil {
		return helpers.InternalError(err)
	}

	return uapi.HttpResponse{Json: types.WebhookRevealed{Webhook: *hook, Secret: hook.Secret}}
}
