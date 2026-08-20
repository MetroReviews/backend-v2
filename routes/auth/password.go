package auth

import (
	"errors"
	"net/http"

	"github.com/MetroReviews/backend-v2/api"
	"github.com/MetroReviews/backend-v2/helpers"
	"github.com/MetroReviews/backend-v2/identity"
	"github.com/MetroReviews/backend-v2/types"
	"github.com/infinitybotlist/eureka/uapi"
)

type setPasswordRequest struct {
	Email    string `json:"email" validate:"required,email" msg:"A valid email is required" description:"The email address to log in with"`
	Password string `json:"password" validate:"required,min=8,max=72" msg:"A password of 8-72 characters is required" description:"The new password"`
}

func setPassword(d uapi.RouteData, r *http.Request) uapi.HttpResponse {
	u, resp := api.AuthUser(d.Context, r)
	if resp != nil {
		return *resp
	}

	var payload setPasswordRequest
	if hresp, ok := uapi.MarshalReq(r, &payload); !ok {
		return hresp
	}

	passwordHash, errResp := hashPassword(payload.Password)
	if errResp != nil {
		return *errResp
	}

	if err := identity.SetLocalAccount(d.Context, u.ID, payload.Email, passwordHash); err != nil {
		if errors.Is(err, identity.ErrEmailTaken) {
			return helpers.ErrorResponse(http.StatusConflict, "Email is already registered")
		}
		return helpers.InternalError(err)
	}

	return uapi.HttpResponse{Json: types.ApiError{Message: "Password set", Error: false}}
}
