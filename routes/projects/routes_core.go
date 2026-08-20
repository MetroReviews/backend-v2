package projects

import (
	"github.com/MetroReviews/backend-v2/types"
	"github.com/go-chi/chi/v5"
	docs "github.com/infinitybotlist/eureka/doclib"
	"github.com/infinitybotlist/eureka/uapi"
)

func registerCoreRoutes(r *chi.Mux) {
	uapi.Route{
		Method:  uapi.GET,
		Pattern: "/businesses/{business_id}/projects",
		OpId:    "get_business_projects",
		Docs: func() *docs.Doc {
			return &docs.Doc{
				Summary:     "Get Business Projects",
				Description: "Gets a business's posted projects. Supports `sort` (`new` (default), `rating`, `reviews`). Anonymous callers only see approved projects; staff also see pending/under-review/denied/suspended ones by passing `all=true`.",
				Resp:        []types.Project{},
				RespName:    "ProjectArray",
				Params: []docs.Parameter{
					{Name: "business_id", In: "path", Description: "The business's ID", Required: true, Schema: docs.IdSchema},
					{Name: "sort", In: "query", Description: "new (default), rating, or reviews", Required: false, Schema: docs.IdSchema},
					{Name: "all", In: "query", Description: "Staff only: include non-approved projects", Required: false, Schema: docs.IdSchema},
				},
			}
		},
		Handler: getBusinessProjects,
	}.Route(r)

	uapi.Route{
		Method:  uapi.GET,
		Pattern: "/projects/{id}",
		OpId:    "get_project",
		Docs: func() *docs.Doc {
			return &docs.Doc{
				Summary:     "Get Project",
				Description: "Gets a single project by its ID.",
				Resp:        types.Project{},
				Params: []docs.Parameter{
					{Name: "id", In: "path", Description: "The project's ID", Required: true, Schema: docs.IdSchema},
				},
			}
		},
		Handler: getProject,
	}.Route(r)

	uapi.Route{
		Method:  uapi.POST,
		Pattern: "/businesses/{business_id}/projects",
		OpId:    "post_project",
		Auth:    []uapi.AuthType{{Type: "User"}},
		Docs: func() *docs.Doc {
			return &docs.Doc{
				Summary:     "Add Project",
				Description: "Posts a project for a business. Requires the business's verified owner or a staff session. Goes through the same staff claim/approve/deny queue as a new business (see /projects/{id}/review/*) before it's publicly visible.",
				Req:         types.ProjectCreate{},
				Resp:        types.Project{},
				Params: []docs.Parameter{
					{Name: "business_id", In: "path", Description: "The business's ID", Required: true, Schema: docs.IdSchema},
				},
			}
		},
		Handler: postProject,
	}.Route(r)

	uapi.Route{
		Method:  uapi.PATCH,
		Pattern: "/projects/{id}",
		OpId:    "update_project",
		Auth:    []uapi.AuthType{{Type: "User"}},
		Docs: func() *docs.Doc {
			return &docs.Doc{
				Summary:     "Update Project",
				Description: "Updates a project's fields. Requires the owning business's verified owner or a staff session.",
				Req:         types.ProjectUpdate{},
				Resp:        types.UpdatedResponse{},
				Params: []docs.Parameter{
					{Name: "id", In: "path", Description: "The project's ID", Required: true, Schema: docs.IdSchema},
				},
			}
		},
		Handler: updateProject,
	}.Route(r)
}
