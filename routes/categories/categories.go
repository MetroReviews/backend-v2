package categories

import (
	"net/http"

	"github.com/MetroReviews/backend-v2/helpers"
	"github.com/MetroReviews/backend-v2/state"
	"github.com/MetroReviews/backend-v2/types"
	"github.com/go-chi/chi/v5"
	docs "github.com/infinitybotlist/eureka/doclib"
	"github.com/infinitybotlist/eureka/uapi"
	"github.com/jackc/pgx/v5"
)

const tagName = "Categories"

type Router struct{}

func (Router) Tag() (string, string) {
	return tagName, "Business categories."
}

func (Router) Routes(r *chi.Mux) {
	uapi.Route{
		Method:  uapi.GET,
		Pattern: "/categories",
		OpId:    "get_categories",
		Docs: func() *docs.Doc {
			return &docs.Doc{
				Summary:     "Get Categories",
				Description: "Gets every business category.",
				Resp:        []types.Category{},
				RespName:    "CategoryArray",
			}
		},
		Handler: getCategories,
	}.Route(r)
}

func getCategories(d uapi.RouteData, r *http.Request) uapi.HttpResponse {
	rows, err := state.Pool.Query(d.Context, "SELECT id, slug, name, description, icon FROM categories ORDER BY name ASC")
	if err != nil {
		return helpers.InternalError(err)
	}

	list, err := pgx.CollectRows(rows, pgx.RowToStructByName[types.Category])
	if err != nil {
		return helpers.InternalError(err)
	}

	return uapi.HttpResponse{Json: list}
}
