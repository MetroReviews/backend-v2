package helpers

import (
	"net/http"

	"github.com/MetroReviews/backend-v2/types"
	"github.com/infinitybotlist/eureka/uapi"
)

// InternalError wraps err in the standard 500 ApiError JSON body used by
// every route handler on a failed DB call or similar.
func InternalError(err error) uapi.HttpResponse {
	return ErrorResponse(http.StatusInternalServerError, err.Error())
}

// ErrorResponse builds the standard ApiError JSON body for a given status
// and message.
func ErrorResponse(status int, msg string) uapi.HttpResponse {
	return uapi.HttpResponse{Status: status, Json: types.ApiError{Message: msg, Error: true}}
}
