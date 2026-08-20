package webhooks

import (
	"context"
	"errors"

	"github.com/MetroReviews/backend-v2/state"
	"github.com/MetroReviews/backend-v2/types"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

func OwnsTarget(ctx context.Context, user *types.User, targetType, targetID string) (bool, error) {
	switch targetType {
	case "business":
		return ownsBusiness(ctx, user, targetID)
	case "project":
		return ownsProject(ctx, user, targetID)
	default:
		return false, nil
	}
}

func ownsBusiness(ctx context.Context, user *types.User, targetID string) (bool, error) {
	id, err := uuid.Parse(targetID)
	if err != nil {
		return false, nil
	}
	var ownerID *uuid.UUID
	err = state.Pool.QueryRow(ctx, "SELECT owner_id FROM businesses WHERE id = $1", id).Scan(&ownerID)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return ownerID != nil && *ownerID == user.ID, nil
}

func ownsProject(ctx context.Context, user *types.User, targetID string) (bool, error) {
	id, err := uuid.Parse(targetID)
	if err != nil {
		return false, nil
	}
	var ownerID *uuid.UUID
	err = state.Pool.QueryRow(ctx, `
		SELECT b.owner_id FROM projects p JOIN businesses b ON b.id = p.business_id WHERE p.id = $1`, id,
	).Scan(&ownerID)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return ownerID != nil && *ownerID == user.ID, nil
}
