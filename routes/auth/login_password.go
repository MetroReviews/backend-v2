package auth

import (
	"net/http"
	"time"

	"github.com/MetroReviews/backend-v2/helpers"
	"github.com/MetroReviews/backend-v2/identity"
	"github.com/MetroReviews/backend-v2/state"
	"github.com/infinitybotlist/eureka/uapi"
	"golang.org/x/crypto/bcrypt"
)

type loginPasswordRequest struct {
	Email    string `json:"email" validate:"required,email" msg:"A valid email is required" description:"The account's email address"`
	Password string `json:"password" validate:"required" msg:"A password is required" description:"The account's password"`
}

var dummyPasswordHash = mustHashDummyPassword()

func mustHashDummyPassword() []byte {
	hash, err := bcrypt.GenerateFromPassword([]byte("timing-safety-dummy-password"), bcrypt.DefaultCost)
	if err != nil {

		panic(err)
	}
	return hash
}

func loginPassword(d uapi.RouteData, r *http.Request) uapi.HttpResponse {

	if resp := helpers.RateLimit(r, "auth-login-password", 10, 15*time.Minute); resp != nil {
		return *resp
	}

	var payload loginPasswordRequest
	if hresp, ok := uapi.MarshalReq(r, &payload); !ok {
		return hresp
	}

	userID, passwordHash, found, err := identity.LookupLocalAccount(d.Context, payload.Email)
	if err != nil {
		return helpers.InternalError(err)
	}

	invalidCreds := helpers.ErrorResponse(http.StatusUnauthorized, "Invalid email or password")
	if !found {
		bcrypt.CompareHashAndPassword(dummyPasswordHash, []byte(payload.Password))
		return invalidCreds
	}
	if len(payload.Password) > maxPasswordBytes {
		return invalidCreds
	}
	if bcrypt.CompareHashAndPassword([]byte(passwordHash), []byte(payload.Password)) != nil {
		return invalidCreds
	}

	var username *string
	if err := state.Pool.QueryRow(d.Context,
		"SELECT username FROM users WHERE id = $1", userID,
	).Scan(&username); err != nil {
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
		Username:     username,
	}}
}
