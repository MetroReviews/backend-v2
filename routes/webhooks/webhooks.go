// Package webhooks exposes the /webhooks endpoints: registering, managing
// and testing outbound event subscriptions against any target (a bot, a
// business, a project, or whatever's added next — see the webhooks
// package this wraps). Handlers are split one concern per file; this file
// just wires routing plus the shared authorization helpers every handler
// goes through.
package webhooks

import (
	"errors"
	"net/http"

	"github.com/MetroReviews/backend-v2/api"
	"github.com/MetroReviews/backend-v2/helpers"
	"github.com/MetroReviews/backend-v2/types"
	svc "github.com/MetroReviews/backend-v2/webhooks"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/infinitybotlist/eureka/uapi"
	"github.com/jackc/pgx/v5"
)

const tagName = "Webhooks"

type Router struct{}

func (Router) Tag() (string, string) {
	return tagName, "Outbound event subscriptions, registered against a bot, a business, a project, or whatever's added next."
}

func (Router) Routes(r *chi.Mux) {
	registerRoutes(r)
}

// authorizeTarget is the shared gate for creating/listing webhooks against
// a target the caller names directly (as opposed to one already loaded
// from an existing webhook row — see authorizeWebhook): a logged-in user
// who either owns targetType/targetID (see svc.OwnsTarget) or is staff.
func authorizeTarget(d uapi.RouteData, r *http.Request, targetType, targetID string) (*types.User, *uapi.HttpResponse) {
	user, resp := api.AuthUser(d.Context, r)
	if resp != nil {
		return nil, resp
	}
	if user.IsStaff {
		return user, nil
	}
	owns, err := svc.OwnsTarget(d.Context, user, targetType, targetID)
	if err != nil {
		errResp := helpers.InternalError(err)
		return nil, &errResp
	}
	if !owns {
		errResp := helpers.ErrorResponse(http.StatusForbidden, "You do not own this "+targetType)
		return nil, &errResp
	}
	return user, nil
}

// authorizeWebhook is authorizeTarget for routes keyed by an existing
// webhook's {id} — loads it once and checks ownership of *its* target, so
// callers get both the authenticated user and the webhook without a
// second query.
func authorizeWebhook(d uapi.RouteData, r *http.Request) (*types.User, *types.Webhook, *uapi.HttpResponse) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		resp := uapi.DefaultResponse(http.StatusBadRequest)
		return nil, nil, &resp
	}

	user, resp := api.AuthUser(d.Context, r)
	if resp != nil {
		return nil, nil, resp
	}

	hook, err := svc.Get(d.Context, id)
	if errors.Is(err, pgx.ErrNoRows) {
		resp := uapi.DefaultResponse(http.StatusNotFound)
		return nil, nil, &resp
	}
	if err != nil {
		errResp := helpers.InternalError(err)
		return nil, nil, &errResp
	}

	if !user.IsStaff {
		owns, err := svc.OwnsTarget(d.Context, user, hook.TargetType, hook.TargetID)
		if err != nil {
			errResp := helpers.InternalError(err)
			return nil, nil, &errResp
		}
		if !owns {
			errResp := helpers.ErrorResponse(http.StatusForbidden, "You do not own this webhook's target")
			return nil, nil, &errResp
		}
	}

	return user, hook, nil
}

// invalidEvents returns every entry in requested that isn't in svc's
// catalog, for rejecting typos up front.
func invalidEvents(requested []string) []string {
	var bad []string
	for _, e := range requested {
		if !svc.ValidEvent(e) {
			bad = append(bad, e)
		}
	}
	return bad
}
