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
	Unauthorized:        `{"message":"Unauthorized. Provide a valid session token","error":true}`,
	InternalServerError: `{"message":"Internal server error","error":true}`,
	MethodNotAllowed:    `{"message":"Method not allowed","error":true}`,
	BodyRequired:        `{"message":"A request body is required","error":true}`,
}

func Setup() {
	uapi.SetupState(uapi.UAPIState{
		Logger:           state.Logger,
		Authorize:        authorize,
		AuthTypeMap:      map[string]string{"User": "User", "Staff": "Staff"},
		DefaultResponder: defaultResponder{},
		Constants:        Constants,
		Context:          context.Background(),
	})

	docs.DocsSetupData = &docs.SetupData{
		URL:         "https://" + state.Config.Auth.AllowedHost,
		ErrorStruct: types.ApiError{},
		Info: docs.Info{
			Title:       "Metro Reviews API",
			Description: "The API powering Metro Reviews: reviews and ratings for any showcased business, company or organization and its products or menu.",
			Version:     "7.0",
			Contact: docs.Contact{
				Name: "Purrquinox",
				URL:  "https://purrquinox.com",
			},
			License: docs.License{
				Name: "MIT",
				URL:  "https://github.com/MetroReviews/backend-v2/blob/main/LICENSE",
			},
		},
	}

	docs.Setup()
	docs.AddSecuritySchema("User", "Authorization", "A user's session token, as `Bearer <token>`.")
	docs.AddSecuritySchema("Staff", "Authorization", "A staff member's session token, as `Bearer <token>`.")
}

func authorize(r uapi.Route, req *http.Request) (uapi.AuthData, uapi.HttpResponse, bool) {
	return uapi.AuthData{Authorized: true}, uapi.HttpResponse{}, true
}
