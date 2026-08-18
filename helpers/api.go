package helpers

import (
	"net/http"
	"strconv"

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

// Pagination reads the "offset"/"limit" query params shared by every
// paginated list endpoint, falling back to defaultLimit and capping at
// maxLimit. Invalid values are ignored rather than rejected, same as each
// call site did before this was unified.
func Pagination(r *http.Request, defaultLimit, maxLimit int64) (limit, offset int64) {
	limit, offset = defaultLimit, 0
	if v := r.URL.Query().Get("offset"); v != "" {
		if p, err := strconv.ParseInt(v, 10, 64); err == nil {
			offset = p
		}
	}
	if v := r.URL.Query().Get("limit"); v != "" {
		if p, err := strconv.ParseInt(v, 10, 64); err == nil {
			limit = p
		}
	}
	if limit > maxLimit {
		limit = maxLimit
	}
	return
}
