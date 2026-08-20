package cache

import (
	"context"
	"time"

	"github.com/MetroReviews/backend-v2/state"
	"github.com/infinitybotlist/eureka/jsonimpl"
)

func Get[T any](ctx context.Context, key string) (value T, ok bool) {
	if state.Cache == nil {
		return value, false
	}

	raw, err := state.Cache.Get(ctx, key)
	if err != nil {

		return value, false
	}

	if err := jsonimpl.Unmarshal([]byte(*raw), &value); err != nil {
		return value, false
	}

	return value, true
}

func Set(ctx context.Context, key string, value any, ttl time.Duration) error {
	if state.Cache == nil {
		return nil
	}

	encoded, err := jsonimpl.Marshal(value)
	if err != nil {
		return err
	}
	s := string(encoded)
	return state.Cache.Set(ctx, key, &s, ttl)
}

func Del(ctx context.Context, key string) error {
	if state.Cache == nil {
		return nil
	}

	return state.Cache.Delete(ctx, key)
}
