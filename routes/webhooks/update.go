package webhooks

import (
	"net/http"
	"strings"

	"github.com/MetroReviews/backend-v2/helpers"
	"github.com/MetroReviews/backend-v2/types"
	svc "github.com/MetroReviews/backend-v2/webhooks"
	"github.com/infinitybotlist/eureka/uapi"
)

func updateWebhook(d uapi.RouteData, r *http.Request) uapi.HttpResponse {
	_, hook, resp := authorizeWebhook(d, r)
	if resp != nil {
		return *resp
	}

	var payload types.WebhookUpdate
	if hresp, ok := uapi.MarshalReq(r, &payload); !ok {
		return hresp
	}

	if bad := invalidEvents(payload.Events); len(bad) > 0 {
		return helpers.ErrorResponse(http.StatusBadRequest, "Unknown event(s): "+strings.Join(bad, ", "))
	}

	updated, err := svc.Update(d.Context, hook.ID, payload.URL, payload.Events, payload.Enabled)
	if err != nil {
		return helpers.InternalError(err)
	}
	return uapi.HttpResponse{Json: updated}
}
