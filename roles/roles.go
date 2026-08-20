package roles

import (
	"github.com/MetroReviews/backend-v2/types"
	"github.com/jackc/pgx/v5"
)

const roleColumns = `id, name, discord_role_id, permissions, created_at, updated_at`

func scanRole(row pgx.Row) (*types.Role, error) {
	var role types.Role
	if err := row.Scan(&role.ID, &role.Name, &role.DiscordRoleID, &role.Permissions, &role.CreatedAt, &role.UpdatedAt); err != nil {
		return nil, err
	}
	return &role, nil
}
