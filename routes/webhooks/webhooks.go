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
	return tagName, "Outbound event subscriptions, registered against a business, a project, or whatever's added next."
}

func (Router) Routes(r *chi.Mux) {
	registerRoutes(r)
}

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

func invalidEvents(requested []string) []string {
	var bad []string
	for _, e := range requested {
		if !svc.ValidEvent(e) {
			bad = append(bad, e)
		}
	}
	return bad
}
