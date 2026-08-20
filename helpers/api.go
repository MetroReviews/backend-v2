package helpers

import (
	"net/http"
	"strconv"

	"github.com/MetroReviews/backend-v2/types"
	"github.com/infinitybotlist/eureka/uapi"
)

func InternalError(err error) uapi.HttpResponse {
	return ErrorResponse(http.StatusInternalServerError, err.Error())
}

func ErrorResponse(status int, msg string) uapi.HttpResponse {
	return uapi.HttpResponse{Status: status, Json: types.ApiError{Message: msg, Error: true}}
}

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
