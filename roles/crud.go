package roles

import (
	"context"
	"fmt"
	"strings"

	"github.com/MetroReviews/backend-v2/state"
	"github.com/MetroReviews/backend-v2/types"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

func List(ctx context.Context) ([]types.Role, error) {
	rows, err := state.Pool.Query(ctx, "SELECT "+roleColumns+" FROM roles ORDER BY name")
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, pgx.RowToStructByName[types.Role])
}

func Get(ctx context.Context, id uuid.UUID) (*types.Role, error) {
	return scanRole(state.Pool.QueryRow(ctx, "SELECT "+roleColumns+" FROM roles WHERE id = $1", id))
}

func Create(ctx context.Context, name string, discordRoleID *int64, permissions []string) (*types.Role, error) {
	if permissions == nil {
		permissions = []string{}
	}
	return scanRole(state.Pool.QueryRow(ctx, `
		INSERT INTO roles (name, discord_role_id, permissions) VALUES ($1, $2, $3)
		RETURNING `+roleColumns,
		name, discordRoleID, permissions,
	))
}

func Update(ctx context.Context, id uuid.UUID, name *string, discordRoleID *int64, unlinkDiscordRole bool, permissions []string) (*types.Role, error) {
	var setClauses []string
	var args []any

	set := func(column string, value any) {
		args = append(args, value)
		setClauses = append(setClauses, fmt.Sprintf("%s = $%d", column, len(args)))
	}

	if name != nil {
		set("name", *name)
	}
	if unlinkDiscordRole {
		setClauses = append(setClauses, "discord_role_id = NULL")
	} else if discordRoleID != nil {
		set("discord_role_id", *discordRoleID)
	}
	if permissions != nil {
		set("permissions", permissions)
	}

	if len(setClauses) == 0 {
		return Get(ctx, id)
	}
	setClauses = append(setClauses, "updated_at = NOW()")

	args = append(args, id)
	query := fmt.Sprintf("UPDATE roles SET %s WHERE id = $%d RETURNING %s",
		strings.Join(setClauses, ", "), len(args), roleColumns)
	return scanRole(state.Pool.QueryRow(ctx, query, args...))
}

func Delete(ctx context.Context, id uuid.UUID) error {
	_, err := state.Pool.Exec(ctx, "DELETE FROM roles WHERE id = $1", id)
	return err
}
