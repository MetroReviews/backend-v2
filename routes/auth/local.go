package auth

import (
	"net/http"
	"time"

	"github.com/MetroReviews/backend-v2/helpers"
	"github.com/google/uuid"
	"github.com/infinitybotlist/eureka/uapi"
	"golang.org/x/crypto/bcrypt"
)

const maxPasswordBytes = 72

type localAuthResponse struct {
	SessionToken string    `json:"session_token" description:"The Metro session token to send as 'Authorization: Bearer <token>'"`
	ExpiresAt    time.Time `json:"expires_at" description:"When the session token expires"`
	UserID       uuid.UUID `json:"user_id" description:"The authenticated user's Metro ID"`
	Email        string    `json:"email" description:"The account's email address"`
	Username     *string   `json:"username" description:"The account's display username, if set"`
}

func hashPassword(password string) (string, *uapi.HttpResponse) {
	if len(password) > maxPasswordBytes {
		resp := helpers.ErrorResponse(http.StatusBadRequest, "Password is too long")
		return "", &resp
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		resp := helpers.InternalError(err)
		return "", &resp
	}
	return string(hash), nil
}
