package webhooks

import (
	"net/http"

	"github.com/MetroReviews/backend-v2/helpers"
	"github.com/MetroReviews/backend-v2/types"
	svc "github.com/MetroReviews/backend-v2/webhooks"
	"github.com/infinitybotlist/eureka/uapi"
)

func deleteWebhook(d uapi.RouteData, r *http.Request) uapi.HttpResponse {
	_, hook, resp := authorizeWebhook(d, r)
	if resp != nil {
		return *resp
	}

	if err := svc.Delete(d.Context, hook.ID); err != nil {
		return helpers.InternalError(err)
	}
	return uapi.HttpResponse{Json: types.ApiError{Message: "Webhook deleted", Error: false}}
}

func rotateWebhookSecret(d uapi.RouteData, r *http.Request) uapi.HttpResponse {
	_, hook, resp := authorizeWebhook(d, r)
	if resp != nil {
		return *resp
	}

	rotated, err := svc.RotateSecret(d.Context, hook.ID)
	if err != nil {
		return helpers.InternalError(err)
	}
	return uapi.HttpResponse{Json: types.WebhookRevealed{Webhook: *rotated, Secret: rotated.Secret}}
}
