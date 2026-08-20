package helpers

import (
	"net/http"
	"time"

	"github.com/infinitybotlist/eureka/ratelimit"
	"github.com/infinitybotlist/eureka/uapi"
)

func RateLimit(r *http.Request, bucket string, max int, window time.Duration) *uapi.HttpResponse {
	if ratelimit.State == nil {
		return nil
	}

	limit, err := (ratelimit.Ratelimit{Bucket: bucket, MaxRequests: max, Expiry: window}).Limit(r.Context(), r)
	if err != nil || !limit.Exceeded {
		return nil
	}

	resp := ErrorResponse(http.StatusTooManyRequests, "Rate limit exceeded, try again later")
	resp.Headers = limit.Headers()
	return &resp
}
