package auth

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/MetroReviews/backend-v2/helpers"
	"github.com/MetroReviews/backend-v2/identity"
	"github.com/infinitybotlist/eureka/uapi"
)

type registerRequest struct {
	Email    string  `json:"email" validate:"required,email" msg:"A valid email is required" description:"The account's email address"`
	Password string  `json:"password" validate:"required,min=8,max=72" msg:"A password of 8-72 characters is required" description:"The account's password"`
	Username *string `json:"username" description:"An optional display username; defaults to the email's local part"`
}

func register(d uapi.RouteData, r *http.Request) uapi.HttpResponse {
	if resp := helpers.RateLimit(r, "auth-register", 5, time.Hour); resp != nil {
		return *resp
	}

	var payload registerRequest
	if hresp, ok := uapi.MarshalReq(r, &payload); !ok {
		return hresp
	}

	passwordHash, errResp := hashPassword(payload.Password)
	if errResp != nil {
		return *errResp
	}

	username, _, _ := strings.Cut(identity.NormalizeEmail(payload.Email), "@")
	if payload.Username != nil && *payload.Username != "" {
		username = *payload.Username
	}

	userID, err := identity.CreateLocalUser(d.Context, payload.Email, passwordHash, username)
	if errors.Is(err, identity.ErrEmailTaken) {
		return helpers.ErrorResponse(http.StatusConflict, "Email is already registered")
	}
	if err != nil {
		return helpers.InternalError(err)
	}

	sessionToken, expiresAt, err := identity.NewSession(d.Context, userID)
	if err != nil {
		return helpers.InternalError(err)
	}

	return uapi.HttpResponse{Json: localAuthResponse{
		SessionToken: sessionToken,
		ExpiresAt:    expiresAt,
		UserID:       userID,
		Email:        identity.NormalizeEmail(payload.Email),
		Username:     &username,
	}}
}
