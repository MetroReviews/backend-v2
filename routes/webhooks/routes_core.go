package webhooks

import (
	"github.com/MetroReviews/backend-v2/types"
	"github.com/go-chi/chi/v5"
	docs "github.com/infinitybotlist/eureka/doclib"
	"github.com/infinitybotlist/eureka/uapi"
)

func registerRoutes(r *chi.Mux) {
	uapi.Route{
		Method:  uapi.GET,
		Pattern: "/webhook-events",
		OpId:    "get_webhook_events",
		Auth:    []uapi.AuthType{{Type: "User"}},
		Docs: func() *docs.Doc {
			return &docs.Doc{
				Summary:     "Get Webhook Events",
				Description: "Returns the fixed catalog of events a webhook can subscribe to.",
				Resp:        []types.WebhookEvent{},
				RespName:    "WebhookEventArray",
			}
		},
		Handler: getWebhookEvents,
	}.Route(r)

	uapi.Route{
		Method:  uapi.GET,
		Pattern: "/webhooks",
		OpId:    "get_webhooks",
		Auth:    []uapi.AuthType{{Type: "User"}},
		Docs: func() *docs.Doc {
			return &docs.Doc{
				Summary:     "Get Webhooks",
				Description: "Lists the webhooks registered against one target. Requires owning that target (or a staff session).",
				Resp:        []types.Webhook{},
				RespName:    "WebhookArray",
				Params: []docs.Parameter{
					{Name: "target_type", In: "query", Description: "The target's type: business, project, ...", Required: true, Schema: docs.IdSchema},
					{Name: "target_id", In: "query", Description: "The target's ID", Required: true, Schema: docs.IdSchema},
				},
			}
		},
		Handler: getWebhooks,
	}.Route(r)

	uapi.Route{
		Method:  uapi.POST,
		Pattern: "/webhooks",
		OpId:    "create_webhook",
		Auth:    []uapi.AuthType{{Type: "User"}},
		Docs: func() *docs.Doc {
			return &docs.Doc{
				Summary:     "Create Webhook",
				Description: "Registers a webhook against a target. Requires owning that target (or a staff session). The response is the only time the signing secret is ever shown — save it.",
				Req:         types.WebhookCreate{},
				Resp:        types.WebhookRevealed{},
			}
		},
		Handler: createWebhook,
	}.Route(r)

	uapi.Route{
		Method:  uapi.PATCH,
		Pattern: "/webhooks/{id}",
		OpId:    "update_webhook",
		Auth:    []uapi.AuthType{{Type: "User"}},
		Docs: func() *docs.Doc {
			return &docs.Doc{
				Summary:     "Update Webhook",
				Description: "Updates a webhook's URL, event subscriptions, or enabled state. Requires owning its target (or a staff session).",
				Req:         types.WebhookUpdate{},
				Resp:        types.Webhook{},
				Params: []docs.Parameter{
					{Name: "id", In: "path", Description: "The webhook's ID", Required: true, Schema: docs.IdSchema},
				},
			}
		},
		Handler: updateWebhook,
	}.Route(r)

	uapi.Route{
		Method:  uapi.DELETE,
		Pattern: "/webhooks/{id}",
		OpId:    "delete_webhook",
		Auth:    []uapi.AuthType{{Type: "User"}},
		Docs: func() *docs.Doc {
			return &docs.Doc{
				Summary:     "Delete Webhook",
				Description: "Unregisters a webhook. Requires owning its target (or a staff session).",
				Resp:        types.ApiError{},
				Params: []docs.Parameter{
					{Name: "id", In: "path", Description: "The webhook's ID", Required: true, Schema: docs.IdSchema},
				},
			}
		},
		Handler: deleteWebhook,
	}.Route(r)

	uapi.Route{
		Method:  uapi.POST,
		Pattern: "/webhooks/{id}/rotate-secret",
		OpId:    "rotate_webhook_secret",
		Auth:    []uapi.AuthType{{Type: "User"}},
		Docs: func() *docs.Doc {
			return &docs.Doc{
				Summary:     "Rotate Webhook Secret",
				Description: "Replaces a webhook's signing secret, invalidating the old one immediately. Requires owning its target (or a staff session). The response is the only time the new secret is ever shown.",
				Resp:        types.WebhookRevealed{},
				Params: []docs.Parameter{
					{Name: "id", In: "path", Description: "The webhook's ID", Required: true, Schema: docs.IdSchema},
				},
			}
		},
		Handler: rotateWebhookSecret,
	}.Route(r)

	uapi.Route{
		Method:  uapi.POST,
		Pattern: "/webhooks/{id}/test",
		OpId:    "test_webhook",
		Auth:    []uapi.AuthType{{Type: "User"}},
		Docs: func() *docs.Doc {
			return &docs.Doc{
				Summary:     "Test Webhook",
				Description: "Sends a signed test delivery to a webhook immediately (bypassing its event subscription filter) and reports whether it succeeded. Requires owning its target (or a staff session).",
				Resp:        types.ApiError{},
				Params: []docs.Parameter{
					{Name: "id", In: "path", Description: "The webhook's ID", Required: true, Schema: docs.IdSchema},
				},
			}
		},
		Handler: testWebhook,
	}.Route(r)
}
