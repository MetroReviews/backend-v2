// Package panel serves internal endpoints used by the Metro Reviews panel
// (Discord OAuth2 login + access checks). Handlers are split one concern
// per file; this file only wires up routing and shared response shapes.
package panel

import (
	"github.com/MetroReviews/backend-v2/state"
	"github.com/MetroReviews/backend-v2/types"
	"github.com/go-chi/chi/v5"
	docs "github.com/infinitybotlist/eureka/doclib"
	"github.com/infinitybotlist/eureka/uapi"
)

const tagName = "Panel (Internal)"

type Router struct{}

func (Router) Tag() (string, string) {
	return tagName, "Internal endpoints used by the Metro Reviews panel."
}

type oauthURLResponse struct {
	URL string `json:"url" description:"The Discord OAuth2 authorize URL"`
}

type panelMember struct {
	ID   string `json:"id" description:"The member's Discord ID"`
	Name string `json:"name" description:"The member's username"`
}

type panelAccessResponse struct {
	Access bool         `json:"access" description:"Whether panel access is granted"`
	Hint   string       `json:"hint,omitempty" description:"A hint about why access was denied"`
	Member *panelMember `json:"member,omitempty" description:"The authenticated member"`
}

func (Router) Routes(r *chi.Mux) {
	uapi.Route{
		Method:  uapi.GET,
		Pattern: "/_panel/strikestone",
		OpId:    "get_oauth2",
		Docs: func() *docs.Doc {
			return &docs.Doc{
				Summary:     "Get OAuth2 URL",
				Description: "Returns the Discord OAuth2 authorize URL for the review panel.",
				Resp:        oauthURLResponse{},
			}
		},
		Handler: getOAuth2,
	}.Route(r)

	uapi.Route{
		Method:  uapi.GET,
		Pattern: "/_panel/mapleshade",
		OpId:    "get_panel_access",
		Docs: func() *docs.Doc {
			return &docs.Doc{
				Summary:     "Get Panel Access",
				Description: "Verifies a panel access ticket and returns whether the holder may access the panel.",
				Resp:        panelAccessResponse{},
				Params: []docs.Parameter{
					{Name: "ticket", In: "query", Description: "The base64url-encoded ticket", Required: true, Schema: docs.IdSchema},
				},
			}
		},
		Handler: getPanelAccess,
	}.Route(r)

	uapi.Route{
		Method:  uapi.GET,
		Pattern: "/_panel/frostpaw",
		OpId:    "complete_oauth2",
		Docs: func() *docs.Doc {
			return &docs.Doc{
				Summary:     "Complete OAuth2",
				Description: "Completes the Discord OAuth2 handshake and redirects back to the panel with a ticket.",
				Resp:        types.ApiError{},
				Params: []docs.Parameter{
					{Name: "code", In: "query", Description: "The OAuth2 authorization code", Required: true, Schema: docs.IdSchema},
					{Name: "state", In: "query", Description: "The origin to redirect back to", Required: true, Schema: docs.IdSchema},
				},
			}
		},
		Handler: completeOAuth2,
	}.Route(r)
}

func appID() string {
	if state.Discord != nil && state.Discord.State != nil && state.Discord.State.User != nil {
		return state.Discord.State.User.ID
	}
	return ""
}
