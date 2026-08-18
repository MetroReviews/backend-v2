package webhooks

import (
	"net/http"

	"github.com/MetroReviews/backend-v2/helpers"
	svc "github.com/MetroReviews/backend-v2/webhooks"
	"github.com/infinitybotlist/eureka/uapi"
)

func getWebhooks(d uapi.RouteData, r *http.Request) uapi.HttpResponse {
	targetType := r.URL.Query().Get("target_type")
	targetID := r.URL.Query().Get("target_id")
	if targetType == "" || targetID == "" {
		return uapi.DefaultResponse(http.StatusBadRequest)
	}

	if _, resp := authorizeTarget(d, r, targetType, targetID); resp != nil {
		return *resp
	}

	list, err := svc.List(d.Context, targetType, targetID)
	if err != nil {
		return helpers.InternalError(err)
	}
	return uapi.HttpResponse{Json: list}
}
