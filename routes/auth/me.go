package auth

import (
	"net/http"

	"github.com/MetroReviews/backend-v2/api"
	"github.com/infinitybotlist/eureka/uapi"
)

func getMe(d uapi.RouteData, r *http.Request) uapi.HttpResponse {
	u, resp := api.AuthUser(d.Context, r)
	if resp != nil {
		return *resp
	}
	return uapi.HttpResponse{Json: u}
}
