package identity

import (
	"context"
	"time"

	"github.com/MetroReviews/backend-v2/state"
	"github.com/google/uuid"
	"github.com/infinitybotlist/eureka/crypto"
)

const SessionTTL = 30 * 24 * time.Hour

func NewSession(ctx context.Context, userID uuid.UUID) (token string, expiresAt time.Time, err error) {
	token = crypto.RandString(64)
	expiresAt = time.Now().Add(SessionTTL)
	_, err = state.Pool.Exec(ctx,
		"INSERT INTO sessions (token, user_id, expires_at) VALUES ($1, $2, $3)",
		token, userID, expiresAt,
	)
	return token, expiresAt, err
}
