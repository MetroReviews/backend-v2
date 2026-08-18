package webhooks

import (
	"net/http"

	"github.com/MetroReviews/backend-v2/api"
	svc "github.com/MetroReviews/backend-v2/webhooks"
	"github.com/infinitybotlist/eureka/uapi"
)

func getWebhookEvents(d uapi.RouteData, r *http.Request) uapi.HttpResponse {
	if _, resp := api.AuthUser(d.Context, r); resp != nil {
		return *resp
	}
	return uapi.HttpResponse{Json: svc.Catalog}
}
