package api

import (
	"context"
	"net/http"

	"github.com/MetroReviews/backend-v2/state"
	"github.com/MetroReviews/backend-v2/types"
	docs "github.com/infinitybotlist/eureka/doclib"
	"github.com/infinitybotlist/eureka/uapi"
)

type defaultResponder struct{}

func (defaultResponder) New(msg string, ctx map[string]string) any {
	return types.ApiError{Message: msg, Error: true, Context: ctx}
}

var Constants = &uapi.UAPIConstants{
	ResourceNotFound:    `{"message":"Resource not found","error":true}`,
	BadRequest:          `{"message":"Bad request","error":true}`,
	Forbidden:           `{"message":"Forbidden","error":true}`,
	Unauthorized:        `{"message":"Unauthorized. Provide a valid list secret key","error":true}`,
	InternalServerError: `{"message":"Internal server error","error":true}`,
	MethodNotAllowed:    `{"message":"Method not allowed","error":true}`,
	BodyRequired:        `{"message":"A request body is required","error":true}`,
}

func Setup() {
	uapi.SetupState(uapi.UAPIState{
		Logger:           state.Logger,
		Authorize:        authorize,
		AuthTypeMap:      map[string]string{"List": "List"},
		DefaultResponder: defaultResponder{},
		Constants:        Constants,
		Context:          context.Background(),
	})

	docs.DocsSetupData = &docs.SetupData{
		URL:         "https://" + state.Config.AllowedHost,
		ErrorStruct: types.ApiError{},
		Info: docs.Info{
			Title:       "Metro Reviews API",
			Description: "The API powering Metro Reviews: a central bot review service that propagates review actions across enrolled bot lists.",
			Version:     "6.0",
			Contact: docs.Contact{
				Name: "Purrquinox",
				URL:  "https://purrquinox.com",
			},
			License: docs.License{
				Name: "AGPL-3.0",
				URL:  "https://github.com/MetroReviews/backend-v2/blob/main/LICENSE",
			},
		},
	}

	docs.Setup()
	docs.AddSecuritySchema("List", "Authorization", "A list's secret key. Sent in the Authorization header alongside the list_id.")
}

func authorize(r uapi.Route, req *http.Request) (uapi.AuthData, uapi.HttpResponse, bool) {
	return uapi.AuthData{Authorized: true}, uapi.HttpResponse{}, true
}
