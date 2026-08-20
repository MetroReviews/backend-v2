package reviews

import (
	"net/http"

	"github.com/MetroReviews/backend-v2/types"
	"github.com/go-chi/chi/v5"
	docs "github.com/infinitybotlist/eureka/doclib"
	"github.com/infinitybotlist/eureka/uapi"
)

func registerModerationRoutes(r *chi.Mux) {
	uapi.Route{
		Method:  uapi.POST,
		Pattern: "/reviews/{id}/vote",
		OpId:    "vote_review",
		Auth:    []uapi.AuthType{{Type: "User"}},
		Docs: func() *docs.Doc {
			return &docs.Doc{
				Summary:     "Vote Review",
				Description: "Marks a review as helpful or unhelpful. Requires a logged-in user session; one vote per user per review (later votes overwrite earlier ones).",
				Req:         types.ReviewVote{},
				Resp:        types.ApiError{},
				Params: []docs.Parameter{
					{Name: "id", In: "path", Description: "The review's ID", Required: true, Schema: docs.IdSchema},
				},
			}
		},
		Handler: voteReview,
	}.Route(r)

	uapi.Route{
		Method:  uapi.POST,
		Pattern: "/reviews/{id}/response",
		OpId:    "respond_to_review",
		Auth:    []uapi.AuthType{{Type: "User"}},
		Docs: func() *docs.Doc {
			return &docs.Doc{
				Summary:     "Respond To Review",
				Description: "Posts (or replaces) the owner's reply to a review. Requires the reviewed business/project's verified owner or a staff session.",
				Req:         types.ReviewResponse{},
				Resp:        types.Review{},
				Params: []docs.Parameter{
					{Name: "id", In: "path", Description: "The review's ID", Required: true, Schema: docs.IdSchema},
				},
			}
		},
		Handler: respondToReview,
	}.Route(r)

	uapi.Route{
		Method:  uapi.POST,
		Pattern: "/reviews/{id}/report",
		OpId:    "report_review",
		Auth:    []uapi.AuthType{{Type: "User"}},
		Docs: func() *docs.Doc {
			return &docs.Doc{
				Summary:     "Report Review",
				Description: "Flags a review for staff attention. Requires a logged-in user session.",
				Req:         types.ReportCreate{},
				Resp:        types.Report{},
				Params: []docs.Parameter{
					{Name: "id", In: "path", Description: "The review's ID", Required: true, Schema: docs.IdSchema},
				},
			}
		},
		Handler: func(d uapi.RouteData, r *http.Request) uapi.HttpResponse {
			return postReport(d, r, "review")
		},
	}.Route(r)
}
