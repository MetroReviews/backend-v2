package webhooks

import (
	"net/http"

	"github.com/MetroReviews/backend-v2/helpers"
	"github.com/MetroReviews/backend-v2/types"
	svc "github.com/MetroReviews/backend-v2/webhooks"
	"github.com/infinitybotlist/eureka/uapi"
)

func testWebhook(d uapi.RouteData, r *http.Request) uapi.HttpResponse {
	_, hook, resp := authorizeWebhook(d, r)
	if resp != nil {
		return *resp
	}

	if err := svc.DeliverTest(*hook); err != nil {
		return helpers.ErrorResponse(http.StatusBadGateway, "Test delivery failed: "+err.Error())
	}
	return uapi.HttpResponse{Json: types.ApiError{Message: "Test delivery succeeded", Error: false}}
}
